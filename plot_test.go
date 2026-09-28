package ntcharts3d

import (
	"github.com/NimbleMarkets/ntcharts3d/math3d"
	"image/color"
	"math"
	"reflect"
	"testing"
)

func TestFixedColorDomain(t *testing.T) {
	m := New(80, 24, WithRenderMode(Software), WithColorDomain(Range{Min: 0, Max: 10}))
	defer m.Close()
	m.SetScatter("a", Scatter{X: []float32{0, 1, 2}, Y: []float32{0, 0, 0}, Z: []float32{-5, 5, 20}})
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
	for i, want := range []color.RGBA{m.colors.At(0), m.colors.At(.5), m.colors.At(1)} {
		if got := m.series[0].geometry.Points[i].Color; got != want {
			t.Fatalf("point %d color %v, want %v", i, got, want)
		}
	}
	m.SetScatter("b", Scatter{X: []float32{0}, Y: []float32{0}, Z: []float32{100}})
	m.SetSharedColorDomain(true)
	m.SetSharedColorDomain(false)
	f := m.frame()
	if len(f.ColorLegends) != 1 || f.ColorLegends[0].Min != 0 || f.ColorLegends[0].Max != 10 {
		t.Fatalf("fixed domain lost: %+v", f.ColorLegends)
	}
	r := m.ColorDomain()
	r.Max = 500
	if m.ColorDomain().Max != 10 {
		t.Fatal("domain alias")
	}
	revision := m.revision
	m.SetColorDomain(&Range{Min: 1, Max: 1})
	if m.Err() == nil || m.revision != revision || m.ColorDomain().Max != 10 {
		t.Fatal("invalid domain applied")
	}
	m.SetColorDomain(nil)
	if m.Err() != nil || m.ColorDomain() != nil || len(m.frame().ColorLegends) != 2 || m.series[0].geometry.ColorMax != 20 {
		t.Fatal("automatic domains not restored")
	}
	m.SetSharedColorDomain(true)
	m.SetColorDomain(&Range{Min: 0, Max: 10})
	m.SetColorDomain(nil)
	if m.series[0].geometry.ColorMax != 100 {
		t.Fatal("shared domain not restored")
	}
	if mapped(m.colors, 0, -math.MaxFloat32, math.MaxFloat32) != m.colors.At(.5) {
		t.Fatal("color range overflow")
	}
}

func TestPlotRangeOwnershipAndCache(t *testing.T) {
	m := New(80, 24, WithRenderMode(Software))
	defer m.Close()
	m.SetScatter("a", Scatter{X: []float32{-2, 0, 2}, Y: []float32{0, 0, 0}, Z: []float32{0, 0, 0}})
	r := &Range{Min: -1, Max: 1}
	m.SetAxes(Axes{X: Axis{Range: r}})
	r.Max = 99
	old := m.frame()
	if len(old.Geometry[0].Points) != 1 || m.Bounds().Max.X != 2 || m.PlotBounds().Max.X != 1 {
		t.Fatal("range did not clip independently of data bounds")
	}
	a := m.Axes()
	a.X.Range.Min = -99
	if m.Axes().X.Range.Min != -1 {
		t.Fatal("getter alias")
	}
	m.SetAxes(Axes{X: Axis{Range: &Range{Min: 0, Max: float32(math.NaN())}}})
	if m.Err() == nil || m.PlotBounds().Max.X != 1 {
		t.Fatal("invalid range accepted")
	}
	m.SetAxes(Axes{X: Axis{Range: &Range{Min: 0, Max: 1}, Categories: []string{"a"}}})
	if m.Err() == nil {
		t.Fatal("category range accepted")
	}
	m.SetAxes(Axes{})
	if len(m.frame().Geometry[0].Points) != 3 || len(old.Geometry[0].Points) != 1 {
		t.Fatal("range reset or snapshot ownership failed")
	}
	m.SetAxes(Axes{X: Axis{Range: &Range{Min: -1, Max: 1}}, Y: Axis{Range: &Range{Min: -1, Max: 1}}, Z: Axis{Range: &Range{Min: -1, Max: 1}}})
	m.Clear()
	if !m.PlotBounds().IsValid() || len(m.frame().Axes[0].Ticks) == 0 {
		t.Fatal("fixed axes lost on empty chart")
	}
}

