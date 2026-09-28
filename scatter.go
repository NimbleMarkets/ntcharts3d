// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"fmt"
	"image/color"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts3d/math3d"
)

const MaxPoints = 500_000
const MaxPickPoints = 200_000

// Scatter holds X/Y/Z columns of equal length. SetScatter copies all slices.
// Optional columns must be empty or match that length. Color uses 0xRRGGBBAA;
// without it, the palette maps ColorValue (or Z). Size is the point radius in
// pixels, defaults to 2, and is clamped to [1,32] before rendering.
type Scatter struct {
	X, Y, Z          []float32
	Color            []uint32
	ColorValue, Size []float32
	Category         []int
	Labels           []string
}

func (m *Model) SetScatter(name string, d Scatter) tea.Cmd {
	d.X = slices.Clone(d.X)
	d.Y = slices.Clone(d.Y)
	d.Z = slices.Clone(d.Z)
	d.Color = slices.Clone(d.Color)
	d.ColorValue = slices.Clone(d.ColorValue)
	d.Size = slices.Clone(d.Size)
	d.Category = slices.Clone(d.Category)
	d.Labels = slices.Clone(d.Labels)
	return m.SetSeries(scatterSeries{name, d})
}

type scatterSeries struct {
	name string
	data Scatter
}

func (s scatterSeries) Name() string { return s.name }
func (s scatterSeries) Geometry(cs Palette) (Geometry, error) {
	return s.geometry(cs, nil)
}

func (s scatterSeries) GeometryWithColorDomain(cs Palette, lo, hi float32) (Geometry, error) {
	domain := [2]float32{lo, hi}
	return s.geometry(cs, &domain)
}

func (s scatterSeries) geometry(cs Palette, domain *[2]float32) (Geometry, error) {
	d := s.data
	n := len(d.X)
	if len(d.Y) != n || len(d.Z) != n {
		return Geometry{}, fmt.Errorf("ntcharts3d: X/Y/Z lengths differ")
	}
	if err := validColumns(n, len(d.Color), len(d.ColorValue), len(d.Size), len(d.Category), len(d.Labels)); err != nil {
		return Geometry{}, err
	}
	g := Geometry{Points: make([]Point, n), Labels: slices.Clone(d.Labels)}
	if len(g.Labels) == 0 && len(d.Category) > 0 {
		g.Labels = make([]string, n)
		for i, v := range d.Category {
			g.Labels[i] = fmt.Sprintf("category %d", v)
		}
	}
	values := d.ColorValue
	if len(values) == 0 {
		values = d.Z
	}
	lo, hi := valuesRange(values)
	if domain != nil {
		lo, hi = domain[0], domain[1]
	}
	if len(d.Color) == 0 {
		g.ColorMin, g.ColorMax, g.HasColorRange = lo, hi, true
	}
	for i := range n {
		p := math3d.Vec3{X: d.X[i], Y: d.Y[i], Z: d.Z[i]}
		radius := float32(2)
		if len(d.Size) > 0 {
			radius = d.Size[i]
		}
		if !validFloat(p.X, p.Y, p.Z, radius, values[i]) {
			return Geometry{}, fmt.Errorf("ntcharts3d: non-finite scatter datum %d", i)
		}
		c := mapped(cs, values[i], lo, hi)
		if len(d.Color) > 0 {
			v := d.Color[i]
			c = color.RGBA{uint8(v >> 24), uint8(v >> 16), uint8(v >> 8), uint8(v)}
		}
		g.Points[i] = Point{p, max(1, min(32, radius)), c, i}
		g.Bounds.Include(p)
	}
	return g, nil
}
