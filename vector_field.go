// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"fmt"
	"image/color"
	"math"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts3d/math3d"
)

const MaxVectors = 100_000

// Arrow is an unlit, screen-facing shaft and triangular head between two world
// positions. Width is in render pixels (zero defaults to 1). HeadSize is the
// head length in pixels, 0..64; zero draws only the shaft. Short projections
// shrink the head to at most 45% of the projected length. End-on arrows appear
// as a square of Width pixels. Datum identifies the original vector for picking.
type Arrow struct {
	Start, End      math3d.Vec3
	Width, HeadSize float32
	Color           color.RGBA
	Datum           int
}

// VectorField pairs Origins with Vectors. The tip is origin + vector * Scale.
// Scale defaults to 1 and must be positive. Normalize makes every nonzero vector
// Scale world units long; colors still use the original magnitude. Zero vectors
// have no arrow. Width defaults to 2 pixels, HeadSize to 8 (range 1..64).
// Optional Color (0xRRGGBBAA), ColorValue, and Labels have one entry per vector.
// Without Color, the palette maps ColorValue or the original vector magnitude.
type VectorField struct {
	Origins, Vectors []math3d.Vec3
	Scale            float32
	Normalize        bool
	Width, HeadSize  float32
	Color            []uint32
	ColorValue       []float32
	Labels           []string
}

// SetVectorField copies all slices and compiles GPU arrow instances. Camera
// changes do not rebuild or upload them. Return the command from Update.
func (m *Model) SetVectorField(name string, d VectorField) tea.Cmd {
	d.Origins = slices.Clone(d.Origins)
	d.Vectors = slices.Clone(d.Vectors)
	d.Color = slices.Clone(d.Color)
	d.ColorValue = slices.Clone(d.ColorValue)
	d.Labels = slices.Clone(d.Labels)
	return m.SetSeries(vectorSeries{name, d})
}

type vectorSeries struct {
	name string
	data VectorField
}

func (s vectorSeries) Name() string                         { return s.name }
func (s vectorSeries) Geometry(p Palette) (Geometry, error) { return s.geometry(p, nil) }
func (s vectorSeries) GeometryWithColorDomain(p Palette, lo, hi float32) (Geometry, error) {
	return s.geometry(p, &[2]float32{lo, hi})
}
func (s vectorSeries) geometry(p Palette, domain *[2]float32) (Geometry, error) {
	d := s.data
	n := len(d.Origins)
	if n > MaxVectors || len(d.Vectors) != n {
		return Geometry{}, fmt.Errorf("ntcharts3d: invalid vector count (maximum %d)", MaxVectors)
	}
	if err := validColumns(n, len(d.Color), len(d.ColorValue), len(d.Labels)); err != nil {
		return Geometry{}, err
	}
	if !validFloat(d.Scale, d.HeadSize) || d.Scale < 0 || !validWidth(d.Width) || d.HeadSize < 0 || d.HeadSize > 64 || d.HeadSize > 0 && d.HeadSize < 1 {
		return Geometry{}, fmt.Errorf("ntcharts3d: invalid vector scale, width, or head size")
	}
	if d.Scale == 0 {
		d.Scale = 1
	}
	if d.Width == 0 {
		d.Width = 2
	}
	if d.HeadSize == 0 {
		d.HeadSize = 8
	}
	values := make([]float32, n)
	for i, v := range d.Vectors {
		if !finiteVec(v) || !finiteVec(d.Origins[i]) {
			return Geometry{}, fmt.Errorf("ntcharts3d: non-finite vector %d", i)
		}
		values[i] = float32(math.Sqrt(float64(v.X)*float64(v.X) + float64(v.Y)*float64(v.Y) + float64(v.Z)*float64(v.Z)))
		if len(d.ColorValue) > 0 {
			values[i] = d.ColorValue[i]
		}
		if !validFloat(values[i]) {
			return Geometry{}, fmt.Errorf("ntcharts3d: non-finite vector magnitude or color %d", i)
		}
	}
	lo, hi := valuesRange(values)
	if domain != nil {
		lo, hi = domain[0], domain[1]
	}
	g := Geometry{Arrows: make([]Arrow, 0, n), Labels: slices.Clone(d.Labels), ColorMin: lo, ColorMax: hi, HasColorRange: len(d.Color) == 0}
	for i, v := range d.Vectors {
		origin := d.Origins[i]
		g.Bounds.Include(origin)
		if v == (math3d.Vec3{}) {
			continue
		}
		if d.Normalize {
			v = v.Normalize()
		}
		tip := origin.Add(v.Scale(d.Scale))
		if !finiteVec(tip) {
			return Geometry{}, fmt.Errorf("ntcharts3d: vector tip overflow at %d", i)
		}
		c := mapped(p, values[i], lo, hi)
		if len(d.Color) > 0 {
			c = packedColor(d.Color[i])
		}
		g.Arrows = append(g.Arrows, Arrow{Start: origin, End: tip, Width: d.Width, HeadSize: d.HeadSize, Color: c, Datum: i})
		g.Bounds.Include(tip)
	}
	return g, nil
}
