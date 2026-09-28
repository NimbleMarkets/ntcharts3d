// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"github.com/NimbleMarkets/ntcharts3d/math3d"
	"math"
	"slices"
)

// Range is a fixed numeric interval. Min and Max must be finite, with Min < Max.
type Range struct{ Min, Max float32 }

func (r Range) valid() bool { return validFloat(r.Min, r.Max) && r.Min < r.Max }
func cloneRange(r *Range) *Range {
	if r == nil {
		return nil
	}
	c := *r
	return &c
}

// WithColorDomain sets a fixed domain for all palette-mapped series.
func WithColorDomain(r Range) Option { return func(m *Model) { m.SetColorDomain(&r) } }

// SetColorDomain fixes the palette domain and clamps values to its endpoints.
// Nil restores automatic domains, respecting SetSharedColorDomain.
// Custom mapped series must implement ColorDomainSeries. The range is copied.
func (m *Model) SetColorDomain(r *Range) tea.Cmd {
	if r != nil && !r.valid() {
		m.err = fmt.Errorf("ntcharts3d: invalid color domain")
		return nil
	}
	ss := slices.Clone(m.series)
	if err := compileColors(ss, m.colors, m.sharedColorDomain, r); err != nil {
		m.err = err
		return nil
	}
	m.colorDomain, m.series, m.err = cloneRange(r), ss, nil
	return m.changed(true)
}

// ColorDomain returns a copy of the fixed domain, or nil for automatic domains.
func (m *Model) ColorDomain() *Range { return cloneRange(m.colorDomain) }

// WithPlotAspect sets relative X, Y, Z plot lengths. Zero preserves data proportions.
func WithPlotAspect(v math3d.Vec3) Option { return func(m *Model) { m.SetPlotAspect(v) } }

// SetPlotAspect sets relative plot lengths, for example (1, 1, .5).
// All components must be positive and finite, or all zero to preserve data proportions.
func (m *Model) SetPlotAspect(v math3d.Vec3) tea.Cmd {
	if v != (math3d.Vec3{}) && (!validFloat(v.X, v.Y, v.Z) || min(v.X, v.Y, v.Z) <= 0 || min(v.X, v.Y, v.Z)/max(v.X, v.Y, v.Z) < 1e-6) {
		m.err = fmt.Errorf("ntcharts3d: plot aspect components must be positive with a ratio of at most 1,000,000, or all zero")
		return nil
	}
	m.plotAspect, m.err = v, nil
	return m.changed(true)
}

// PlotAspect returns the configured relative plot lengths.
func (m *Model) PlotAspect() math3d.Vec3 { return m.plotAspect }

// PlotBounds returns data bounds with fixed axis ranges applied. Bounds returns the data bounds.
func (m *Model) PlotBounds() math3d.AABB {
	b := m.bounds()
	a := m.axes.list()
	if !b.IsValid() && (a[0].Range == nil || a[1].Range == nil || a[2].Range == nil) {
		return b
	}
	lo, hi := components(b.Min), components(b.Max)
	for i, axis := range a {
		if axis.Range != nil {
			lo[i], hi[i] = axis.Range.Min, axis.Range.Max
		}
	}
	b.Include(vector(lo))
	b.Include(vector(hi))
	b.Min, b.Max = vector(lo), vector(hi)
	return b
}
func (m *Model) clipsPlot() bool {
	return m.axes.X.Range != nil || m.axes.Y.Range != nil || m.axes.Z.Range != nil
}

type plotTransform struct {
	matrix        math3d.Mat4
	center, scale math3d.Vec3
}

func (m *Model) plotTransform() plotTransform {
	b := m.PlotBounds()
	t := plotTransform{matrix: math3d.Identity(), scale: math3d.Vec3{X: 1, Y: 1, Z: 1}}
	if !b.IsValid() {
		return t
	}
	lo, hi, aspect := components(b.Min), components(b.Max), components(m.plotAspect)
	var center, scale [3]float32
	var extent [3]float64
	for i := range 3 {
		center[i] = float32((float64(lo[i]) + float64(hi[i])) * .5)
		extent[i] = math.Max(.001, float64(hi[i])-float64(lo[i]))
	}
	longest := max(extent[0], extent[1], extent[2])
	for i := range 3 {
		s := 2 / longest
		if m.plotAspect != (math3d.Vec3{}) {
			extent := extent[i]
			if hi[i] == lo[i] {
				extent = longest
			}
			s = 2 * (float64(aspect[i]) / float64(max(aspect[0], aspect[1], aspect[2]))) / extent
		}
		scale[i] = float32(s)
		t.matrix[i*5] = scale[i]
		t.matrix[12+i] = float32(-float64(center[i]) * s)
	}
	t.center, t.scale = vector(center), vector(scale)
	return t
}
func (t plotTransform) inverseRay(r math3d.Ray) math3d.Ray {
	o, d, c, s := components(r.Origin), components(r.Direction), components(t.center), components(t.scale)
	for i := range 3 {
		o[i] = float32(float64(c[i]) + float64(o[i])/float64(s[i]))
		d[i] /= s[i]
	}
	return math3d.Ray{Origin: vector(o), Direction: vector(d)}
}
func (t plotTransform) normal(n math3d.Vec3) math3d.Vec3 {
	// The inverse transpose of a diagonal scale divides each normal component.
	x, y, z := float64(n.X)/float64(t.scale.X), float64(n.Y)/float64(t.scale.Y), float64(n.Z)/float64(t.scale.Z)
	length := math.Sqrt(x*x + y*y + z*z)
	if length == 0 {
		return n
	}
	return math3d.Vec3{X: float32(x / length), Y: float32(y / length), Z: float32(z / length)}
}

func (m *Model) preparedGeometry() []Geometry {
	if m.preparedRevision == m.revision && m.prepared != nil {
		return m.prepared
	}
	out := make([]Geometry, len(m.series))
	transform := m.plotTransform()
	for i, s := range m.series {
		g := s.geometry
		if m.clipsPlot() {
			g = clipGeometry(g, m.PlotBounds())
		}
		if m.plotAspect != (math3d.Vec3{}) {
			g.Vertices = slices.Clone(g.Vertices)
			for j := range g.Vertices {
				g.Vertices[j].Normal = transform.normal(g.Vertices[j].Normal)
			}
		}
		out[i] = g
	}
	m.prepared, m.preparedRevision = out, m.revision
	return out
}