func TestClipPrimitives(t *testing.T) {
	var bounds math3d.AABB
	bounds.Include(math3d.Vec3{X: -1, Y: -1, Z: -1})
	bounds.Include(math3d.Vec3{X: 1, Y: 1, Z: 1})
	red, blue := color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255}
	a, b, c := Vertex{Position: math3d.Vec3{X: -2}, Color: red}, Vertex{Position: math3d.Vec3{X: 2}, Color: blue}, Vertex{Position: math3d.Vec3{Y: 2}, Color: red}
	g := Geometry{Points: []Point{{Position: a.Position}, {Position: math3d.Vec3{}, Datum: 42}}, Vertices: []Vertex{a, b, c}, Indices: []uint32{0, 1, 2}, Lines: []Vertex{a, b}, Boxes: []Box{{Min: math3d.Vec3{X: -2, Y: -2, Z: -2}, Size: math3d.Vec3{X: 4, Y: 4, Z: 4}, Datum: 7}}}
	clipped := clipGeometry(g, bounds)
	if len(clipped.Points) != 1 || clipped.Points[0].Datum != 42 || len(clipped.Indices) < 3 || len(clipped.Lines) != 2 || len(clipped.Boxes) != 1 {
		t.Fatalf("bad clip: %+v", clipped)
	}
	for _, v := range append(clipped.Vertices, clipped.Lines...) {
		if !contains(bounds, v.Position) {
			t.Fatal("vertex outside", v.Position)
		}
	}
	if clipped.Lines[0].Color.R != 191 || clipped.Lines[0].Color.B != 64 {
		t.Fatal("clip did not interpolate colors", clipped.Lines)
	}
	if clipped.Boxes[0].Min != bounds.Min || clipped.Boxes[0].Size != (math3d.Vec3{X: 2, Y: 2, Z: 2}) || clipped.Boxes[0].Datum != 7 {
		t.Fatal("bar clip lost extent or datum")
	}
	if g.Vertices[0].Position.X != -2 {
		t.Fatal("source mutated")
	}
	outside := Box{Min: math3d.Vec3{X: 3}, Size: math3d.Vec3{X: 1, Y: 1, Z: 1}}
	if _, ok := clipBox(outside, bounds); ok {
		t.Fatal("outside box retained")
	}
}

func TestPlotAspectTransformAndPicking(t *testing.T) {
	for _, projection := range []Projection{Orthographic, Perspective} {
		m := New(81, 43, WithRenderMode(Software), WithPlotAspect(math3d.Vec3{X: 1, Y: .7, Z: .5}), WithAxes(Axes{Z: Axis{Range: &Range{Min: 0, Max: 3}}}))
		m.SetBars("bar", Bars{NX: 1, NY: 1, Values: []float32{10}})
		c := DefaultCamera()
		c.Projection = projection
		m.SetCamera(c)
		transform := m.plotTransform()
		b := m.PlotBounds()
		x0, y0, z0, _ := transform.matrix.Transform(b.Min)
		x1, y1, z1, _ := transform.matrix.Transform(b.Max)
		for i, pair := range [][2]float32{{x1 - x0, 2}, {y1 - y0, 1.4}, {z1 - z0, 1}} {
			if abs(pair[0]-pair[1]) > 1e-5 {
				t.Fatalf("aspect dimension %d: %v", i, pair)
			}
		}
		hit := m.Pick(40, 20)
		if hit == nil || hit.Datum != 0 || hit.Position.Z != 10 {
			t.Fatalf("%v clipped bar pick: %+v", projection, hit)
		}
		m.setHover(hit)
		f := m.frame()
		if f.Emphasis == nil || f.Emphasis.Box == nil || f.Emphasis.Box.Size.Z != 3 {
			t.Fatal("highlight does not match clipped bar")
		}
		normal := transform.normal(math3d.Vec3{X: 1, Y: 1, Z: 1})
		tangent := math3d.Vec3{X: transform.scale.X, Y: -transform.scale.Y}
		if abs(normal.Dot(tangent)) > 1e-5 {
			t.Fatal("normal is not perpendicular after scaling")
		}
		old := m.PlotAspect()
		m.SetPlotAspect(math3d.Vec3{X: 1})
		if m.Err() == nil || m.PlotAspect() != old {
			t.Fatal("invalid aspect applied")
		}
		m.SetPlotAspect(math3d.Vec3{})
		if m.Err() != nil {
			t.Fatal(m.Err())
		}
		m.Close()
	}
}

