// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"math"

	"github.com/NimbleMarkets/ntcharts3d/math3d"
)

// PickMsg identifies a clicked datum. SeriesIndex is the zero-based series
// position at the time of the pick. SeriesName comes from the setter or Series.Name.
type PickMsg struct {
	ModelID     uint64
	SeriesIndex int
	SeriesName  string
	Datum       int
	Position    math3d.Vec3
	Label       string
}

// Pick finds a datum at cell coordinates relative to the plot, or returns nil.
// Points use projected radii with a minimum tolerance based on cell size.
// Above MaxPickPoints total points, only bars and surface vertices are tested.
// Bars use ray/box intersections; surfaces use nearby grid vertices.
func (m *Model) Pick(x, y int) *PickMsg {
	plotCols := m.plotWidth()
	if x < 0 || y < 0 || x >= plotCols || y >= m.plotHeight() {
		return nil
	}
	f := m.frame()
	px, py := (float32(x)+.5)*float32(f.Width)/float32(plotCols), (float32(y)+.5)*float32(f.Height)/float32(m.plotHeight())
	count := 0
	for _, s := range m.series {
		count += len(s.geometry.Points)
	}
	best := float32(math.Inf(1))
	var hit *PickMsg
	set := func(si, idx int, p math3d.Vec3, d float32) {
		if d >= best {
			return
		}
		best = d
		label := m.datumLabel(si, idx)
		hit = &PickMsg{ModelID: m.id, SeriesIndex: si, SeriesName: m.series[si].name, Datum: idx, Position: p, Label: label}
	}
	test := func(si, idx int, p math3d.Vec3, radius float32) {
		if m.clipsPlot() && !contains(f.Bounds, p) {
			return
		}
		sx, sy, depth, ok := f.Matrix.Project(p, f.Width, f.Height)
		if !ok {
			return
		}
		tolerance := max(radius, float32(f.Width)/float32(plotCols)*.6)
		dx, dy := sx-px, sy-py
		if dx*dx+dy*dy <= tolerance*tolerance {
			set(si, idx, p, depth)
		}
	}
	cw, ch := m.pic.CellPixelSize()
	r := m.camera.RayFromCell(x, y, plotCols, m.plotHeight(), cw, ch)
	r = m.plotTransform().inverseRay(r)
	for si, entry := range m.series {
		g := entry.geometry
		if count <= MaxPickPoints {
			for _, p := range g.Points {
				test(si, p.Datum, p.Position, p.Radius)
			}
		}
		for i, v := range g.Vertices {
			test(si, i, v.Position, 2)
		}
		for _, b := range g.Boxes {
			visibleBox := b
			if m.clipsPlot() {
				var ok bool
				visibleBox, ok = clipBox(b, f.Bounds)
				if !ok {
					continue
				}
			}
			if t, ok := rayBox(r, visibleBox); ok {
				p := r.Origin.Add(r.Direction.Scale(t))
				_, _, depth, visible := f.Matrix.Project(p, f.Width, f.Height)
				if visible {
					value := b.Min.Z + b.Size.Z
					if b.Min.Z < 0 {
						value = b.Min.Z
					}
					set(si, b.Datum, math3d.Vec3{X: b.Min.X + .4, Y: b.Min.Y + .4, Z: value}, depth)
				}
			}
		}
	}
	return hit
}

func rayBox(r math3d.Ray, b Box) (float32, bool) {
	lo, hi := float32(0), float32(math.Inf(1))
	o := []float32{r.Origin.X, r.Origin.Y, r.Origin.Z}
	d := []float32{r.Direction.X, r.Direction.Y, r.Direction.Z}
	a := []float32{b.Min.X, b.Min.Y, b.Min.Z}
	s := []float32{b.Size.X, b.Size.Y, b.Size.Z}
	for i := range 3 {
		if d[i] == 0 {
			if o[i] < a[i] || o[i] > a[i]+s[i] {
				return 0, false
			}
			continue
		}
		x, y := (a[i]-o[i])/d[i], (a[i]+s[i]-o[i])/d[i]
		if x > y {
			x, y = y, x
		}
		lo = max(lo, x)
		hi = min(hi, y)
		if hi < lo {
			return 0, false
		}
	}
	return lo, true
}
