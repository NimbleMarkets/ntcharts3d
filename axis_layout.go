// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"image"
	"image/color"
	"math"
	"unicode/utf8"

	"github.com/NimbleMarkets/ntcharts3d/math3d"
)

type axisPoint struct{ x, y float32 }
type axisSegment struct {
	a, b  axisPoint
	color color.RGBA
}
type axisText struct {
	text  string
	rect  image.Rectangle
	color color.RGBA
}
type axisLayout struct {
	lineWidth int // Zero uses one pixel.
	lines     []axisSegment
	labels    []axisText
}
type axisEdge struct {
	start, end    math3d.Vec3
	a, b, outward axisPoint
	valid         bool
}

var axisColors = [3]color.RGBA{{210, 65, 65, 255}, {50, 155, 85, 255}, {65, 105, 220, 255}}

// axisEdges selects silhouette edges, preferring bottom edges for X/Y and the
// left edge for Z. Selection depends only on the projected bounds.
func axisEdges(f Frame, width, height int) [3]axisEdge {
	var edges [3]axisEdge
	if !f.Bounds.IsValid() || width < 1 || height < 1 {
		return edges
	}
	lo, hi := components(f.Bounds.Min), components(f.Bounds.Max)
	extent := max(hi[0]-lo[0], hi[1]-lo[1], hi[2]-lo[2], .001)
	for i := range 3 {
		if hi[i] == lo[i] {
			lo[i] -= extent * .15
			hi[i] += extent * .15
		}
	}
	var corners [8]math3d.Vec3
	var screen [8]axisPoint
	var visible [8]bool
	var center axisPoint
	n := 0
	for i := range 8 {
		p := lo
		for dim := range 3 {
			if i&(1<<dim) != 0 {
				p[dim] = hi[dim]
			}
		}
		corners[i] = vector(p)
		screen[i], visible[i] = projectAxis(f.Matrix, corners[i], width, height)
		if visible[i] {
			center.x += screen[i].x
			center.y += screen[i].y
			n++
		}
	}
	if n == 0 {
		return edges
	}
	center.x /= float32(n)
	center.y /= float32(n)
	for dim := range 3 {
		best := float32(math.Inf(-1))
		for i := range 8 {
			if i&(1<<dim) != 0 {
				continue
			}
			j := i | (1 << dim)
			if !visible[i] || !visible[j] {
				continue
			}
			a, b := screen[i], screen[j]
			dx, dy := b.x-a.x, b.y-a.y
			length := float32(math.Hypot(float64(dx), float64(dy)))
			if length < 1 {
				continue
			}
			mid := axisPoint{(a.x + b.x) * .5, (a.y + b.y) * .5}
			normal := axisPoint{-dy / length, dx / length}
			if normal.x*(mid.x-center.x)+normal.y*(mid.y-center.y) < 0 {
				normal.x = -normal.x
				normal.y = -normal.y
			}
			outside := true
			for k, p := range screen {
				if visible[k] && normal.x*(p.x-mid.x)+normal.y*(p.y-mid.y) > 1 {
					outside = false
					break
				}
			}
			if !outside {
				continue
			}
			distance := normal.x*(mid.x-center.x) + normal.y*(mid.y-center.y)
			score := mid.y - center.y + distance*.25
			if dim == 2 {
				score = center.x - mid.x + distance*.25
			}
			if score > best {
				best = score
				edges[dim] = axisEdge{corners[i], corners[j], a, b, normal, true}
			}
		}
	}
	return edges
}

func projectAxis(matrix math3d.Mat4, p math3d.Vec3, width, height int) (axisPoint, bool) {
	x, y, z, w := matrix.Transform(p)
	if w <= 0 {
		return axisPoint{}, false
	}
	x, y, z = x/w, y/w, z/w
	if !validFloat(x, y, z) || z < 0 || z > 1 {
		return axisPoint{}, false
	}
	return axisPoint{(x + 1) * float32(width) * .5, (1 - y) * float32(height) * .5}, true
}

