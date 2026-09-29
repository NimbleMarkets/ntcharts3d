// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"fmt"
	"image/color"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/NimbleMarkets/ntcharts3d/math3d"
	zone "github.com/lrstanley/bubblezone/v2"
)

var nextID atomic.Uint64

// Model is a Bubble Tea chart containing named series in a shared 3D space.
type Model struct {
	colorDomain                       *Range
	plotAspect                        math3d.Vec3
	prepared                          []Geometry
	preparedRevision                  uint64
	textureRevision                   uint64
	grid                              Grid
	emphasis                          Emphasis
	transport                         string
	forceGlyph                        bool
	pic                               picture.Model
	camera                            Camera
	axes                              Axes
	light                             Light
	series                            []series
	colorLegendTitle                  string
	colorLegendVisible                bool
	colorLegendMode                   ColorLegendMode
	sharedColorDomain                 bool
	colors                            Palette
	background                        color.RGBA
	renderer                          Renderer
	renderMode                        RenderMode
	requestedRenderMode               RenderMode
	width, height, kittyID, maxPoints int
	medium                            picture.KittyMedium
	id, seq, revision, wake           uint64
	started, busy, dirty, closed      bool
	zones                             *zone.Manager
	ownZones                          bool
	zoneID                            string
	drag, moved                       bool
	lastX, lastY                      int
	lastInput                         time.Time
	hover                             *PickMsg
	err                               error
	wire                              string
	slow                              int
}

func New(width, height int, opts ...Option) *Model {
	m := &Model{camera: DefaultCamera(), light: Light{math3d.Vec3{X: 1, Y: -1, Z: 2}.Normalize(), .3}, colors: defaultPalette(), background: color.RGBA{235, 238, 242, 255}, maxPoints: MaxPoints, id: nextID.Add(1), revision: 1, dirty: true, renderer: &gpuRenderer{}, colorLegendVisible: true}
	for _, o := range opts {
		o(m)
	}
	if m.kittyID <= 0 {
		m.kittyID = picture.DefaultKittyID + 1000 + int(m.id)
	}
	if m.zones == nil {
		m.zones = zone.New()
		m.ownZones = true
	}
	m.zoneID = m.zones.NewPrefix() + "ntcharts3d"
	m.pic = picture.NewWithConfig(picture.Config{KittyZ: -1, Fit: picture.FitFill, Background: m.background, KittyID: m.kittyID, KittyMedium: m.medium, CellPixelWidth: 8, CellPixelHeight: 16})
	m.SetSize(width, height)
	return m
}

func (m *Model) ID() uint64 { return m.id }
func (m *Model) Err() error { return m.err }

// RenderMode returns the active rendering mode, including automatic fallback.
func (m *Model) RenderMode() RenderMode { return m.renderMode }

// RequestedRenderMode returns the last selected mode, defaulting to WebGPU.
// Fallback does not change it.
func (m *Model) RequestedRenderMode() RenderMode  { return m.requestedRenderMode }
func (m *Model) Camera() Camera                   { return m.camera }
func (m *Model) PictureMode() picture.PictureMode { return m.pic.Mode() }
func (m *Model) SetCamera(c Camera) tea.Cmd {
	m.camera = c.valid()
	m.hover = nil
	m.wake++
	return m.changed(false)
}
func (m *Model) SetLight(l Light) tea.Cmd {
	if !validFloat(l.Direction.X, l.Direction.Y, l.Direction.Z, l.Ambient) {
		m.err = fmt.Errorf("ntcharts3d: non-finite light")
		return nil
	}
	if l.Direction.Dot(l.Direction) == 0 {
		l.Direction = math3d.Vec3{Z: 1}
	}
	l.Direction = l.Direction.Normalize()
	l.Ambient = max(0, min(1, l.Ambient))
	m.light = l
	return m.changed(false)
}

func (m *Model) SetSize(w, h int) tea.Cmd {
	w, h = max(1, w), max(3, h)
	if m.width == w && m.height == h {
		return nil
	}
	m.width, m.height = w, h
	_ = m.pic.SetSize(m.plotWidth(), m.plotHeight())
	return m.changed(false)
}

// SetRenderMode selects both the requested and active mode. Selecting WebGPU
// after fallback retries GPU initialization. Invalid modes are ignored.
func (m *Model) SetRenderMode(l RenderMode) tea.Cmd {
	if l > Wireframe {
		return nil
	}
	m.renderMode = l
	m.requestedRenderMode = l
	_ = m.resizePicture()
	m.slow = 0
	m.wire = ""
	m.wake++
	if l == WebGPU && m.renderer == nil {
		m.renderer = &gpuRenderer{}
	}
	return m.changed(false)
}

func (m *Model) Init() tea.Cmd { m.started = true; return tea.Batch(m.pic.Init(), m.schedule()) }

// Close releases GPU resources and pending shared-memory images. Call it after
// the Bubble Tea program exits. The GPU renderer waits for any active render.
func (m *Model) Close() error {
	m.closed = true
	m.wake++
	m.pic.SetImage(nil)
	if m.ownZones {
		m.zones.Close()
	}
	if m.renderer != nil {
		return m.renderer.Close()
	}
	return nil
}
