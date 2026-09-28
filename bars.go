// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"fmt"
	"math"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts3d/math3d"
)

// Bars holds a category grid stored by row. Values are signed heights from zero;
// Axis.Categories names the columns and rows. SetBars copies all slices.
type Bars struct {
	NX, NY             int
	Values, ColorValue []float32
}

func (m *Model) SetBars(name string, d Bars) tea.Cmd {
	d.Values = slices.Clone(d.Values)
	d.ColorValue = slices.Clone(d.ColorValue)
	return m.SetSeries(barSeries{name, d})
}

type barSeries struct {
	name string
	data Bars
}

func (s barSeries) Name() string { return s.name }

func (s barSeries) Geometry(cs Palette) (Geometry, error) {
	return s.geometry(cs, nil)
}

func (s barSeries) GeometryWithColorDomain(cs Palette, lo, hi float32) (Geometry, error) {
	domain := [2]float32{lo, hi}
	return s.geometry(cs, &domain)
}

func (s barSeries) geometry(cs Palette, domain *[2]float32) (Geometry, error) {
	d := s.data
	if d.NX < 1 || d.NY < 1 || d.NX > 512 || d.NY > 512 || len(d.Values) != d.NX*d.NY {
		return Geometry{}, fmt.Errorf("ntcharts3d: bar grid length/dimensions invalid")
	}
	if err := validColumns(len(d.Values), len(d.ColorValue)); err != nil {
		return Geometry{}, err
	}
	vals := d.ColorValue
	if len(vals) == 0 {
		vals = d.Values
	}
	lo, hi := valuesRange(vals)
	if domain != nil {
		lo, hi = domain[0], domain[1]
	}
	g := Geometry{ColorMin: lo, ColorMax: hi, HasColorRange: true}
	for i, v := range d.Values {
		if !validFloat(v, vals[i]) {
			return Geometry{}, fmt.Errorf("ntcharts3d: non-finite bar datum")
		}
		p := math3d.Vec3{X: float32(i%d.NX) - .4, Y: float32(i/d.NX) - .4, Z: min(0, v)}
		size := math3d.Vec3{X: .8, Y: .8, Z: float32(math.Abs(float64(v)))}
		g.Boxes = append(g.Boxes, Box{p, size, mapped(cs, vals[i], lo, hi), i})
		g.Bounds.Include(p)
		g.Bounds.Include(p.Add(size))
	}
	return g, nil
}
