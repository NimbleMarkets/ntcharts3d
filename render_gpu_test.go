//go:build webgpu

package ntcharts3d

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"

	nt3dmath "nimbleterminal.dev/nt3d/math"
)

func TestGPUChartGeometryAndCamera(t *testing.T) {
	r := &gpuRenderer{}
	defer r.Close()
	scale := defaultPalette()
	surface, err := (surfaceSeries{"surface", Surface{NX: 8, NY: 8, Sampler: func(x, y float64) float64 { return .2 * x * y }, Wireframe: true}}).Geometry(scale)
	if err != nil {
		t.Fatal(err)
	}
	bars, _ := (barSeries{"bars", Bars{NX: 2, NY: 2, Values: []float32{.2, .8, .4, .6}}}).Geometry(scale)
	points, _ := (scatterSeries{"points", Scatter{X: []float32{0, .5}, Y: []float32{0, .5}, Z: []float32{.6, .8}}}).Geometry(scale)
	for i, g := range []Geometry{surface, bars, points} {
		n, _, _ := g.Bounds.Normalization()
		f := Frame{Width: 400, Height: 240, Matrix: DefaultCamera().Matrix(1.67).Mul(n), Revision: uint64(i + 1), Geometry: []Geometry{g}, Bounds: g.Bounds, Axes: [3]AxisFrame{{Name: "X", Ticks: numericTicks(g.Bounds.Min.X, g.Bounds.Max.X, 5, nil)}, {Name: "Y"}, {Name: "Z"}}, ColorLegends: []ColorLegend{{Label: "value", Min: 0, Max: 1, Palette: scale}}, Background: color.RGBA{235, 238, 242, 255}, Light: Light{nt3dmath.Vec3{X: 1, Y: -1, Z: 2}, .3}}
		img, e := r.Render(f)
		if e != nil {
			t.Fatal(e)
		}
		nonBackground(t, img, f.Background)
		if got := color.RGBAModel.Convert(img.At(376, 29)).(color.RGBA); got != scale.At(1) {
			t.Fatalf("GPU legend max color = %v, want %v", got, scale.At(1))
		}
		c := DefaultCamera()
		c.Beta += 15
		f.Matrix = c.Matrix(1.25).Mul(n)
		if _, e = r.Render(f); e != nil {
			t.Fatal(e)
		}
	}
}

func TestGPUAxisLinesStayThin(t *testing.T) {
	r := &gpuRenderer{}
	defer r.Close()
	red := color.RGBA{210, 65, 65, 255}
	bg := color.RGBA{235, 238, 242, 255}
	g := Geometry{Lines: []Vertex{
		{Position: nt3dmath.Vec3{X: -.8, Y: 0, Z: .5}, Color: red}, {Position: nt3dmath.Vec3{X: .8, Y: 0, Z: .5}, Color: red},
		{Position: nt3dmath.Vec3{X: 0, Y: -.8, Z: .5}, Color: red}, {Position: nt3dmath.Vec3{X: 0, Y: .8, Z: .5}, Color: red},
	}}
	img, err := r.Render(Frame{Width: 200, Height: 150, Matrix: nt3dmath.Identity(), Revision: 1, Geometry: []Geometry{g}, Background: bg})
	if err != nil {
		t.Fatal(err)
	}
	changed := 0
	for y := 0; y < 150; y++ {
		for x := 0; x < 200; x++ {
			if color.RGBAModel.Convert(img.At(x, y)).(color.RGBA) != bg {
				changed++
			}
		}
	}
	if changed < 150 || changed > 600 {
		t.Fatalf("axis lines should cover roughly one pixel in width, got %d colored pixels", changed)
	}
}

func TestGPUAxisLabelsAndCameraUpdates(t *testing.T) {
	r := &gpuRenderer{}
	defer r.Close()
	f := axisTestFrame()
	f.Background = color.RGBA{235, 238, 242, 255}
	f.Revision = 1
	for _, beta := range []float64{40, 130, 220, 310} {
		c := DefaultCamera()
		c.Beta = beta
		f.Matrix = c.Matrix(float32(f.Width) / float32(f.Height))
		img, err := r.Render(f)
		if err != nil {
			t.Fatal(err)
		}
		labels := layoutAxes(f, f.Width, f.Height, 2*textWidth, 2*textHeight, nil).labels
		if len(labels) < 3 {
			t.Fatal("missing projected labels")
		}
		for _, label := range labels {
			pixels := bitmapText(label.text, label.rect.Min.X, label.rect.Min.Y, label.color, 2)
			if len(pixels) == 0 {
				t.Fatal("empty label bitmap")
			}
			p := pixels[0]
			if got := color.RGBAModel.Convert(img.At(p.x, p.y)).(color.RGBA); got != label.color {
				t.Fatalf("label %q at camera %g: %v, want %v", label.text, beta, got, label.color)
			}
		}
	}
}

