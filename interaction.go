// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"math"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
)

func (m *Model) mouse(msg tea.MouseMsg) tea.Cmd {
	z := m.zones.Get(m.zoneID)
	event := msg.Mouse()
	inside := z.InBounds(msg)
	x, y := z.Pos(msg)
	inside = inside && x < m.plotWidth()
	switch msg.(type) {
	case tea.MouseReleaseMsg:
		wasClick := m.drag && !m.moved
		m.drag = false
		if wasClick && inside {
			cmd := m.setHover(m.Pick(x, y))
			if m.hover != nil {
				pick := *m.hover
				return tea.Batch(cmd, func() tea.Msg { return pick })
			}
			return cmd
		}
		return nil
	case tea.MouseClickMsg:
		if inside && event.Button == tea.MouseLeft {
			m.drag = true
			m.moved = false
			m.lastX, m.lastY = event.X, event.Y
		}
		return nil
	case tea.MouseWheelMsg:
		if !inside {
			return nil
		}
		if event.Button == tea.MouseWheelUp {
			m.camera.Distance *= .9
		} else {
			m.camera.Distance /= .9
		}
	case tea.MouseMotionMsg:
		if m.drag {
			dx, dy := event.X-m.lastX, event.Y-m.lastY
			m.lastX, m.lastY = event.X, event.Y
			if dx == 0 && dy == 0 {
				return nil
			}
			m.moved = true
			if event.Mod&tea.ModShift != 0 {
				_, r, u, _ := m.camera.basis()
				cw, ch := m.pic.CellPixelSize()
				h := float32(m.camera.Distance)
				if m.camera.Projection == Perspective {
					h *= float32(2 * math.Tan(math.Pi/8))
				}
				m.camera.Target = m.camera.Target.Add(r.Scale(-float32(dx*cw) * h / float32((m.plotHeight())*ch)).Add(u.Scale(float32(dy) * h / float32(m.plotHeight()))))
			} else {
				m.camera.Beta += float64(dx) * 2
				m.camera.Alpha += float64(dy) * 2
			}
		} else {
			if inside {
				return m.setHover(m.Pick(x, y))
			}
			return m.setHover(nil)
		}
	default:
		return nil
	}
	m.hover = nil
	m.camera = m.camera.valid()
	m.lastInput = time.Now()
	return m.changed(false)
}

// togglePicture skips encoding the old glyph image when entering Kitty mode;
// a new frame will replace it. When leaving, return the image deletion command.
func (m *Model) togglePicture() tea.Cmd {
	wasKitty := m.pic.Mode() == picture.PictureKitty
	cmd := m.pic.Toggle()
	if wasKitty {
		return cmd
	}
	return nil
}
