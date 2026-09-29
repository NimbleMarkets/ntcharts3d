// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package main

import (
	"fmt"
	"math"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts3d"
	"github.com/NimbleMarkets/ntcharts3d/math3d"
)

// A steady swirling flow with upward motion. Streamlines below are analytic
// integral curves of this same vector field, not unrelated decorative lines.
func swirl(p math3d.Vec3) math3d.Vec3 {
	r := math.Hypot(float64(p.X), float64(p.Y))
	return math3d.Vec3{X: -p.Y, Y: p.X, Z: float32(.3 + .12*math.Cos(2*r))}
}
func (g *gallery) loadVectorField() tea.Cmd {
	axes := g.scene.SetAxes(ntcharts3d.Axes{
		X: ntcharts3d.Axis{Name: "X", Range: &ntcharts3d.Range{Min: -1.5, Max: 1.5}},
		Y: ntcharts3d.Axis{Name: "Y", Range: &ntcharts3d.Range{Min: -1.5, Max: 1.5}},
		Z: ntcharts3d.Axis{Name: "Z", Range: &ntcharts3d.Range{Min: -1, Max: 1.4}},
	})
	domain := g.scene.SetColorDomain(&ntcharts3d.Range{Min: 0, Max: 1.4})
	return tea.Batch(axes, domain, g.setVectorField(), g.setStreamlines())
}
func (g *gallery) setVectorField() tea.Cmd {
	width := float32(2)
	if g.fieldWide {
		width = 4
	}
	d := ntcharts3d.VectorField{Scale: .3, Normalize: g.fieldNormalized, Width: width, HeadSize: 9}
	for iz := range 3 {
		for iy := range 7 {
			for ix := range 7 {
				p := math3d.Vec3{X: float32(ix-3) * .3, Y: float32(iy-3) * .3, Z: float32(iz-1) * .7}
				v := swirl(p)
				d.Origins = append(d.Origins, p)
				d.Vectors = append(d.Vectors, v)
				magnitude := math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z))
				d.Labels = append(d.Labels, fmt.Sprintf("v=(%.2f, %.2f, %.2f), |v|=%.2f", v.X, v.Y, v.Z, magnitude))
			}
		}
	}
	return g.scene.SetVectorField("flow speed", d)
}
func (g *gallery) setStreamlines() tea.Cmd {
	if g.fieldHideLines {
		return nil
	}
	width := float32(1)
	if g.fieldWide {
		width = 4
	}
	d := ntcharts3d.Lines{Width: width}
	for seed := range 6 {
		radius := .5 + float64(seed%2)*.45
		phase := float64(seed) * math.Pi / 3
		lift := float64(swirl(math3d.Vec3{X: float32(radius)}).Z)
		at := func(t float64) math3d.Vec3 {
			return math3d.Vec3{X: float32(radius * math.Cos(phase+t)), Y: float32(radius * math.Sin(phase+t)), Z: float32(-.95 + lift*t)}
		}
		for step := range 80 {
			a, b := at(float64(step)*.08), at(float64(step+1)*.08)
			d.Start = append(d.Start, a)
			d.End = append(d.End, b)
			d.Color = append(d.Color, 0x64748bff)
			d.Labels = append(d.Labels, fmt.Sprintf("streamline %d", seed+1))
		}
	}
	return g.scene.SetLines("streamlines", d)
}
func (g *gallery) vectorCaption() string {
	lengths := "magnitude"
	if g.fieldNormalized {
		lengths = "normalized"
	}
	return fmt.Sprintf("Swirling flow · %s lengths | n normalize · s streamlines · w width", lengths)
}
