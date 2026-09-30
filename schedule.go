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

// maxFramePixels bounds the framebuffer's area.
const maxFramePixels = 4096 * 2160

// Software frames are drawn at full size while they are prompt, and smaller
// while they are not: a frame over softwareSlow is followed by one reduced to
// take softwareAim, and one under softwareFast by one a half larger.
const (
	softwareSlow = 60 * time.Millisecond
	softwareAim  = 40 * time.Millisecond
	softwareFast = 20 * time.Millisecond
)

// smallest is the scale at which the plot fits 320 by 200 pixels. Glyphs
// show two pixels to a cell, and are never given more. Software frames are
// never reduced further.
func (m *Model) smallest() float64 {
	cw, ch := m.pic.CellPixelSize()
	return math.Min(1, math.Min(320/float64(m.plotWidth()*cw), 200/float64(m.plotHeight()*ch)))
}

// softwareScale is the scale of the next software frame.
func (m *Model) softwareScale() float64 {
	if m.pic.Mode() == picture.PictureGlyph {
		return m.smallest()
	}
	return math.Max(m.smallest(), 1-m.shrink)
}

// paced sets the size of software frames to come by the time the last took.
// It reports whether that frame was slow at the smallest size, where there
// is nothing left to reduce.
func (m *Model) paced(elapsed time.Duration) bool {
	scale, smallest := m.softwareScale(), m.smallest()
	switch {
	case scale <= smallest:
		if elapsed < softwareFast && m.pic.Mode() != picture.PictureGlyph && scale < 1 {
			break
		}
		return elapsed > 150*time.Millisecond
	case elapsed > softwareSlow:
		// Time goes with area, and area with the square of the scale.
		m.shrink = 1 - math.Max(smallest, scale*math.Sqrt(float64(softwareAim)/float64(elapsed)))
		return false
	case elapsed >= softwareFast || scale >= 1:
		return false
	}
	// Quick enough to be drawn again, larger, even if nothing else changes.
	m.shrink = 1 - math.Min(1, scale*1.5)
	m.dirty = true
	return false
}

// frame is what the terminal shows: sized by its cells, and reduced where
// the picture is shown as glyphs or drawn slowly in software.
func (m *Model) frame() Frame {
	cw, ch := m.pic.CellPixelSize()
	plotRows := m.plotHeight()
	plotCols := m.plotWidth()
	w, h := plotCols*cw, plotRows*ch
	factor := 1.0
	switch {
	case m.pic.Mode() == picture.PictureGlyph:
		factor = m.smallest()
	case m.renderMode == Software:
		factor = m.softwareScale()
	}
	w, h = max(1, int(float64(w)*factor)), max(1, int(float64(h)*factor))
	if w*h > maxFramePixels {
		factor := math.Sqrt(float64(maxFramePixels) / float64(w*h))
		w, h = max(1, int(float64(w)*factor)), max(1, int(float64(h)*factor))
	}
	return m.frameSized(w, h, float32(plotCols*cw)/float32(plotRows*ch), m.renderMode)
}

// frameSized is the chart as it stands, w by h pixels, seen through a camera
// of the given aspect and drawn in the given mode.
func (m *Model) frameSized(w, h int, aspect float32, mode RenderMode) Frame {
	b := m.PlotBounds()
	n := m.plotTransform().matrix
	geoms := m.preparedGeometry()
	var legends []ColorLegend
	if m.colorLegendVisible && m.colorLegendMode == ColorLegendImage && mode != Wireframe {
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
	f := Frame{Width: w, Height: h, Matrix: m.camera.Matrix(aspect).Mul(n), Revision: m.revision, TextureRevision: m.textureRevision, Bounds: b, Axes: m.axisFrames(b), Geometry: geoms, ColorLegends: legends, Background: m.background, Light: m.light}
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
