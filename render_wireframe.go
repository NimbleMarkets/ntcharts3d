// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"image/color"
	"math"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts3d/math3d"
)

func wireRender(f Frame, cols, rows int) string {
	c := canvas.New(cols, rows)
	drawLine := func(a, b math3d.Vec3, ink color.Color) {
		style := lipgloss.NewStyle()
		if ink != nil {
			style = style.Foreground(ink)
		}
		x, y, _, ok := f.Matrix.Project(a, cols, rows)
		xx, yy, _, ok2 := f.Matrix.Project(b, cols, rows)
		if !ok && !ok2 {
			return
		}
		n := min(4*max(cols, rows), max(1, int(max(abs(xx-x), abs(yy-y)))))
		for i := 0; i <= n; i++ {
			t := float32(i) / float32(n)
			c.SetRuneWithStyle(canvas.Point{X: int(x + (xx-x)*t), Y: int(y + (yy-y)*t)}, '·', style)
		}
	}
	line := func(a, b math3d.Vec3) { drawLine(a, b, nil) }
	for i := 0; i+1 < len(f.GridLines); i += 2 {
		drawLine(f.GridLines[i].Position, f.GridLines[i+1].Position, f.GridLines[i].Color)
	}
	for _, g := range f.Geometry {
		step := max(1, (len(g.Points)+1999)/2000)
		for i := 0; i < len(g.Points); i += step {
			p := g.Points[i]
			x, y, _, ok := f.Matrix.Project(p.Position, cols, rows)
			if ok {
				c.SetRune(canvas.Point{X: int(x), Y: int(y)}, '•')
			}
		}
		step = 3 * max(1, (len(g.Indices)/3+1999)/2000)
		for i := 0; i+2 < len(g.Indices); i += step {
			a, b, cc := g.Vertices[g.Indices[i]].Position, g.Vertices[g.Indices[i+1]].Position, g.Vertices[g.Indices[i+2]].Position
			line(a, b)
			line(b, cc)
			line(cc, a)
		}
		for i, b := range g.Boxes {
			if i >= 500 {
				break
			}
			v := boxVertices(b)
			for j := 0; j < len(v); j += 3 {
				line(v[j].Position, v[j+1].Position)
				line(v[j+1].Position, v[j+2].Position)
			}
		}
		for i := 0; i+1 < len(g.Lines); i += 2 {
			line(g.Lines[i].Position, g.Lines[i+1].Position)
		}
	}
	layout := layoutAxes(f, cols, rows, 1, 1, nil)
	emphasis := emphasisLayout(f, cols, rows, 1, 1)
	layout.lines = append(layout.lines, emphasis.lines...)
	layout.labels = append(layout.labels, emphasis.labels...)
	for _, segment := range layout.lines {
		segment, ok := clipAxisSegment(segment, cols, rows)
		if !ok {
			continue
		}
		n := max(1, int(math.Ceil(float64(max(abs(segment.b.x-segment.a.x), abs(segment.b.y-segment.a.y))))))
		style := lipgloss.NewStyle().Foreground(segment.color)
		for i := 0; i <= n; i++ {
			t := float32(i) / float32(n)
			x, y := int(math.Round(float64(segment.a.x+(segment.b.x-segment.a.x)*t))), int(math.Round(float64(segment.a.y+(segment.b.y-segment.a.y)*t)))
			c.SetRuneWithStyle(canvas.Point{X: x, Y: y}, '·', style)
		}
	}
	for _, label := range layout.labels {
		c.SetStringWithStyle(canvas.Point{X: label.rect.Min.X, Y: label.rect.Min.Y}, label.text, lipgloss.NewStyle().Foreground(label.color))
	}
	return c.View()
}
