// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"time"

	tea "charm.land/bubbletea/v2"
	picture "nimbleterminal.dev/ntpicture"
)

func (m *Model) Update(msg tea.Msg) (model tea.Model, cmd tea.Cmd) {
	before := m.pic.RasterGeometry()
	pictureCmd := m.pic.Update(msg)
	defer func() { cmd = tea.Batch(pictureCmd, cmd) }()

	if m.closed {
		return m, nil
	}
	if before != m.pic.RasterGeometry() {
		m.seq++
		m.dirty = true
		m.wake++
	}
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		return m, m.SetSize(v.Width, v.Height)
	case rendered:
		if v.id != m.id {
			return m, nil
		}
		m.busy = false
		if v.err != nil {
			m.err = v.err
			if m.renderer != nil {
				_ = m.renderer.Close()
				m.renderer = nil
			}
			m.renderMode = v.renderMode
		}
		if v.seq != m.seq {
			return m, m.schedule()
		}
		m.renderMode = v.renderMode
		m.wire = v.wire
		if v.renderMode == Software && m.paced(v.elapsed) {
			m.slow++
		} else {
			m.slow = 0
		}
		if m.slow >= 3 {
			m.renderMode = Wireframe
			pictureCmd = tea.Batch(pictureCmd, m.pic.SetVisible(false), m.resizePicture())
			m.dirty = true
		}
		if v.renderMode == Wireframe {
			return m, m.finish()
		}
		if m.renderMode == Wireframe {
			return m, m.schedule()
		}
		if v.frame == nil {
			return m, m.finish()
		}
		return m, m.present(v.frame)
	case picture.PresentedMsg:
		if v.Owner != m.pic || v.Generation != m.pic.Status().Generation {
			return m, nil
		}
		status := m.pic.Status()
		m.transport = status.Format.String()
		if status.Medium == picture.KittyMediumSharedMemory {
			m.transport = "shm"
		}
		return m, m.finish()
	case picture.ErrorMsg:
		if v.Owner != m.pic || v.Generation != m.pic.Status().Generation {
			return m, nil
		}
		m.err = v.Err
		if m.pic.Mode() == picture.PictureKitty {
			m.forceGlyph = true
			return m, tea.Batch(m.togglePicture(), m.changed(false))
		}
		return m, m.SetRenderMode(Wireframe)
	case tick:
		if v.id != m.id || v.generation != m.wake || !m.camera.AutoRotate {
			return m, nil
		}
		if time.Since(m.lastInput).Seconds() >= m.camera.ResumeAfter {
			m.camera.Beta += 1
			return m, m.changed(false)
		}
		return m, m.timer()
	case tea.KeyPressMsg:
		switch v.String() {
		case "q", "ctrl+c":
			return m, tea.Sequence(m.Dispose(), tea.Quit)
		case "left", "h":
			m.camera.Beta -= 5
		case "right", "l":
			m.camera.Beta += 5
		case "up", "k":
			m.camera.Alpha += 5
		case "down", "j":
			m.camera.Alpha -= 5
		case "+", "=":
			m.camera.Distance *= .9
		case "-", "_":
			m.camera.Distance /= .9
		case "r":
			m.camera.AutoRotate = !m.camera.AutoRotate
			m.wake++
		case "o":
			if m.camera.Projection == Orthographic {
				m.camera.Projection = Perspective
			} else {
				m.camera.Projection = Orthographic
			}
		case "g":
			m.forceGlyph = !m.forceGlyph
			cleanup := m.togglePicture()
			render := m.changed(false)
			return m, tea.Batch(cleanup, render)
		case "v":
			mode := ColorLegendText
			if m.colorLegendMode == ColorLegendText {
				mode = ColorLegendImage
			}
			return m, m.SetColorLegendMode(mode)
		default:
			return m, nil
		}
		m.camera = m.camera.valid()
		m.lastInput = time.Now()
		return m, m.changed(false)
	case tea.MouseMsg:
		return m, m.mouse(v)
	}
	return m, m.schedule()
}