func nonBackground(t *testing.T, img image.Image, bg color.RGBA) {
	t.Helper()
	n := 0
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			if c != bg {
				n++
			}
		}
	}
	if n < 10 {
		t.Fatalf("empty scene (%d changed pixels)", n)
	}
}

func BenchmarkScatterGPU(b *testing.B) {
	for _, n := range []int{50000, 200000, 500000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			points := make([]Point, n)
			for i := range points {
				a := float64(i) * 2.399963229728653
				z := 1 - 2*float64(i)/float64(n)
				r := math.Sqrt(1 - z*z)
				points[i] = Point{Position: nt3dmath.Vec3{X: float32(r * math.Cos(a)), Y: float32(r * math.Sin(a)), Z: float32(z)}, Radius: 2, Color: color.RGBA{40, 180, 150, 255}, Datum: i}
			}
			r := &gpuRenderer{}
			defer r.Close()
			c := DefaultCamera()
			f := Frame{Width: 640, Height: 400, Matrix: c.Matrix(1.6), Revision: 1, Geometry: []Geometry{{Points: points}}, Background: color.RGBA{235, 238, 242, 255}, Light: Light{nt3dmath.Vec3{Z: 1}, .3}}
			if _, err := r.Render(f); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				c.Beta += 1
				f.Matrix = c.Matrix(1.6)
				if _, err := r.Render(f); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestGPUGridAndHover(t *testing.T) {
	r := &gpuRenderer{}
	defer r.Close()
	f := gridOcclusionFrame()
	img, err := r.Render(f)
	if err != nil {
		t.Fatal(err)
	}
	if got := color.RGBAModel.Convert(img.At(100, 75)).(color.RGBA); got != f.Geometry[0].Points[0].Color {
		t.Fatal("grid covers foreground point", got)
	}
	// A one-pixel GPU line falls on either neighboring pixel center.
	visible := false
	for _, y := range []int{74, 75} {
		if color.RGBAModel.Convert(img.At(30, y)).(color.RGBA) == f.GridLines[0].Color {
			visible = true
		}
	}
	if !visible {
		t.Fatal("GPU grid missing")
	}
	f.Emphasis = &EmphasisFrame{Position: nt3dmath.Vec3{Z: .2}, Radius: 8, Color: color.RGBA{B: 255, A: 255}, Label: "5 µm"}
	img, err = r.Render(f)
	if err != nil {
		t.Fatal(err)
	}
	pixels := emphasisRects(f)
	if len(pixels) == 0 {
		t.Fatal("no hover overlay")
	}
	p := pixels[0]
	if got := color.RGBAModel.Convert(img.At(p.x, p.y)).(color.RGBA); got != p.c {
		t.Fatal("GPU hover missing", got, p.c)
	}
	f.GridLines = nil
	f.Emphasis = nil
	if _, err = r.Render(f); err != nil {
		t.Fatal(err)
	}
}

func TestGPUTextureMipmaps(t *testing.T) {
	r := &gpuRenderer{}
	defer r.Close()
	f := minifiedTextureFrame(t)
	for _, n := range []int{1, 4, 16} {
		f.Anisotropy = n
		img, err := r.Render(f)
		if err != nil {
			t.Fatal(err)
		}
		checkMinifiedTexture(t, img)
	}
}

func TestGPUTexturesAndIndependentUpdates(t *testing.T) {
	r := &gpuRenderer{}
	defer r.Close()
	f := textureTestFrame(t)
	gpu, err := r.Render(f)
	if err != nil {
		t.Fatal(err)
	}
	cpu := softwareRender(f)
	for _, p := range []image.Point{{10, 10}, {90, 10}, {10, 90}, {90, 90}, {40, 60}, {55, 45}} {
		a, b := color.RGBAModel.Convert(gpu.At(p.X, p.Y)).(color.RGBA), color.RGBAModel.Convert(cpu.At(p.X, p.Y)).(color.RGBA)
		if abs(float32(a.R)-float32(b.R)) > 2 || abs(float32(a.G)-float32(b.G)) > 2 || abs(float32(a.B)-float32(b.B)) > 2 {
			t.Fatal("GPU/software texture mismatch", p, a, b)
		}
	}
	// Perspective interpolation remains consistent between both renderers.
	f.Matrix[3] = .5
	gpu, err = r.Render(f)
	if err != nil {
		t.Fatal(err)
	}
	cpu = softwareRender(f)
	a, b := color.RGBAModel.Convert(gpu.At(55, 45)).(color.RGBA), cpu.At(55, 45).(color.RGBA)
	if abs(float32(a.R)-float32(b.R)) > 2 || abs(float32(a.G)-float32(b.G)) > 2 || abs(float32(a.B)-float32(b.B)) > 2 {
		t.Fatal("GPU/software perspective mismatch", a, b)
	}
	replacement := image.NewRGBA(image.Rect(0, 0, 3, 1))
	for x := range 3 {
		replacement.SetRGBA(x, 0, color.RGBA{123, 45, 67, 255})
	}
	next, err := NewTexture(replacement)
	if err != nil {
		t.Fatal(err)
	}
	f.Geometry[0].Material = &Material{Texture: next, Unlit: true}
	f.TextureRevision++
	gpu, err = r.Render(f)
	if err != nil {
		t.Fatal(err)
	}
	if got := color.RGBAModel.Convert(gpu.At(55, 45)).(color.RGBA); got != (color.RGBA{123, 45, 67, 255}) {
		t.Fatal("replacement image", got)
	}
	// Geometry rebuilds keep an unchanged texture resident.
	f.Revision++
	if _, err = r.Render(f); err != nil {
		t.Fatal(err)
	}
	// Shared images upload only once, and lit materials match the CPU path.
	f.Geometry[0].Material = &Material{Texture: next}
	f.Geometry = append(f.Geometry, f.Geometry[0])
	f.Revision++
	gpu, err = r.Render(f)
	if err != nil {
		t.Fatal(err)
	}
	cpu = softwareRender(f)
	a, b = color.RGBAModel.Convert(gpu.At(55, 45)).(color.RGBA), cpu.At(55, 45).(color.RGBA)
	if abs(float32(a.R)-float32(b.R)) > 1 || abs(float32(a.G)-float32(b.G)) > 1 || abs(float32(a.B)-float32(b.B)) > 1 {
		t.Fatal("lit GPU/software mismatch", a, b)
	}
	f.Geometry = nil
	f.Revision++
	if _, err = r.Render(f); err != nil {
		t.Fatal(err)
	}
}

func TestGPUWideLinesAndArrowInstances(t *testing.T) {
	r := &gpuRenderer{}
	defer r.Close()
	f := strokeTestFrame()
	compare := func() {
		t.Helper()
		gpu, err := r.Render(f)
		if err != nil {
			t.Fatal(err)
		}
		cpu := softwareRender(f)
		mismatch := 0
		for y := range f.Height {
			for x := range f.Width {
				a, b := color.RGBAModel.Convert(gpu.At(x, y)).(color.RGBA), cpu.At(x, y).(color.RGBA)
				if abs(float32(a.R)-float32(b.R)) > 2 || abs(float32(a.G)-float32(b.G)) > 2 || abs(float32(a.B)-float32(b.B)) > 2 {
					mismatch++
				}
			}
		}
		if mismatch > 60 {
			t.Fatalf("GPU/software strokes differ at %d pixels", mismatch)
		}
	}
	// Compare the strokes separately from legacy point rasterization.
	f.Geometry[0].Points = nil
	compare()
	// Camera and viewport changes preserve visible stroke agreement.
	f.Matrix[3] = .5
	compare()
	f.Width, f.Height = 240, 180
	compare()
	// Both depth clipping directions and a view-aligned arrow stay well-defined.
	ink := color.RGBA{20, 160, 100, 255}
	f.Matrix = nt3dmath.Identity()
	f.Revision++
	f.Geometry = []Geometry{{Arrows: []Arrow{
		{Start: nt3dmath.Vec3{X: -.8, Y: .5, Z: -.5}, End: nt3dmath.Vec3{X: .8, Y: .5, Z: .5}, Width: 4, HeadSize: 20, Color: ink},
		{Start: nt3dmath.Vec3{X: -.8, Y: -.5, Z: .5}, End: nt3dmath.Vec3{X: .8, Y: -.5, Z: 1.5}, Width: 4, HeadSize: 20, Color: ink},
		{Start: nt3dmath.Vec3{Z: .2}, End: nt3dmath.Vec3{Z: .8}, Width: 4, HeadSize: 20, Color: ink},
	}}}
	compare()
	f.Geometry = nil
	f.Revision++
	if _, err := r.Render(f); err != nil {
		t.Fatal(err)
	}
}
