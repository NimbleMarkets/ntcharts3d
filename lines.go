// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"fmt"
	"image/color"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts3d/math3d"
)

const MaxLineSegments = 512 * 512

// Lines contains independent Start/End pairs. Width is in render pixels (1..32);
// zero defaults to 1. Segments have butt caps and no joins. Repeat adjacent
// endpoints to form a polyline. Optional columns have one entry per segment.
// Color uses 0xRRGGBBAA (alpha ignored). Without Color, the palette maps
// ColorValue, or each segment's midpoint Z when ColorValue is absent.
type Lines struct {
	Start, End []math3d.Vec3
	Width      float32
	Color      []uint32
	ColorValue []float32
	Labels     []string
}

// SetLines copies all slices. Invalid input leaves the existing series intact.
func (m *Model) SetLines(name string, d Lines) tea.Cmd {
	d.Start = slices.Clone(d.Start)
	d.End = slices.Clone(d.End)
	d.Color = slices.Clone(d.Color)
	d.ColorValue = slices.Clone(d.ColorValue)
	d.Labels = slices.Clone(d.Labels)
	return m.SetSeries(lineSeries{name, d})
}

type lineSeries struct {
	name string
	data Lines
}

func (s lineSeries) Name() string                         { return s.name }
func (s lineSeries) Geometry(p Palette) (Geometry, error) { return s.geometry(p, nil) }
func (s lineSeries) GeometryWithColorDomain(p Palette, lo, hi float32) (Geometry, error) {
	return s.geometry(p, &[2]float32{lo, hi})
}
func (s lineSeries) geometry(p Palette, domain *[2]float32) (Geometry, error) {
	d := s.data
	n := len(d.Start)
	if n > MaxLineSegments || len(d.End) != n {
		return Geometry{}, fmt.Errorf("ntcharts3d: invalid line endpoint count")
	}
	if err := validColumns(n, len(d.Color), len(d.ColorValue), len(d.Labels)); err != nil {
		return Geometry{}, err
	}
	if !validWidth(d.Width) {
		return Geometry{}, fmt.Errorf("ntcharts3d: line width must be 1..32 or zero")
	}
	values := make([]float32, n)
	for i := range n {
		a, b := d.Start[i], d.End[i]
		if !finiteVec(a) || !finiteVec(b) {
			return Geometry{}, fmt.Errorf("ntcharts3d: non-finite line %d", i)
		}
		values[i] = float32((float64(a.Z) + float64(b.Z)) * .5)
		if len(d.ColorValue) > 0 {
			values[i] = d.ColorValue[i]
		}
		if !validFloat(values[i]) {
			return Geometry{}, fmt.Errorf("ntcharts3d: non-finite line color %d", i)
		}
	}
	lo, hi := valuesRange(values)
	if domain != nil {
		lo, hi = domain[0], domain[1]
	}
	g := Geometry{LineWidth: lineWidth(d.Width), Lines: make([]Vertex, 0, 2*n), Labels: slices.Clone(d.Labels), ColorMin: lo, ColorMax: hi, HasColorRange: len(d.Color) == 0}
	for i := range n {
		c := mapped(p, values[i], lo, hi)
		if len(d.Color) > 0 {
			c = packedColor(d.Color[i])
		}
		g.Lines = append(g.Lines, Vertex{Position: d.Start[i], Color: c}, Vertex{Position: d.End[i], Color: c})
		g.Bounds.Include(d.Start[i])
		g.Bounds.Include(d.End[i])
	}
	return g, nil
}
func finiteVec(v math3d.Vec3) bool { return validFloat(v.X, v.Y, v.Z) }
func validWidth(w float32) bool    { return validFloat(w) && (w == 0 || w >= 1 && w <= 32) }
func lineWidth(w float32) float32 {
	if w == 0 {
		return 1
	}
	return w
}
func packedColor(v uint32) color.RGBA {
	return color.RGBA{uint8(v >> 24), uint8(v >> 16), uint8(v >> 8), 255}
}
