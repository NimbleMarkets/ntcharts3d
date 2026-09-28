// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"fmt"
	"image"
	"math"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
)

func (m *Model) changed(data bool) tea.Cmd {
	m.seq++
	m.dirty = true
	if data {
		m.revision++
		m.hover = nil
	}
	return m.schedule()
}

type rendered struct {
	id, seq    uint64
	frame      image.Image
	wire       string
	renderMode RenderMode
	err        error
	elapsed    time.Duration
}

type encoded struct {
	id  uint64
	msg tea.Msg
}

type presented struct{ id uint64 }

type tick struct {
	id, generation uint64
	at             time.Time
}

func (m *Model) frame() Frame {
	cw, ch := m.pic.CellPixelSize()
	plotRows := m.plotHeight()
	plotCols := m.plotWidth()
	w, h := plotCols*cw, plotRows*ch
	if m.renderMode == Software || m.pic.Mode() == picture.PictureGlyph {
		factor := math.Min(1, math.Min(320/float64(w), 200/float64(h)))
		w, h = max(1, int(float64(w)*factor)), max(1, int(float64(h)*factor))
	}
	if w*h > 4096*2160 {
		factor := math.Sqrt(float64(4096*2160) / float64(w*h))
		w, h = max(1, int(float64(w)*factor)), max(1, int(float64(h)*factor))
	}
	b := m.PlotBounds()
	n := m.plotTransform().matrix
	geoms := m.preparedGeometry()
	var legends []ColorLegend
	if m.colorLegendVisible && m.colorLegendMode == ColorLegendImage && m.renderMode != Wireframe {
		if m.sharedColorDomain || m.colorDomain != nil {
			lo, hi, found := float32(0), float32(0), false
			for _, s := range m.series {
				if s.geometry.HasColorRange {
					if !found {
						lo, hi, found = s.geometry.ColorMin, s.geometry.ColorMax, true
					} else {
						lo, hi = min(lo, s.geometry.ColorMin), max(hi, s.geometry.ColorMax)
					}
				}
			}
			if found {
				label := m.colorLegendTitle
				if label == "" {
					label = "value"
				}
				legends = append(legends, ColorLegend{label, lo, hi, slices.Clone(m.colors)})
			}
		} else {
			for _, s := range m.series {
				if s.geometry.HasColorRange {
					label := s.name
					if m.colorLegendTitle != "" {
						label = m.colorLegendTitle
					}
					legends = append(legends, ColorLegend{label, s.geometry.ColorMin, s.geometry.ColorMax, slices.Clone(m.colors)})
				}
			}
		}
	}
	f := Frame{Width: w, Height: h, Matrix: m.camera.Matrix(float32(plotCols*cw) / float32(plotRows*ch)).Mul(n), Revision: m.revision, Bounds: b, Axes: m.axisFrames(b), Geometry: geoms, ColorLegends: legends, Background: m.background, Light: m.light}
	f.GridLines = m.gridLines(f)
	f.Emphasis = m.emphasisFrame(f)
	return f
}

func (m *Model) schedule() tea.Cmd {
	if !m.started || m.closed || m.busy || !m.dirty {
		return nil
	}
	m.busy = true
	m.dirty = false
	f := m.frame()
	id, seq, renderMode, r, cols, rows := m.id, m.seq, m.renderMode, m.renderer, m.plotWidth(), m.plotHeight()
	return func() tea.Msg {
		start := time.Now()
		var img image.Image
		var err error
		wire := ""
		if renderMode == WebGPU {
			if r == nil {
				err = fmt.Errorf("ntcharts3d: GPU unavailable")
			} else {
				img, err = r.Render(f)
			}
			if err != nil {
				renderMode = Software
				factor := math.Min(1, math.Min(320/float64(f.Width), 200/float64(f.Height)))
				f.Width, f.Height = max(1, int(float64(f.Width)*factor)), max(1, int(float64(f.Height)*factor))
			}
		}
		if renderMode == Software {
			img = softwareRender(f)
		}
		if renderMode == Wireframe {
			wire = wireRender(f, cols, rows)
		}
		return rendered{id, seq, img, wire, renderMode, err, time.Since(start)}
	}
}

// present sends a complete frame to picture. The dirty rectangle is unused.
func (m *Model) present(img image.Image, dirty image.Rectangle) tea.Cmd {
	cmd := m.pic.SetImage(img)
	id := m.id
	if cmd == nil {
		return func() tea.Msg { return presented{id} }
	}
	return func() tea.Msg { return encoded{id, cmd()} }
}

func (m *Model) timer() tea.Cmd {
	if !m.camera.AutoRotate || m.closed {
		return nil
	}
	m.wake++
	id, generation := m.id, m.wake
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg { return tick{id, generation, t} })
}
