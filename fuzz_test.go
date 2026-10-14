// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"encoding/binary"
	"image/color"
	"math"
	"testing"
	"unicode"
	"unicode/utf8"

	nt3dmath "nimbleterminal.dev/nt3d/math"
)

// fuzzFloats decodes little-endian float32s from raw fuzz input.
func fuzzFloats(b []byte) []float32 {
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

func FuzzNumericTicks(f *testing.F) {
	f.Add(float32(0), float32(1), 5)
	f.Add(float32(-3.7), float32(1e7), 0)
	f.Add(float32(2), float32(2), 64)
	f.Add(float32(1e-30), float32(2e-30), 3)
	f.Add(float32(-math.MaxFloat32), float32(math.MaxFloat32), 2)
	f.Fuzz(func(t *testing.T, lo, hi float32, count int) {
		ticks := numericTicks(lo, hi, count, nil)
		if !validFloat(lo, hi) || lo > hi {
			if ticks != nil {
				t.Fatalf("invalid range [%v,%v] produced ticks %+v", lo, hi, ticks)
			}
			return
		}
		if len(ticks) == 0 || len(ticks) > 64 {
			t.Fatalf("[%v,%v] count %d: %d ticks", lo, hi, count, len(ticks))
		}
		for i, tick := range ticks {
			if tick.Value < lo || tick.Value > hi || tick.Label == "" {
				t.Fatalf("[%v,%v] bad tick %d: %+v", lo, hi, i, tick)
			}
			if i > 0 && tick.Value <= ticks[i-1].Value {
				t.Fatalf("[%v,%v] ticks not ascending: %+v", lo, hi, ticks)
			}
		}
	})
}

func FuzzDisplayText(f *testing.F) {
	f.Add("temperature", 8)
	f.Add("αβγ\x00\tx́", 4)
	f.Add("\xff\xfe", 1)
	f.Fuzz(func(t *testing.T, s string, limit int) {
		limit = limit % 256
		got := displayText(s, limit)
		if !utf8.ValidString(got) {
			t.Fatalf("invalid UTF-8 %q", got)
		}
		if n := utf8.RuneCountInString(got); n > max(0, limit) {
			t.Fatalf("%d runes exceeds limit %d", n, limit)
		}
		for _, r := range got {
			if unicode.IsControl(r) || unicode.Is(unicode.Mn, r) {
				t.Fatalf("unsanitized rune %U in %q", r, got)
			}
		}
	})
}

func FuzzPaletteAt(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8}, .5)
	f.Add([]byte{}, 0.)
	f.Add([]byte{255, 255, 255, 255}, math.NaN())
	f.Fuzz(func(t *testing.T, stops []byte, at float64) {
		p := make(Palette, len(stops)/4)
		for i := range p {
			p[i] = color.RGBA{stops[i*4], stops[i*4+1], stops[i*4+2], stops[i*4+3]}
		}
		p.At(at) // Must not panic, including for NaN and ±Inf.
	})
}

func fuzzBounds(v []float32) (nt3dmath.AABB, bool) {
	b := nt3dmath.AABB{
		Min: nt3dmath.Vec3{X: min(v[0], v[3]), Y: min(v[1], v[4]), Z: min(v[2], v[5])},
		Max: nt3dmath.Vec3{X: max(v[0], v[3]), Y: max(v[1], v[4]), Z: max(v[2], v[5])},
	}
	return b, validFloat(v[:6]...)
}

// nearlyInside allows float32 rounding at the clip plane.
func nearlyInside(b nt3dmath.AABB, p nt3dmath.Vec3) bool {
	lo, hi, q := components(b.Min), components(b.Max), components(p)
	for i := range 3 {
		tol := 1e-4 * max(1, float32(math.Abs(float64(lo[i]))), float32(math.Abs(float64(hi[i]))))
		if !(q[i] >= lo[i]-tol && q[i] <= hi[i]+tol) {
			return false
		}
	}
	return true
}

func FuzzClip(f *testing.F) {
	seed := func(v ...float32) []byte {
		b := make([]byte, 4*len(v))
		for i, x := range v {
			binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(x))
		}
		return b
	}
	f.Add(seed(-1, -1, -1, 1, 1, 1, -2, 0, 0, 2, 0, 0, 0, 3, 0))
	f.Add(seed(0, 0, 0, 1, 1, 1, .5, .5, .5, 5, 5, 5, -5, .5, .5))
	f.Fuzz(func(t *testing.T, raw []byte) {
		v := fuzzFloats(raw)
		if len(v) < 15 {
			return
		}
		b, ok := fuzzBounds(v)
		if !ok {
			return
		}
		pts := make([]nt3dmath.Vec3, 3)
		for i := range pts {
			pts[i] = vector([3]float32{v[6+i*3], v[7+i*3], v[8+i*3]})
			if !finiteVec(pts[i]) {
				return
			}
		}
		if a, z, ok := clipLine(Vertex{Position: pts[0]}, Vertex{Position: pts[1]}, b); ok {
			if !nearlyInside(b, a.Position) || !nearlyInside(b, z.Position) {
				t.Fatalf("clipLine left bounds %+v: %+v %+v", b, a.Position, z.Position)
			}
		}
		poly := clipTriangle(Vertex{Position: pts[0]}, Vertex{Position: pts[1]}, Vertex{Position: pts[2]}, b)
		if len(poly) > 9 {
			t.Fatalf("triangle clipped to %d vertices", len(poly))
		}
		for _, p := range poly {
			if !nearlyInside(b, p.Position) {
				t.Fatalf("clipTriangle left bounds %+v: %+v", b, p.Position)
			}
		}
	})
}

// FuzzModelRender drives arbitrary scatter and line data through validation,
// framing, and the software renderer.
func FuzzModelRender(f *testing.F) {
	f.Add([]byte{0, 0, 128, 63, 0, 0, 0, 64, 0, 0, 64, 64, 0, 0, 128, 64}, []byte{0, 0, 128, 63}, uint8(0))
	f.Add([]byte{0, 0, 192, 127, 0, 0, 128, 127}, []byte{}, uint8(3))
	f.Fuzz(func(t *testing.T, scatterRaw, linesRaw []byte, flags uint8) {
		s := fuzzFloats(scatterRaw)
		n := min(len(s)/4, 256)
		m := New(24, 8, WithRenderMode(Software), WithSharedColorDomain(flags&1 != 0))
		defer m.Close()
		if n > 0 {
			d := Scatter{X: s[:n], Y: s[n : 2*n], Z: s[2*n : 3*n]}
			if flags&2 != 0 {
				d.Size = s[3*n : 4*n]
			}
			m.SetScatter("s", d)
		}
		l := fuzzFloats(linesRaw)
		segs := min(len(l)/6, 256)
		if segs > 0 {
			d := Lines{Start: make([]nt3dmath.Vec3, segs), End: make([]nt3dmath.Vec3, segs), Width: float32(flags>>2) * 2}
			for i := range segs {
				d.Start[i] = vector([3]float32{l[i*6], l[i*6+1], l[i*6+2]})
				d.End[i] = vector([3]float32{l[i*6+3], l[i*6+4], l[i*6+5]})
			}
			m.SetLines("l", d)
		}
		if flags&4 != 0 {
			m.SetPlotAspect(nt3dmath.Vec3{X: 1, Y: 2, Z: .5})
		}
		frame := m.frame()
		if img := softwareRender(frame); img == nil {
			t.Fatal("nil image")
		}
	})
}
