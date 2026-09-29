// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"image/color"
	"math"
	"testing"

	"github.com/NimbleMarkets/ntcharts3d/math3d"
)

func TestVectorFieldOwnershipMagnitudeAndNormalization(t *testing.T) {
	m := New(50, 25, WithRenderMode(Software))
	defer m.Close()
	d := VectorField{Origins: []math3d.Vec3{{}, {X: 1}, {Y: 1}}, Vectors: []math3d.Vec3{{}, {X: 3, Y: 4}, {Z: 2}}, Scale: .5, Normalize: true, Labels: []string{"zero", "five", "two"}}
	m.SetVectorField("velocity", d)
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
	g := m.series[0].geometry
	if len(g.Arrows) != 2 || g.Arrows[0].Datum != 1 || g.ColorMin != 0 || g.ColorMax != 5 {
		t.Fatal("magnitude domain or zero vector handling", g)
	}
	a := g.Arrows[0]
	if abs(a.End.X-1.3) > 1e-6 || abs(a.End.Y-.4) > 1e-6 || a.Width != 2 || a.HeadSize != 8 {
		t.Fatal("normalization/defaults", a)
	}
	d.Origins[1].X = 99
	d.Vectors[1].X = 99
	d.Labels[1] = "changed"
	m.SetPalette([]color.Color{color.Black, color.White})
	if m.Err() != nil || m.series[0].geometry.Arrows[0].Start.X != 1 || m.series[0].geometry.Labels[1] != "five" {
		t.Fatal("caller alias", m.Err())
	}
	m.SetColorDomain(&Range{Min: 0, Max: 10})
	if m.Err() != nil || m.series[0].geometry.Arrows[0].Color != mapped(m.colors, 5, 0, 10) {
		t.Fatal("fixed domain", m.Err())
	}
	old := m.frame()
	camera := m.Camera()
	camera.Beta += 30
	m.SetCamera(camera)
	next := m.frame()
	if old.Revision != next.Revision || &old.Geometry[0].Arrows[0] != &next.Geometry[0].Arrows[0] {
		t.Fatal("camera rebuilt arrows")
	}
}
func TestLinesOwnershipColorsAndValidation(t *testing.T) {
	m := New(40, 20)
	defer m.Close()
	d := Lines{Start: []math3d.Vec3{{Z: 1}, {Z: 3}}, End: []math3d.Vec3{{X: 1, Z: 3}, {X: 1, Z: 5}}, Width: 7, Labels: []string{"a", "b"}}
	m.SetLines("paths", d)
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
	g := m.series[0].geometry
	if g.LineWidth != 7 || len(g.Lines) != 4 || g.ColorMin != 2 || g.ColorMax != 4 {
		t.Fatal("line geometry", g)
	}
	d.Start[0].X = 50
	d.Labels[0] = "changed"
	m.SetSharedColorDomain(true)
	if m.Err() != nil || m.series[0].geometry.Lines[0].Position.X != 0 || m.series[0].geometry.Labels[0] != "a" {
		t.Fatal("line ownership")
	}
	revision := m.revision
	for _, bad := range []Lines{{}, {Start: []math3d.Vec3{{}}, End: []math3d.Vec3{}}, {Start: []math3d.Vec3{{}}, End: []math3d.Vec3{{}}, Width: 33}, {Start: []math3d.Vec3{{}}, End: []math3d.Vec3{{X: float32(math.NaN())}}}} {
		m.SetLines("paths", bad)
		if m.Err() == nil || m.revision != revision {
			t.Fatal("accepted invalid lines")
		}
	}
	m.SetLines("paths", Lines{Start: []math3d.Vec3{{}}, End: []math3d.Vec3{{X: 1}}, Color: []uint32{0xff008000}})
	if m.Err() != nil || m.series[0].geometry.HasColorRange || m.series[0].geometry.Lines[0].Color != (color.RGBA{255, 0, 128, 255}) {
		t.Fatal("explicit color", m.Err())
	}
}
func TestInvalidVectorFieldIsAtomic(t *testing.T) {
	m := New(40, 20)
	defer m.Close()
	base := VectorField{Origins: []math3d.Vec3{{}}, Vectors: []math3d.Vec3{{X: 1}}}
	m.SetVectorField("v", base)
	revision := m.revision
	for _, change := range []func(*VectorField){
		func(d *VectorField) { d.Vectors = nil }, func(d *VectorField) { d.Width = -1 }, func(d *VectorField) { d.Width = .5 }, func(d *VectorField) { d.HeadSize = 65 }, func(d *VectorField) { d.Scale = -1 }, func(d *VectorField) { d.Scale = float32(math.Inf(1)) }, func(d *VectorField) { d.ColorValue = []float32{float32(math.NaN())} }, func(d *VectorField) { d.Vectors = []math3d.Vec3{{X: math.MaxFloat32}}; d.Scale = 2 },
	} {
		d := base
		change(&d)
		m.SetVectorField("v", d)
		if m.Err() == nil || m.revision != revision {
			t.Fatal("accepted invalid vector field", d)
		}
	}
	for _, g := range []Geometry{{LineWidth: -1}, {Arrows: []Arrow{{Width: 33}}}, {Arrows: []Arrow{{HeadSize: 65}}}, {Arrows: []Arrow{{End: math3d.Vec3{X: float32(math.NaN())}}}}} {
		if validateGeometry(g) == nil {
			t.Fatal("accepted invalid custom geometry")
		}
	}
}
func TestArrowPlotClipping(t *testing.T) {
	var bounds math3d.AABB
	bounds.Include(math3d.Vec3{X: -1, Y: -1, Z: -1})
	bounds.Include(math3d.Vec3{X: 1, Y: 1, Z: 1})
	a := Arrow{Start: math3d.Vec3{X: -2}, End: math3d.Vec3{X: 2}, Width: 3, HeadSize: 10, Datum: 9}
	clipped, ok := clipArrow(a, bounds)
	if !ok || clipped.Start.X != -1 || clipped.End.X != 1 || clipped.HeadSize != 0 || clipped.Datum != 9 || clipped.Width != 3 {
		t.Fatal("tip clipping", clipped)
	}
	a.End.X = .5
	clipped, ok = clipArrow(a, bounds)
	if !ok || clipped.Start.X != -1 || clipped.HeadSize != 10 {
		t.Fatal("start clipping lost head", clipped)
	}
	g := clipGeometry(Geometry{Arrows: []Arrow{a}, LineWidth: 5, Lines: []Vertex{{Position: math3d.Vec3{X: -2}}, {Position: math3d.Vec3{X: 2}}}}, bounds)
	if len(g.Arrows) != 1 || g.LineWidth != 5 || len(g.Lines) != 2 {
		t.Fatal("geometry clipping lost strokes")
	}
}
func strokeTestFrame() Frame {
	red, green := color.RGBA{220, 30, 40, 255}, color.RGBA{20, 180, 80, 255}
	return Frame{Width: 200, Height: 150, Revision: 1, Matrix: math3d.Identity(), Background: color.RGBA{240, 240, 240, 255}, Geometry: []Geometry{{LineWidth: 6, Lines: []Vertex{{Position: math3d.Vec3{X: -.8, Y: .6, Z: .5}, Color: red}, {Position: math3d.Vec3{X: .8, Y: .6, Z: .5}, Color: red}}, Arrows: []Arrow{{Start: math3d.Vec3{X: -.8, Y: -.2, Z: .5}, End: math3d.Vec3{X: .8, Y: -.2, Z: .5}, Width: 4, HeadSize: 20, Color: green}}, Points: []Point{{Position: math3d.Vec3{Y: -.2, Z: .2}, Radius: 3, Color: color.RGBA{30, 50, 220, 255}}}}}}
}
func TestSoftwareStrokesAndDepth(t *testing.T) {
	f := strokeTestFrame()
	img := softwareRender(f)
	g := f.Geometry[0]
	for _, test := range []struct {
		x, y int
		c    color.RGBA
	}{{40, 27, g.Lines[0].Color}, {40, 32, g.Lines[0].Color}, {40, 26, f.Background}, {40, 33, f.Background}, {40, 89, g.Arrows[0].Color}, {40, 86, f.Background}, {164, 94, g.Arrows[0].Color}, {170, 99, f.Background}, {178, 90, g.Arrows[0].Color}, {100, 90, g.Points[0].Color}} {
		if got := img.At(test.x, test.y); got != test.c {
			t.Fatal("stroke pixel", test, got)
		}
	}
	// Both endpoints may be outside the screen while the line crosses it.
	f.Geometry = []Geometry{{Lines: []Vertex{{Position: math3d.Vec3{X: -2, Z: .5}, Color: g.Lines[0].Color}, {Position: math3d.Vec3{X: 2, Z: .5}, Color: g.Lines[0].Color}}}}
	if got := softwareRender(f).At(100, 74); got != g.Lines[0].Color {
		t.Fatal("offscreen endpoints rejected", got)
	}
	// A view-aligned arrow stays finite and has a small end-on marker.
	f.Geometry = []Geometry{{Arrows: []Arrow{{Start: math3d.Vec3{Z: .2}, End: math3d.Vec3{Z: .8}, Width: 4, HeadSize: 8, Color: g.Arrows[0].Color}}}}
	if got := softwareRender(f).At(100, 75); got != g.Arrows[0].Color {
		t.Fatal("end-on arrow missing", got)
	}
}
func TestStrokeDepthClippingAndPerspective(t *testing.T) {
	f := strokeTestFrame()
	s, ok := projectStroke(f, math3d.Vec3{X: -.8, Z: -.5}, math3d.Vec3{X: .8, Z: .5})
	if !ok || abs(s.t0-.5) > 1e-6 || abs(s.a.z) > 1e-6 {
		t.Fatal("near clip", s)
	}
	s, ok = projectStroke(f, math3d.Vec3{X: -.8, Z: .5}, math3d.Vec3{X: .8, Z: 1.5})
	hl, _ := s.head(10, 2)
	if !ok || s.t1 != .5 || hl != 0 {
		t.Fatal("far clip invented head", s)
	}
	if _, ok = projectStroke(f, math3d.Vec3{Z: -2}, math3d.Vec3{Z: -1}); ok {
		t.Fatal("accepted hidden segment")
	}
	f.Matrix[3] = .5
	s, ok = projectStroke(f, math3d.Vec3{X: -.5, Z: .5}, math3d.Vec3{X: .5, Z: .5})
	if !ok || abs(s.worldT(.5)-.375) > 1e-6 {
		t.Fatal("perspective parameter", s.worldT(.5))
	}
}
func TestPickLineAndArrowDatums(t *testing.T) {
	for _, arrows := range []bool{false, true} {
		m := New(80, 30, WithRenderMode(Software))
		defer m.Close()
		a, b := math3d.Vec3{X: -1}, math3d.Vec3{X: 1}
		if arrows {
			m.SetVectorField("field", VectorField{Origins: []math3d.Vec3{{}, a}, Vectors: []math3d.Vec3{{}, b.Sub(a)}, Labels: []string{"zero", "velocity"}, Width: 12})
		} else {
			m.SetLines("path", Lines{Start: []math3d.Vec3{a}, End: []math3d.Vec3{b}, Width: 12, Labels: []string{"streamline"}})
		}
		f := m.frame()
		x, y, _, _ := f.Matrix.Project(math3d.Vec3{}, f.Width, f.Height)
		hit := m.Pick(int(x*float32(m.plotWidth())/float32(f.Width)), int(y*float32(m.plotHeight())/float32(f.Height)))
		if hit == nil {
			t.Fatal("stroke not pickable", arrows)
		}
		if arrows && (hit.Datum != 1 || hit.Label != "velocity") || !arrows && (hit.Datum != 0 || hit.Label != "streamline") {
			t.Fatal("wrong source datum", hit)
		}
	}
}