func TestClippedPointCannotBePicked(t *testing.T) {
	m := New(81, 43, WithRenderMode(Software))
	defer m.Close()
	m.SetScatter("points", Scatter{X: []float32{0, 0}, Y: []float32{0, 0}, Z: []float32{0, 3}})
	m.SetAxes(Axes{Z: Axis{Range: &Range{Min: 0, Max: 1}}})
	c := DefaultCamera()
	c.Distance = 10
	m.SetCamera(c)
	f := m.frame()
	x, y, _, ok := f.Matrix.Project(math3d.Vec3{Z: 3}, f.Width, f.Height)
	if !ok {
		t.Fatal("test point is off screen")
	}
	if hit := m.Pick(int(x*float32(m.plotWidth())/float32(f.Width)), int(y*float32(m.plotHeight())/float32(f.Height))); hit != nil && hit.Datum == 1 {
		t.Fatal("clipped datum picked")
	}
}

func TestHoverDoesNotRebuildGeometry(t *testing.T) {
	m := New(81, 43, WithRenderMode(Software), WithEmphasis(Emphasis{ShowLabel: true}))
	defer m.Close()
	m.SetScatter("point", Scatter{X: []float32{-1, 0, 1}, Y: []float32{0, 0, 0}, Z: []float32{0, 0, 0}})
	original := m.frame()
	hit := m.Pick(40, 20)
	if hit == nil {
		t.Fatal("no point at center")
	}
	seq := m.seq
	m.setHover(hit)
	highlighted := m.frame()
	if highlighted.Revision != original.Revision || m.seq == seq || highlighted.Emphasis == nil || highlighted.Emphasis.Label == "" {
		t.Fatal("hover did not update overlay independently")
	}
	if &highlighted.Geometry[0].Points[0] != &original.Geometry[0].Points[0] {
		t.Fatal("hover rebuilt geometry")
	}
	seq = m.seq
	m.setHover(hit)
	if m.seq != seq {
		t.Fatal("unchanged hover scheduled a frame")
	}
	if reflect.DeepEqual(frameOverlayRects(original), frameOverlayRects(highlighted)) {
		t.Fatal("hover has no visible effect")
	}
	m.setHover(nil)
	if m.frame().Emphasis != nil {
		t.Fatal("hover did not clear")
	}
}

func TestPlotAspectFlatDataKeepsAxesVisible(t *testing.T) {
	m := New(100, 40, WithRenderMode(Software), WithPlotAspect(math3d.Vec3{X: 1, Y: 1, Z: 1}))
	defer m.Close()
	m.SetScatter("flat", Scatter{X: []float32{-1, 1}, Y: []float32{-1, 1}, Z: []float32{0, 0}})
	f := m.frame()
	f.Width, f.Height = 800, 600
	f.Matrix = m.Camera().Matrix(800.0 / 600).Mul(m.plotTransform().matrix)
	names := map[string]bool{}
	for _, label := range layoutAxes(f, 800, 600, textWidth, textHeight, nil).labels {
		names[label.text] = true
	}
	if !names["X"] || !names["Y"] || !names["Z"] {
		t.Fatal("flat plot lost axes", names)
	}
}
