// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"github.com/NimbleMarkets/ntcharts3d/math3d"
	"image"
	"image/color"
	"math"
	"unicode/utf8"
)

// Emphasis controls the foreground highlight for the hovered datum.
// The zero value draws a point ring or bar outline without a label.
type Emphasis struct {
	Hidden    bool
	ShowLabel bool
	Color     color.Color // Nil uses orange.
}

// EmphasisFrame describes a resolved hover highlight. Box is nil for a point ring.
type EmphasisFrame struct {
	Position math3d.Vec3
	Radius   float32 // Point radius in pixels, before the ring's padding.
	Box      *Box
	Label    string
	Color    color.RGBA
}

// WithEmphasis sets the initial hover highlight settings.
func WithEmphasis(e Emphasis) Option { return func(m *Model) { m.SetEmphasis(e) } }

// SetEmphasis replaces the hover highlight settings.
func (m *Model) SetEmphasis(e Emphasis) tea.Cmd {
	if e.Color != nil {
		e.Color = color.RGBAModel.Convert(e.Color)
	}
	m.emphasis = e
	return m.changed(false)
}

// Emphasis returns the current hover highlight settings.
func (m *Model) Emphasis() Emphasis { return m.emphasis }

func (m *Model) setHover(hit *PickMsg) tea.Cmd {
	if (hit == nil && m.hover == nil) || (hit != nil && m.hover != nil && *hit == *m.hover) {
		return nil
	}
	m.hover = hit
	return m.changed(false)
}
func (m *Model) emphasisFrame(f Frame) *EmphasisFrame {
	if m.emphasis.Hidden || m.hover == nil {
		return nil
	}
	hit := m.hover
	if hit.SeriesIndex < 0 || hit.SeriesIndex >= len(m.series) {
		return nil
	}
	e := &EmphasisFrame{Position: hit.Position, Radius: 2, Color: color.RGBA{R: 230, G: 85, B: 35, A: 255}}
	if m.emphasis.Color != nil {
		e.Color = color.RGBAModel.Convert(m.emphasis.Color).(color.RGBA)
	}
	g := m.series[hit.SeriesIndex].geometry
	for _, p := range g.Points {
		if p.Datum == hit.Datum {
			e.Position, e.Radius = p.Position, p.Radius
			break
		}
	}
	for _, b := range g.Boxes {
		if b.Datum == hit.Datum {
			if m.clipsPlot() {
				var ok bool
				b, ok = clipBox(b, f.Bounds)
				if !ok {
					return nil
				}
			}
			e.Box = &b
			e.Position = b.Min.Add(b.Size.Scale(.5))
			break
		}
	}
	if m.clipsPlot() && !contains(f.Bounds, e.Position) {
		return nil
	}
	if m.emphasis.ShowLabel {
		label := hit.Label
		if label == "" {
			label = fmt.Sprintf("#%d", hit.Datum)
		}
		e.Label = fmt.Sprintf("%s: %s (%.3g, %.3g, %.3g)", hit.SeriesName, label, hit.Position.X, hit.Position.Y, hit.Position.Z)
	}
	return e
}

func emphasisLayout(f Frame, width, height, charWidth, charHeight int) axisLayout {
	out := axisLayout{lineWidth: 2}
	e := f.Emphasis
	if e == nil {
		return out
	}
	anchor, ok := projectAxis(f.Matrix, e.Position, width, height)
	if !ok {
		return out
	}
	if e.Box != nil {
		lo, hi := components(e.Box.Min), components(e.Box.Min.Add(e.Box.Size))
		for corner := 0; corner < 8; corner++ {
			for axis := 0; axis < 3; axis++ {
				if corner&(1<<axis) != 0 {
					continue
				}
				a, b := lo, lo
				for dim := 0; dim < 3; dim++ {
					if corner&(1<<dim) != 0 {
						a[dim], b[dim] = hi[dim], hi[dim]
					}
				}
				b[axis] = hi[axis]
				pa, oka := projectAxis(f.Matrix, vector(a), width, height)
				pb, okb := projectAxis(f.Matrix, vector(b), width, height)
				if oka && okb {
					out.lines = append(out.lines, axisSegment{pa, pb, e.Color})
				}
			}
		}
	} else {
		rx := max(float32(1), (e.Radius+3)*float32(width)/float32(max(1, f.Width)))
		ry := max(float32(1), (e.Radius+3)*float32(height)/float32(max(1, f.Height)))
		for i := 0; i < 32; i++ {
			a, b := float64(i)*2*math.Pi/32, float64(i+1)*2*math.Pi/32
			out.lines = append(out.lines, axisSegment{axisPoint{anchor.x + rx*float32(math.Cos(a)), anchor.y + ry*float32(math.Sin(a))}, axisPoint{anchor.x + rx*float32(math.Cos(b)), anchor.y + ry*float32(math.Sin(b))}, e.Color})
		}
	}
	if e.Label == "" || width < charWidth*4 || height < charHeight*2 {
		return out
	}
	text := displayText(e.Label, min(64, (width-4)/charWidth))
	w, h := utf8.RuneCountInString(text)*charWidth, charHeight
	occupied := colorLegendPanels(f)
	if width != f.Width || height != f.Height {
		occupied = nil
	}
	for _, l := range layoutAxes(f, width, height, charWidth, charHeight, occupied).labels {
		occupied = append(occupied, l.rect)
	}
	for _, offset := range [][2]int{{charWidth, -h - 4}, {charWidth, 4}, {-w - charWidth, -h - 4}, {-w - charWidth, 4}} {
		x, y := max(2, min(int(anchor.x)+offset[0], width-w-2)), max(2, min(int(anchor.y)+offset[1], height-h-2))
		r := image.Rect(x, y, x+w, y+h)
		blocked := false
		for _, other := range occupied {
			if r.Inset(-2).Overlaps(other) {
				blocked = true
				break
			}
		}
		if !blocked {
			out.labels = append(out.labels, axisText{text, r, e.Color})
			break
		}
	}
	return out
}
func emphasisRects(f Frame) []overlayRect {
	scale := 1
	if f.Height >= 360 {
		scale = 2
	}
	return layoutRects(emphasisLayout(f, f.Width, f.Height, textWidth*scale, textHeight*scale), f, scale)
}
