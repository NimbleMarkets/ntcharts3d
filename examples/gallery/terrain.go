// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts3d"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const terrainName = "Cedar Island (synthetic map)"

// The map and mesh share local coordinates in kilometers. No tile server or
// API key is needed; real map imagery can replace these generated images.
func terrainHeight(x, y float64) float64 {
	ridge := .55 * math.Exp(-((x+.5)*(x+.5)/.6 + (y-.4)*(y-.4)/.5))
	peak := .8 * math.Exp(-((x-.55)*(x-.55)/.3 + (y-.15)*(y-.15)/.6))
	coast := .16 * (1 - x*x/2.3 - y*y/1.3)
	return math.Max(0, ridge+peak+coast-.07)
}

func (g *gallery) loadTerrain() tea.Cmd {
	if g.mapTextures[0] == nil {
		for i := range g.mapTextures {
			texture, err := ntcharts3d.NewTexture(terrainImage(i == 1))
			if err != nil {
				g.pick = err.Error()
				return nil
			}
			g.mapTextures[i] = texture
		}
	}
	axes := g.scene.SetAxes(ntcharts3d.Axes{
		X: ntcharts3d.Axis{Name: "East (km)"}, Y: ntcharts3d.Axis{Name: "North (km)"}, Z: ntcharts3d.Axis{Name: "Elevation (km)", Range: &ntcharts3d.Range{Min: 0, Max: 1}},
	})
	return tea.Batch(axes, g.setTerrain())
}

func (g *gallery) setTerrain() tea.Cmd {
	flat := g.flatMap
	return g.scene.SetSurface(terrainName, ntcharts3d.Surface{
		NX: 96, NY: 72, MinX: -2, MaxX: 2, MinY: -1.5, MaxY: 1.5,
		Sampler: func(x, y float64) float64 {
			if flat {
				return 0
			}
			return terrainHeight(x, y)
		},
		Material: &ntcharts3d.Material{Texture: g.mapTextures[g.mapStyle], Unlit: !g.mapLit},
	})
}

func (g *gallery) terrainCaption() string {
	shape, style, light := "terrain", "street", "unlit"
	if g.flatMap {
		shape = "flat"
	}
	if g.mapStyle == 1 {
		style = "topographic"
	}
	if g.mapLit {
		light = "lit"
	}
	return fmt.Sprintf("Cedar Island · %s · %s · %s | t image · f flat · l light", shape, style, light)
}

func terrainImage(topo bool) *image.RGBA {
	const w, h = 768, 576
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for py := range h {
		for px := range w {
			x, y := float64(px)/float64(w-1)*4-2, 1.5-float64(py)/float64(h-1)*3
			z := terrainHeight(x, y)
			c := color.RGBA{160, 207, 220, 255}
			if z > 0 {
				c = color.RGBA{224, 231, 199, 255}
				if topo {
					c = color.RGBA{uint8(204 - z*65), uint8(221 - z*60), uint8(172 - z*45), 255}
				}
				if z < .018 {
					c = color.RGBA{239, 224, 176, 255}
				}
				// A quiet street grid around the harbor.
				if !topo && y < -.15 && x > -.9 && x < .9 && (px%32 < 2 || py%32 < 2) {
					c = color.RGBA{250, 248, 234, 255}
				}
				if topo && math.Mod(z, .05) < .0028 {
					c = color.RGBA{122, 150, 105, 255}
				}
			}
			img.SetRGBA(px, py, c)
		}
	}
	pixel := func(x, y float64) image.Point {
		return image.Pt(int((x+2)/4*float64(w-1)), int((1.5-y)/3*float64(h-1)))
	}
	road := []image.Point{pixel(-1.25, -.45), pixel(-.7, -.65), pixel(-.1, -.5), pixel(.55, -.45), pixel(.95, -.1), pixel(.8, .45), pixel(.45, .8), pixel(-.3, .7), pixel(-.85, .25), pixel(-1.25, -.45)}
	mapPath(img, road, 7, color.RGBA{171, 151, 114, 255})
	mapPath(img, road, 4, color.RGBA{255, 243, 202, 255})
	trail := []image.Point{pixel(-.85, .25), pixel(-.5, .4), pixel(-.05, .3), pixel(.25, .15), pixel(.55, .15)}
	mapPath(img, trail, 2, color.RGBA{247, 247, 230, 255})
	mapLabel(img, pixel(-.65, .96), "CEDAR ISLAND", 2)
	mapLabel(img, pixel(-.75, -.85), "HARBOR", 2)
	mapLabel(img, pixel(-.95, .35), "NORTH RIDGE", 1)
	mapLabel(img, pixel(.25, .1), "SUMMIT", 1)
	mapLabel(img, image.Pt(30, 28), "N ^", 2)
	mapLabel(img, image.Pt(30, h-44), "SYNTHETIC / NOT OSM", 1)
	mapPath(img, []image.Point{image.Pt(w-140, h-36), image.Pt(w-44, h-36)}, 2, color.RGBA{42, 65, 66, 255})
	mapLabel(img, image.Pt(w-135, h-62), "500 m", 1)
	return img
}

func mapPath(img *image.RGBA, points []image.Point, radius int, c color.RGBA) {
	for i := 1; i < len(points); i++ {
		a, b := points[i-1], points[i]
		steps := max(1, max(absInt(b.X-a.X), absInt(b.Y-a.Y)))
		for j := 0; j <= steps; j++ {
			x, y := a.X+(b.X-a.X)*j/steps, a.Y+(b.Y-a.Y)*j/steps
			for dy := -radius; dy <= radius; dy++ {
				for dx := -radius; dx <= radius; dx++ {
					if dx*dx+dy*dy <= radius*radius {
						img.SetRGBA(x+dx, y+dy, c)
					}
				}
			}
		}
	}
}
func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func mapLabel(img *image.RGBA, p image.Point, text string, scale int) {
	label := image.NewRGBA(image.Rect(0, 0, len(text)*7+8, 19))
	draw.Draw(label, label.Bounds(), image.NewUniform(color.RGBA{250, 248, 235, 255}), image.Point{}, draw.Src)
	d := font.Drawer{Dst: label, Src: image.NewUniform(color.RGBA{36, 62, 59, 255}), Face: basicfont.Face7x13, Dot: fixed.P(4, 14)}
	d.DrawString(text)
	for y := 0; y < label.Bounds().Dy()*scale; y++ {
		for x := 0; x < label.Bounds().Dx()*scale; x++ {
			img.Set(p.X+x, p.Y+y, label.At(x/scale, y/scale))
		}
	}
}
