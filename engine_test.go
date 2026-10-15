package ntcharts3d

import (
	"image"
	"image/color"
	"testing"

	"nimbleterminal.dev/nt3d"
	nt3dmath "nimbleterminal.dev/nt3d/math"
)

type engineRecorder struct {
	frames []nt3d.Frame
	closed bool
}

func TestAnisotropyReachesRendererWithoutRebuildingGeometry(t *testing.T) {
	r := &engineRecorder{}
	m := New(40, 20, WithRenderer(&gpuRenderer{engine: r}), WithAnisotropy(1))
	defer m.Close()
	m.SetSurface("terrain", Surface{NX: 2, NY: 2, Z: make([]float32, 4), Material: &Material{Texture: testTexture(t, color.RGBA{R: 255, A: 255})}})
	for _, n := range []int{1, 4, nt3d.MaxAnisotropy, 0} {
		m.SetAnisotropy(n)
		if _, mode, err := m.Snapshot(80, 60); err != nil || mode != WebGPU {
			t.Fatalf("snapshot: %v %v", mode, err)
		}
		f := r.frames[len(r.frames)-1]
		want := n
		if n == 0 {
			want = 4
		}
		if m.Anisotropy() != want || f.Anisotropy != want {
			t.Fatalf("setting %d: model=%d engine=%d, want %d", n, m.Anisotropy(), f.Anisotropy, want)
		}
		first := r.frames[0]
		if f.Revision != first.Revision || f.TextureRevision != first.TextureRevision || &f.Geometry[0].Vertices[0] != &first.Geometry[0].Vertices[0] {
			t.Fatal("filter change rebuilt geometry or changed data revisions")
		}
	}
	if got := engineFrame(Frame{}).Anisotropy; got != 4 {
		t.Fatalf("zero-value chart frame anisotropy = %d, want 4", got)
	}
	for _, tc := range []struct{ input, want int }{{-1, 1}, {0, 4}, {1, 1}, {nt3d.MaxAnisotropy + 1, nt3d.MaxAnisotropy}} {
		m.SetAnisotropy(tc.input)
		if m.Anisotropy() != tc.want {
			t.Fatalf("setting %d = %d, want %d", tc.input, m.Anisotropy(), tc.want)
		}
	}
}

func (r *engineRecorder) Render(f nt3d.Frame) (image.Image, error) {
	r.frames = append(r.frames, f)
	return image.NewRGBA(image.Rect(0, 0, f.Width, f.Height)), nil
}
func (r *engineRecorder) Close() error { r.closed = true; return nil }

func TestEngineAdapterPreservesIndependentRevisions(t *testing.T) {
	r := &engineRecorder{}
	m := New(40, 20, WithRenderer(&gpuRenderer{engine: r}))
	defer m.Close()
	first := testTexture(t, color.RGBA{R: 255, A: 255})
	next := testTexture(t, color.RGBA{G: 255, A: 255})
	m.SetSurface("terrain", Surface{NX: 3, NY: 3, Sampler: func(x, y float64) float64 { return x + y }, Material: &Material{Texture: first, Unlit: true}})
	snapshot := func() {
		t.Helper()
		if _, mode, err := m.Snapshot(100, 100); err != nil || mode != WebGPU {
			t.Fatalf("snapshot: %v %v", mode, err)
		}
	}
	snapshot()
	c := m.Camera()
	c.Beta += 30
	m.SetCamera(c)
	snapshot()
	m.SetSeriesTexture("terrain", next)
	snapshot()
	a, b, cframe := r.frames[0], r.frames[1], r.frames[2]
	if a.Revision != b.Revision || b.Revision != cframe.Revision || a.Matrix == b.Matrix {
		t.Fatal("camera/texture changed mesh revision or camera was lost")
	}
	if a.TextureRevision != b.TextureRevision || b.TextureRevision == cframe.TextureRevision {
		t.Fatal("texture revision not forwarded independently")
	}
	if &a.Geometry[0].Vertices[0] != &cframe.Geometry[0].Vertices[0] {
		t.Fatal("adapter copied mesh buffers for a texture change")
	}
	if a.Geometry[0].Material.Texture != first || cframe.Geometry[0].Material.Texture != next {
		t.Fatal("texture replacement mutated an earlier frame")
	}
	if !a.Complete || len(a.Overlays) == 0 {
		t.Fatal("snapshot completeness or chart decorations lost")
	}
	m.Close()
	if !r.closed {
		t.Fatal("engine renderer was not closed")
	}
}

// Test helpers fail immediately on invalid engine frames, rather than silently
// substituting blank images and hiding an adapter validation error.
func softwareRender(f Frame) image.Image {
	img, err := renderSoftware(f)
	if err != nil {
		panic(err)
	}
	return img
}
func wireRender(f Frame, cols, rows int) string {
	text, err := renderWireframe(f, cols, rows)
	if err != nil {
		panic(err)
	}
	return text
}

func TestLightingAdapterPreservesAmbientEndpoints(t *testing.T) {
	for _, ambient := range []float32{0, .3, 1} {
		f := Frame{Light: Light{Direction: nt3dmath.Vec3{Z: 1}, Ambient: ambient}}
		got := engineFrame(f).Lighting
		if ambient == 0 {
			if got.Ambient.Sky.R != 0 || got.Ambient.Ground.R != 0 {
				t.Fatal("zero ambient became illuminated")
			}
		} else if got.Ambient.Intensity != ambient || got.Ambient.Sky.R != 255 || got.Ambient.Ground != got.Ambient.Sky {
			t.Fatalf("ambient %g: %+v", ambient, got.Ambient)
		}
		if ambient == 1 {
			if len(got.Lights) != 0 {
				t.Fatal("fully ambient frame retained a directional light")
			}
		} else if len(got.Lights) != 1 || got.Lights[0].Intensity != 1-ambient || got.Lights[0].Direction != f.Light.Direction {
			t.Fatalf("ambient %g: %+v", ambient, got.Lights)
		}
	}
}

func TestLightingAdapterZeroDirection(t *testing.T) {
	for _, ambient := range []float32{0, .3, 1} {
		f := Frame{Width: 8, Height: 8, Matrix: nt3dmath.Identity(), Light: Light{Ambient: ambient}}
		if len(engineFrame(f).Lighting.Lights) != 0 {
			t.Fatal("zero direction produced an invalid directional light")
		}
		if _, err := renderSoftware(f); err != nil {
			t.Fatalf("ambient %g: %v", ambient, err)
		}
	}
}
