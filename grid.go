// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	tea "charm.land/bubbletea/v2"
	"image/color"
)

// Grid adds tick-aligned lines on the three faces behind the data. It is off by default.
type Grid struct {
	Show  bool
	Color color.Color // Nil uses a muted gray.
}

// WithGrid sets the initial back-plane grid settings.
func WithGrid(g Grid) Option { return func(m *Model) { m.SetGrid(g) } }

// SetGrid replaces the back-plane grid settings.
func (m *Model) SetGrid(g Grid) tea.Cmd {
	if g.Color != nil {
		g.Color = color.RGBAModel.Convert(g.Color)
	}
	m.grid = g
	return m.changed(false)
}

// Grid returns the current back-plane grid settings.
func (m *Model) Grid() Grid { return m.grid }

func (m *Model) gridLines(f Frame) []Vertex {
	if !m.grid.Show || !f.Bounds.IsValid() {
		return nil
	}
	ink := color.RGBA{R: 185, G: 190, B: 198, A: 255}
	if m.grid.Color != nil {
		ink = color.RGBAModel.Convert(m.grid.Color).(color.RGBA)
	}
	eye, _, _, forward := m.camera.basis()
	t := m.plotTransform()
	lo, hi, e, center := components(f.Bounds.Min), components(f.Bounds.Max), components(eye), components(t.center)
	// Convert the camera position back to data coordinates to select the far faces.
	scale := components(t.scale)
	for i := range 3 {
		e[i] = center[i] + e[i]/scale[i]
	}
	var lines []Vertex
	for plane := range 3 {
		face := lo[plane]
		farIsMax := e[plane] < center[plane]
		if m.camera.Projection == Orthographic {
			farIsMax = components(forward)[plane] > 0
		}
		if farIsMax {
			face = hi[plane]
		}
		for axis := range 3 {
			if axis == plane || f.Axes[axis].Hidden {
				continue
			}
			other := 3 - plane - axis
			ticks := f.Axes[axis].Ticks
			step := max(1, (len(ticks)+63)/64)
			for j := 0; j < len(ticks); j += step {
				tick := ticks[j]
				if tick.Value < lo[axis] || tick.Value > hi[axis] {
					continue
				}
				a, b := lo, hi
				a[plane], b[plane], a[axis], b[axis] = face, face, tick.Value, tick.Value
				if a[other] == b[other] {
					continue
				}
				lines = append(lines, Vertex{Position: vector(a), Color: ink}, Vertex{Position: vector(b), Color: ink})
			}
		}
	}
	return lines
}