func layoutAxes(f Frame, width, height, charWidth, charHeight int, reserved []image.Rectangle) axisLayout {
	var out axisLayout
	if width < charWidth*3 || height < charHeight*2 {
		return out
	}
	edges := axisEdges(f, width, height)
	viewport := image.Rect(1, 1, width-1, height-1)
	occupied := append([]image.Rectangle(nil), reserved...)
	gap := float32(max(1, charHeight/3))
	place := func(text string, anchor, normal axisPoint, distance float32, ink color.RGBA) bool {
		if anchor.x < 0 || anchor.x > float32(width) || anchor.y < 0 || anchor.y > float32(height) {
			return false
		}
		text = displayText(text, max(1, min(32, (width-4)/charWidth)))
		if text == "" {
			return false
		}
		w, h := utf8.RuneCountInString(text)*charWidth, charHeight
		offset := distance + (abs(normal.x)*float32(w)+abs(normal.y)*float32(h))*.5
		x := int(math.Round(float64(anchor.x + normal.x*offset - float32(w)*.5)))
		y := int(math.Round(float64(anchor.y + normal.y*offset - float32(h)*.5)))
		// Small adjustments keep labels at the viewport edge without detaching them.
		xx, yy := max(viewport.Min.X, min(x, viewport.Max.X-w)), max(viewport.Min.Y, min(y, viewport.Max.Y-h))
		if abs(float32(xx-x)) > float32(charHeight*2) || abs(float32(yy-y)) > float32(charHeight*2) {
			return false
		}
		rect := image.Rect(xx, yy, xx+w, yy+h)
		if !rect.In(viewport) {
			return false
		}
		for _, other := range occupied {
			if rect.Inset(-1).Overlaps(other) {
				return false
			}
		}
		occupied = append(occupied, rect)
		out.labels = append(out.labels, axisText{text, rect, ink})
		return true
	}
	// Reserve space for all axis names before considering tick labels.
	for i, e := range edges {
		a := f.Axes[i]
		if a.Hidden || !e.valid {
			continue
		}
		out.lines = append(out.lines, axisSegment{e.a, e.b, axisColors[i]})
		if !a.HideName {
			for _, t := range []float32{.5, .25, .75} {
				p := e.start.Add(e.end.Sub(e.start).Scale(t))
				anchor, ok := projectAxis(f.Matrix, p, width, height)
				if ok && place(a.Name, anchor, e.outward, float32(charHeight)*2.2, axisColors[i]) {
					break
				}
			}
		}
	}
	for i, e := range edges {
		a := f.Axes[i]
		if a.Hidden || !e.valid {
			continue
		}
		// Endpoints get first choice of label space, followed by the interior ticks.
		order := make([]int, 0, len(a.Ticks))
		if len(a.Ticks) > 0 {
			order = append(order, 0)
		}
		if len(a.Ticks) > 1 {
			order = append(order, len(a.Ticks)-1)
		}
		for j := 1; j < len(a.Ticks)-1; j++ {
			order = append(order, j)
		}
		for _, j := range order {
			tick := a.Ticks[j]
			p := components(e.start)
			p[i] = tick.Value
			anchor, ok := projectAxis(f.Matrix, vector(p), width, height)
			if !ok {
				continue
			}
			if !a.HideTicks {
				tip := axisPoint{anchor.x + e.outward.x*gap, anchor.y + e.outward.y*gap}
				out.lines = append(out.lines, axisSegment{anchor, tip, axisColors[i]})
			}
			if !a.HideLabels {
				place(tick.Label, anchor, e.outward, gap*2, axisColors[i])
			}
		}
	}
	return out
}

// clipAxisSegment clips in screen space before rasterizing, keeping work bounded
// when zooming or panning pushes an endpoint far outside the viewport.
func clipAxisSegment(s axisSegment, width, height int) (axisSegment, bool) {
	dx, dy := s.b.x-s.a.x, s.b.y-s.a.y
	lo, hi := float32(0), float32(1)
	for _, p := range [][2]float32{{-dx, s.a.x}, {dx, float32(width-1) - s.a.x}, {-dy, s.a.y}, {dy, float32(height-1) - s.a.y}} {
		if p[0] == 0 {
			if p[1] < 0 {
				return s, false
			}
			continue
		}
		t := p[1] / p[0]
		if p[0] < 0 {
			lo = max(lo, t)
		} else {
			hi = min(hi, t)
		}
		if lo > hi {
			return s, false
		}
	}
	a := s.a
	s.a = axisPoint{a.x + lo*dx, a.y + lo*dy}
	s.b = axisPoint{a.x + hi*dx, a.y + hi*dy}
	return s, true
}

func axisRects(f Frame) []overlayRect {
	scale := 1
	if f.Height >= 360 {
		scale = 2
	}
	layout := layoutAxes(f, f.Width, f.Height, textWidth*scale, textHeight*scale, colorLegendPanels(f))
	return layoutRects(layout, f, scale)
}

func layoutRects(layout axisLayout, f Frame, scale int) []overlayRect {
	var out []overlayRect
	for _, s := range layout.lines {
		s, ok := clipAxisSegment(s, f.Width, f.Height)
		if !ok {
			continue
		}
		n := max(1, int(math.Ceil(float64(max(abs(s.b.x-s.a.x), abs(s.b.y-s.a.y))))))
		for j := 0; j <= n; j++ {
			t := float32(j) / float32(n)
			x, y := int(math.Round(float64(s.a.x+(s.b.x-s.a.x)*t))), int(math.Round(float64(s.a.y+(s.b.y-s.a.y)*t)))
			out = append(out, overlayRect{x, y, max(1, layout.lineWidth), max(1, layout.lineWidth), s.color})
		}
	}
	for _, l := range layout.labels {
		r := l.rect.Inset(-1)
		out = append(out, overlayRect{r.Min.X, r.Min.Y, r.Dx(), r.Dy(), f.Background})
		out = append(out, bitmapText(l.text, l.rect.Min.X, l.rect.Min.Y, l.color, scale)...)
	}
	return out
}
