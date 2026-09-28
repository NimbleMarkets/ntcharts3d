package ntcharts3d

import (
	"fmt"
	"image"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts3d/math3d"
)

func TestNumericTicks(t *testing.T) {
	for _, tt := range []struct {
		lo, hi float32
		values []float32
		labels []string
	}{
		{-3, 7, []float32{0, 5}, []string{"0", "5"}},
		{0, .004, []float32{0, .001, .002, .003, .004}, []string{"0", "0.001", "0.002", "0.003", "0.004"}},
		{12, 12, []float32{12}, []string{"12"}},
		{1000000, 1000004, []float32{1000000, 1000001, 1000002, 1000003, 1000004}, nil},
	} {
		ticks := numericTicks(tt.lo, tt.hi, 5, nil)
		var values []float32
		var labels []string
		for _, tick := range ticks {
			values = append(values, tick.Value)
			labels = append(labels, tick.Label)
		}
		if !reflect.DeepEqual(values, tt.values) {
			t.Errorf("%g..%g: ticks %v, want %v", tt.lo, tt.hi, values, tt.values)
		}
		if tt.labels != nil && !reflect.DeepEqual(labels, tt.labels) {
			t.Errorf("labels %v, want %v", labels, tt.labels)
		}
		for i := 1; i < len(labels); i++ {
			if labels[i] == labels[i-1] {
				t.Errorf("duplicate label %q", labels[i])
			}
		}
	}
	ticks := numericTicks(0, 1, 3, func(v float64) string { return fmt.Sprintf("%.0f%%", v*100) })
	if ticks[1].Label != "50%" {
		t.Fatalf("formatter ignored: %v", ticks)
	}
	for _, pair := range [][2]float32{{-math.MaxFloat32, math.MaxFloat32}, {0, math.SmallestNonzeroFloat32}, {-1e-30, 1e-30}, {4, 3}, {float32(math.NaN()), 1}} {
		ticks := numericTicks(pair[0], pair[1], 64, nil)
		if len(ticks) > 64 {
			t.Fatal("unbounded ticks")
		}
		for _, tick := range ticks {
			if !math3d.IsFinite(float64(tick.Value)) {
				t.Fatal("non-finite tick")
			}
		}
	}
}

func TestAxesCategoryOwnershipAndValidation(t *testing.T) {
	categories := []string{"Mon", "Tue"}
	m := New(80, 25, WithAxes(Axes{X: Axis{Name: "Day", Categories: categories}}), WithRenderMode(Software))
	defer m.Close()
	categories[0] = "changed"
	m.SetBars("bars", Bars{NX: 2, NY: 1, Values: []float32{1, 2}})
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
	if got := m.datumLabel(0, 0); got != "Mon / 0" {
		t.Fatal(got)
	}
	f := m.frame()
	rev := f.Revision
	if got := f.Axes[0].Ticks; len(got) != 2 || got[1].Label != "Tue" || got[1].Value != 1 {
		t.Fatalf("category ticks: %v", got)
	}
	copy := m.Axes()
	copy.X.Categories[0] = "copy"
	if m.Axes().X.Categories[0] != "Mon" {
		t.Fatal("getter exposed category storage")
	}
	m.SetAxes(Axes{X: Axis{Categories: []string{"wrong count"}}})
	if m.Err() == nil || m.Axes().X.Name != "Day" {
		t.Fatal("invalid axes partially applied")
	}
	m.SetBars("bad", Bars{NX: 3, NY: 1, Values: []float32{1, 2, 3}})
	if m.Err() == nil || len(m.series) != 1 {
		t.Fatal("mismatched series accepted")
	}
	m.SetAxes(Axes{X: Axis{Categories: []string{"A", "B"}}})
	if m.Err() != nil || m.revision != rev || m.datumLabel(0, 0) != "A / 0" {
		t.Fatal("axis update failed or dirtied geometry")
	}
	if f.Axes[0].Ticks[0].Label != "Mon" {
		t.Fatal("axis update changed a pending frame")
	}
	m.SetAxes(Axes{Z: Axis{TickCount: -1}})
	if m.Err() == nil {
		t.Fatal("invalid tick count accepted")
	}
}

func axisTestFrame() Frame {
	var b math3d.AABB
	b.Include(math3d.Vec3{X: -1, Y: -1, Z: -1})
	b.Include(math3d.Vec3{X: 1, Y: 1, Z: 1})
	f := Frame{Width: 800, Height: 600, Bounds: b}
	for i := range 3 {
		f.Axes[i] = AxisFrame{Name: axisName(i), Ticks: numericTicks(-1, 1, 5, nil)}
	}
	return f
}

func TestAxisLayoutOrbitAndCollisions(t *testing.T) {
	f := axisTestFrame()
	reserve := []image.Rectangle{image.Rect(600, 8, 790, 160)}
	for _, projection := range []Projection{Orthographic, Perspective} {
		for _, elevation := range []float64{-45, 25, 70} {
			for azimuth := 0; azimuth < 360; azimuth += 30 {
				c := DefaultCamera()
				c.Projection = projection
				c.Alpha = elevation
				c.Beta = float64(azimuth)
				c.Distance = 4.5
				f.Matrix = c.Matrix(float32(f.Width) / float32(f.Height))
				layout := layoutAxes(f, f.Width, f.Height, 7, 13, reserve)
				names := map[string]bool{}
				for i, label := range layout.labels {
					if !label.rect.In(image.Rect(0, 0, f.Width, f.Height)) {
						t.Fatal("label outside viewport")
					}
					for _, r := range reserve {
						if label.rect.Overlaps(r) {
							t.Fatal("label overlaps legend")
						}
					}
					for _, other := range layout.labels[:i] {
						if label.rect.Overlaps(other.rect) {
							t.Fatal("labels overlap")
						}
					}
					names[label.text] = true
				}
				if !names["X"] || !names["Y"] || !names["Z"] {
					t.Errorf("missing names at %v/%g/%d: %v", projection, elevation, azimuth, names)
				}
			}
		}
	}
	f.Axes = [3]AxisFrame{{Hidden: true}, {Hidden: true}, {Hidden: true}}
	if layout := layoutAxes(f, 800, 600, 7, 13, nil); len(layout.lines) != 0 || len(layout.labels) != 0 {
		t.Fatal("hidden axes rendered")
	}
}

func TestAxisLayoutTinyAndDegenerate(t *testing.T) {
	f := axisTestFrame()
	for _, size := range [][2]int{{1, 1}, {12, 8}, {80, 24}, {320, 200}} {
		for _, distance := range []float64{.15, 3.4, 100} {
			c := DefaultCamera()
			c.Distance = distance
			f.Width, f.Height = size[0], size[1]
			f.Matrix = c.Matrix(float32(f.Width) / float32(f.Height))
			for _, rect := range axisRects(f) {
				if rect.w < 0 || rect.h < 0 {
					t.Fatal("invalid rectangle")
				}
			}
		}
	}
	f.Bounds = math3d.AABB{}
	f.Bounds.Include(math3d.Vec3{})
	n, _, _ := f.Bounds.Normalization()
	f.Matrix = DefaultCamera().Matrix(1.6).Mul(n)
	if len(axisRects(f)) == 0 {
		t.Fatal("constant data has no axes")
	}
}

func TestWireframeIncludesAxisText(t *testing.T) {
	f := axisTestFrame()
	f.Matrix = DefaultCamera().Matrix(2)
	view := wireRender(f, 100, 40)
	for _, name := range []string{"X", "Y", "Z"} {
		if !strings.Contains(view, name) {
			t.Errorf("missing %s in wireframe", name)
		}
	}
}
