package ntcharts3d

import (
	"github.com/NimbleMarkets/ntcharts3d/math3d"
	"image/color"
	"reflect"
	"sync"
	"testing"
	"unicode/utf8"
)

func TestGridFollowsBackPlanes(t *testing.T) {
	m := New(80, 30, WithRenderMode(Software), WithGrid(Grid{Show: true}))
	defer m.Close()
	m.SetScatter("bounds", Scatter{X: []float32{-1, 1}, Y: []float32{-1, 1}, Z: []float32{-1, 1}})
	first := m.frame()
	if len(first.GridLines) == 0 {
		t.Fatal("no grid")
	}
	for _, projection := range []Projection{Orthographic, Perspective} {
		for _, angle := range []float64{40, 130, 220, 310} {
			c := DefaultCamera()
			c.Beta = angle
			c.Projection = projection
			m.SetCamera(c)
			f := m.frame()
			eye, _, _, _ := c.basis()
			e := components(eye)
			lo, hi := components(f.Bounds.Min), components(f.Bounds.Max)
			for i := 0; i < len(f.GridLines); i += 2 {
				a, b := components(f.GridLines[i].Position), components(f.GridLines[i+1].Position)
				far := false
				for axis := range 3 {
					bound := lo[axis]
					if e[axis] < 0 {
						bound = hi[axis]
					}
					if a[axis] == bound && b[axis] == bound {
						far = true
					}
				}
				if !far {
					t.Fatalf("grid on near face: %v %v", a, b)
				}
			}
			if f.Revision != first.Revision {
				t.Fatal("grid orbit rebuilt data")
			}
		}
	}
	m.SetGrid(Grid{})
	if len(m.frame().GridLines) != 0 {
		t.Fatal("grid not hidden")
	}
}

func gridOcclusionFrame() Frame {
	red := color.RGBA{R: 255, A: 255}
	green := color.RGBA{G: 255, A: 255}
	return Frame{Width: 200, Height: 150, Revision: 1, Matrix: math3d.Identity(), Background: color.RGBA{R: 240, G: 240, B: 240, A: 255},
		Geometry:  []Geometry{{Points: []Point{{Position: math3d.Vec3{Z: .2}, Radius: 8, Color: red}}}},
		GridLines: []Vertex{{Position: math3d.Vec3{X: -.8, Z: .8}, Color: green}, {Position: math3d.Vec3{X: .8, Z: .8}, Color: green}},
	}
}
func TestSoftwareGridOcclusion(t *testing.T) {
	f := gridOcclusionFrame()
	img := softwareRender(f)
	if got := color.RGBAModel.Convert(img.At(100, 75)).(color.RGBA); got != f.Geometry[0].Points[0].Color {
		t.Fatal("grid covers foreground point", got)
	}
	if got := color.RGBAModel.Convert(img.At(30, 75)).(color.RGBA); got != f.GridLines[0].Color {
		t.Fatal("grid missing", got)
	}
}
func TestScientificText(t *testing.T) {
	const text = "−20 °C · 5 µm × 2"
	if got := displayText(text, 64); got != text {
		t.Fatalf("unsupported scientific text: %q", got)
	}
	if got := displayText(text, 5); utf8.RuneCountInString(got) != 5 || !utf8.ValidString(got) {
		t.Fatal("bad Unicode truncation", got)
	}
	ink := color.RGBA{A: 255}
	fallback := bitmapText("?", 0, 0, ink, 1)
	for _, r := range []rune{'°', 'µ', '−'} {
		pixels := bitmapText(string(r), 0, 0, ink, 1)
		if len(pixels) == 0 || reflect.DeepEqual(pixels, fallback) {
			t.Fatalf("missing glyph %c", r)
		}
	}
	f := axisTestFrame()
	f.Matrix = DefaultCamera().Matrix(1.333)
	f.Axes[0].Name = text
	for _, label := range layoutAxes(f, f.Width, f.Height, textWidth, textHeight, nil).labels {
		if label.rect.Dx() != utf8.RuneCountInString(label.text)*textWidth {
			t.Fatal("label uses byte width")
		}
	}
	// Multiple models can render overlays concurrently.
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10 {
				bitmapText(displayText(text, 64), 0, 0, ink, 1)
			}
		}()
	}
	wg.Wait()
}
