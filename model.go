// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"fmt"
	"image/color"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"nimbleterminal.dev/nt3d"
	nt3dmath "nimbleterminal.dev/nt3d/math"
	picture "nimbleterminal.dev/ntpicture"
)

var nextID atomic.Uint64

// Model is a Bubble Tea chart containing named series in a shared 3D space.
type Model struct {
	colorDomain                  *Range
	plotAspect                   nt3dmath.Vec3
	prepared                     []Geometry
	preparedRevision             uint64
	textureRevision              uint64
	grid                         Grid
	emphasis                     Emphasis
	transport                    string
	forceGlyph                   bool
	pic                          *picture.Picture
	session                      *picture.TerminalSession
	ownSession                   bool
	nextImageID                  func() (int, error)
	camera                       Camera
	axes                         Axes
	light                        Light
	series                       []series
	colorLegendTitle             string
	colorLegendVisible           bool
	colorLegendMode              ColorLegendMode
	sharedColorDomain            bool
	colors                       Palette
	background                   color.RGBA
	renderer                     Renderer
	renderMode                   RenderMode
	requestedRenderMode          RenderMode
	width, height, maxPoints     int
	anisotropy                   int
	medium                       picture.KittyMedium
	kittyFormat                  picture.KittyFormat
	id, seq, revision, wake      uint64
	started, busy, dirty, closed bool
	zones                        *zone.Manager
	ownZones                     bool
	zoneID                       string
	drag, moved                  bool
	lastX, lastY                 int
	lastInput                    time.Time
	hover                        *PickMsg
	err                          error
	wire                         string
	slow                         int
	shrink                       float64 // How far software frames are reduced: 0 is not at all.
}

func New(width, height int, opts ...Option) *Model {
	m := &Model{
		camera:             DefaultCamera(),
		light:              Light{nt3dmath.Vec3{X: 1, Y: -1, Z: 2}.Normalize(), .3},
		colors:             defaultPalette(),
		background:         color.RGBA{235, 238, 242, 255},
		maxPoints:          nt3d.MaxPoints,
		anisotropy:         defaultAnisotropy,
		id:                 nextID.Add(1),
		revision:           1,
		dirty:              true,
		kittyFormat:        picture.KittyFormatZlib,
		renderer:           &gpuRenderer{},
		colorLegendVisible: true,
	}
	for _, o := range opts {
		o(m)
	}
	if m.session == nil {
		m.session = picture.NewTerminalSession(picture.TerminalOptions{})
		m.ownSession = true
	}
	if m.zones == nil {
		m.zones = zone.New()
		m.ownZones = true
	}
	m.zoneID = m.zones.NewPrefix() + "ntcharts3d"
	m.pic = picture.New(picture.Options{Session: m.session, AutoMode: true, NextID: m.nextImageID, StableID: true, Render: picture.Config{
		MaxRasterPixels: maxFramePixels, KittyZ: -1, Fit: picture.FitFill, Background: m.background, KittyMedium: m.medium, KittyFormat: m.kittyFormat,
	}})
	m.pic.SetVisible(m.renderMode != Wireframe)
	m.SetSize(width, height)
	return m
}

func (m *Model) ID() uint64 { return m.id }
func (m *Model) Err() error { return m.err }

// Anisotropy is the texture sample limit used by both raster renderers.
func (m *Model) Anisotropy() int { return m.anisotropy }

// SetAnisotropy changes texture filtering without rebuilding chart geometry.
// Zero restores the default of 4; other values are clamped to 1..nt3d.MaxAnisotropy.
// Use 1 for mipmapping alone or higher values for detail at oblique angles.
func (m *Model) SetAnisotropy(n int) tea.Cmd {
	n = textureAnisotropy(n)
	if m.anisotropy == n {
		return nil
	}
	m.anisotropy = n
	return m.changed(false)
}

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

// Background returns the color behind the scene.
func (m *Model) Background() color.RGBA { return m.background }

// SetBackground changes the color behind the scene and redraws it. It is
// WithBackground for a chart already made; the color is always opaque.
func (m *Model) SetBackground(c color.Color) tea.Cmd {
	m.background = color.RGBAModel.Convert(c).(color.RGBA)
	m.background.A = 255
	return m.changed(false)
}

func (m *Model) SetLight(l Light) tea.Cmd {
	if !validFloat(l.Direction.X, l.Direction.Y, l.Direction.Z, l.Ambient) {
		m.err = fmt.Errorf("ntcharts3d: non-finite light")
		return nil
	}
	if l.Direction.Dot(l.Direction) == 0 {
		l.Direction = nt3dmath.Vec3{Z: 1}
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
	return tea.Batch(m.resizePicture(), m.changed(false))
}

// SetRenderMode selects both the requested and active mode. Selecting WebGPU
// after fallback retries GPU initialization. Invalid modes are ignored.
func (m *Model) SetRenderMode(l RenderMode) tea.Cmd {
	if l > Wireframe {
		return nil
	}
	m.renderMode = l
	m.requestedRenderMode = l
	cleanup := m.resizePicture()
	m.slow, m.shrink = 0, 0
	m.wire = ""
	m.wake++
	if l == WebGPU && m.renderer == nil {
		m.renderer = &gpuRenderer{}
	}
	return tea.Batch(cleanup, m.pic.SetVisible(l != Wireframe), m.resolution(), m.changed(false))
}

func (m *Model) Init() tea.Cmd {
	m.started = true
	// Select already-known capability before producing the first frame. Later
	// discovery is handled through this session's TerminalSessionMsg.
	return tea.Batch(m.pic.Init(), m.togglePicture(), m.schedule())
}

// PictureStatus reports encoding, transport, and presentation progress.
func (m *Model) PictureStatus() picture.Status { return m.pic.Status() }

// Dispose releases terminal placements. Execute the returned command before
// quitting, and continue routing late messages through Update.
func (m *Model) Dispose() tea.Cmd {
	if m.closed {
		return nil
	}
	m.closed = true
	m.wake++
	cmd := m.pic.Close()
	if m.ownSession {
		m.session.Close()
	}
	return cmd
}

// Close releases GPU resources after the event loop exits. Call Dispose while
// the loop is running to execute terminal placement cleanup.
func (m *Model) Close() error {
	m.Dispose()
	if m.ownZones {
		m.zones.Close()
	}
	if m.renderer != nil {
		return m.renderer.Close()
	}
	return nil
}
