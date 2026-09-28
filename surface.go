// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"fmt"
	"image/color"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts3d/math3d"
)

// Surface is a regular XY grid stored by row. Sampler overrides Z and runs
// when geometry is compiled; an update may sample the grid more than once.
// Equal bounds on either horizontal axis default to [-1,1].
type Surface struct {
	NX, NY                 int
	Z                      []float32
	Sampler                func(float64, float64) float64
	MinX, MaxX, MinY, MaxY float64
	Wireframe              bool
}

func (m *Model) SetSurface(name string, d Surface) tea.Cmd {
	d.Z = slices.Clone(d.Z)
	return m.SetSeries(surfaceSeries{name, d})
}

type surfaceSeries struct {
	name string
	data Surface
}

func (s surfaceSeries) Name() string { return s.name }
func (s surfaceSeries) Geometry(cs Palette) (Geometry, error) {
	return s.geometry(cs, nil)
}

func (s surfaceSeries) GeometryWithColorDomain(cs Palette, lo, hi float32) (Geometry, error) {
	domain := [2]float32{lo, hi}
	return s.geometry(cs, &domain)
}

func (s surfaceSeries) geometry(cs Palette, domain *[2]float32) (Geometry, error) {
	d := s.data
	if d.NX < 2 || d.NY < 2 || d.NX > 512 || d.NY > 512 {
		return Geometry{}, fmt.Errorf("ntcharts3d: surface dimensions must be 2..512")
	}
	n := d.NX * d.NY
	if d.Sampler == nil && len(d.Z) != n {
		return Geometry{}, fmt.Errorf("ntcharts3d: surface grid length mismatch")
	}
	if d.MinX == d.MaxX {
		d.MinX, d.MaxX = -1, 1
	}
	if d.MinY == d.MaxY {
		d.MinY, d.MaxY = -1, 1
	}
	if !math3d.IsFinite(d.MinX) || !math3d.IsFinite(d.MaxX) || !math3d.IsFinite(d.MinY) || !math3d.IsFinite(d.MaxY) || d.MaxX <= d.MinX || d.MaxY <= d.MinY {
		return Geometry{}, fmt.Errorf("ntcharts3d: invalid surface bounds")
	}
	g := Geometry{Vertices: make([]Vertex, n)}
	zs := make([]float32, n)
	for y := range d.NY {
		for x := range d.NX {
			i := y*d.NX + x
			px := d.MinX + float64(x)*(d.MaxX-d.MinX)/float64(d.NX-1)
			py := d.MinY + float64(y)*(d.MaxY-d.MinY)/float64(d.NY-1)
			if d.Sampler != nil {
				zs[i] = float32(d.Sampler(px, py))
			} else {
				zs[i] = d.Z[i]
			}
			if !validFloat(zs[i]) {
				return Geometry{}, fmt.Errorf("ntcharts3d: non-finite surface datum %d", i)
			}
			p := math3d.Vec3{X: float32(px), Y: float32(py), Z: zs[i]}
			g.Vertices[i].Position = p
			g.Bounds.Include(p)
		}
	}
	for y := 0; y < d.NY-1; y++ {
		for x := 0; x < d.NX-1; x++ {
			a := uint32(y*d.NX + x)
			b := a + 1
			c := a + uint32(d.NX)
			dd := c + 1
			g.Indices = append(g.Indices, a, b, c, b, dd, c)
		}
	}
	for i := 0; i < len(g.Indices); i += 3 {
		a, b, c := g.Indices[i], g.Indices[i+1], g.Indices[i+2]
		normal := g.Vertices[b].Position.Sub(g.Vertices[a].Position).Cross(g.Vertices[c].Position.Sub(g.Vertices[a].Position))
		for _, j := range []uint32{a, b, c} {
			g.Vertices[j].Normal = g.Vertices[j].Normal.Add(normal)
		}
	}
	lo, hi := valuesRange(zs)
	if domain != nil {
		lo, hi = domain[0], domain[1]
	}
	g.ColorMin, g.ColorMax, g.HasColorRange = lo, hi, true
	for i := range g.Vertices {
		g.Vertices[i].Normal = g.Vertices[i].Normal.Normalize()
		g.Vertices[i].Color = mapped(cs, zs[i], lo, hi)
	}
	if d.Wireframe {
		for y := range d.NY {
			for x := range d.NX {
				i := y*d.NX + x
				for _, j := range []int{i + 1, i + d.NX} {
					if (j == i+1 && x+1 == d.NX) || j >= n {
						continue
					}
					a, b := g.Vertices[i], g.Vertices[j]
					a.Color = color.RGBA{40, 45, 50, 255}
					b.Color = a.Color
					g.Lines = append(g.Lines, a, b)
				}
			}
		}
	}
	return g, nil
}
