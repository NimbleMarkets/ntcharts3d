// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"fmt"
	"image"
	"math"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"
	picture "nimbleterminal.dev/ntpicture"
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

// resolution delegates raster rounding to ntpicture while retaining chart pacing.
func (m *Model) resolution() tea.Cmd {
	factor := 1.0
	if m.renderMode == Software {
		factor = m.softwareScale()
	}
	cw, ch := m.pic.CellPixelSize()
	area := float64(m.plotWidth()) * float64(cw) * float64(m.plotHeight()) * float64(ch)
	if area > maxFramePixels {
		factor = math.Min(factor, math.Sqrt(float64(maxFramePixels)/area))
	}
	cmd := m.pic.SetResolution(factor)
	for g := m.pic.RasterGeometry(); g.Mode == picture.PictureKitty && g.Pixels.X > maxFramePixels/max(1, g.Pixels.Y); g = m.pic.RasterGeometry() {
		factor *= .99
		cmd = tea.Batch(cmd, m.pic.SetResolution(factor))
	}
	return cmd
}

func (m *Model) frame() Frame {
	g := m.pic.RasterGeometry()
	return m.frameSized(max(1, g.Pixels.X), max(1, g.Pixels.Y), float32(g.DisplayAspect), m.renderMode)
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
	if !m.started || m.closed || m.busy || !m.dirty || (m.renderMode != Wireframe && m.pic.Status().Busy) {
		return nil
	}
	m.busy = true
	m.dirty = false
	resolution := m.resolution()
	f := m.frame()
	id, seq, renderMode, r, cols, rows := m.id, m.seq, m.renderMode, m.renderer, m.plotWidth(), m.plotHeight()
	return tea.Batch(resolution, func() tea.Msg {
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
			}
		}
		if renderMode == Software {
			var renderErr error
			img, renderErr = renderSoftware(f)
			if renderErr != nil {
				err = renderErr
			}
		}
		if renderMode == Wireframe {
			wire, err = renderWireframe(f, cols, rows)
		}
		return rendered{id, seq, img, wire, renderMode, err, time.Since(start)}
	})
}

func (m *Model) present(img image.Image) tea.Cmd { return m.pic.SetImageImmutable(img) }

func (m *Model) finish() tea.Cmd {
	if m.busy {
		return nil
	}
	if m.dirty {
		return m.schedule()
	}
	return m.timer()
}

func (m *Model) timer() tea.Cmd {
	if !m.camera.AutoRotate || m.closed {
		return nil
	}
	m.wake++
	id, generation := m.id, m.wake
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg { return tick{id, generation, t} })
}
