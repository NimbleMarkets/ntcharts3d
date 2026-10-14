package main

import (
	"fmt"
	"image"
	"math"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	picture "nimbleterminal.dev/ntpicture"

	gpuimage "nimbleterminal.dev/ntgpuimage"
)

type tickMsg struct{ generation uint64 }
type quitMsg struct{}
type renderedMsg struct {
	epoch   uint64
	image   *image.NRGBA
	stats   gpuimage.Stats
	elapsed time.Duration
	err     error
}

const chromeRows = 4 // title, stats, breakdown, help

type model struct {
	pic            *picture.Picture
	session        *picture.TerminalSession
	pictures       map[[2]int]*picture.Picture
	format         picture.KittyFormat
	medium         picture.KittyMedium
	closed         bool
	prepareStarted time.Time
	renderer       sceneRenderer

	width, height, cols, rows int
	rasterW, rasterH          int
	density, targetFPS        int

	yaw, pitch, dist float64
	pointSize        float32

	lastClock, frameStarted, lastPresented time.Time
	renderMS, presentMS, fps               float64
	drawMS, submitMS, mapMS, copyMS        float64
	transport                              string
	frames                                 int
	encodedFrames                          map[string]int

	playing, fullscreen, forceGlyph, busy, dirty bool
	epoch, wakeGeneration                        uint64
	duration                                     time.Duration
	err                                          error
}

func newModel(r sceneRenderer, format picture.KittyFormat, medium picture.KittyMedium, fps, density int) *model {
	m := &model{renderer: r, density: density, targetFPS: fps, yaw: .6, pitch: .35, dist: 3, pointSize: 2,
		playing: true, dirty: true, transport: "probing", encodedFrames: make(map[string]int),
		session: picture.NewTerminalSession(picture.TerminalOptions{}), pictures: make(map[[2]int]*picture.Picture), format: format, medium: medium}
	m.selectPicture()
	return m
}

// Picture encoding options are fixed at construction. Cache the small set of
// diagnostic configurations, sharing one terminal session and image allocator.
// Hidden pictures still receive completion/retirement messages.
func (m *model) selectPicture() tea.Cmd {
	var hide tea.Cmd
	if m.pic != nil {
		hide = m.pic.SetVisible(false)
	}
	key := [2]int{int(m.format), int(m.medium)}
	p := m.pictures[key]
	if p == nil {
		p = picture.New(picture.Options{Session: m.session, StableID: true, Render: picture.Config{Fit: picture.FitFill, KittyFormat: m.format, KittyMedium: m.medium}})
		m.pictures[key] = p
	}
	m.pic = p
	mode := picture.PictureGlyph
	if !m.forceGlyph && m.session.Snapshot().Kitty == picture.KittyCapabilitySupported {
		mode = picture.PictureKitty
	}
	m.epoch++
	m.dirty = true
	return tea.Batch(hide, p.SetMode(mode), p.SetVisible(true), m.geometry())
}

func (m *model) Dispose() tea.Cmd {
	if m.closed {
		return nil
	}
	m.closed = true
	m.wakeGeneration++
	var cmds []tea.Cmd
	for _, p := range m.pictures {
		cmds = append(cmds, p.Close())
	}
	m.session.Close()
	return tea.Batch(cmds...)
}

func (m *model) Init() tea.Cmd {
	var quit tea.Cmd
	if m.duration > 0 {
		quit = tea.Tick(m.duration, func(time.Time) tea.Msg { return quitMsg{} })
	}
	return tea.Batch(m.pic.Init(), quit)
}

func (m *model) showChrome() bool { return !m.fullscreen && m.width >= 40 && m.height >= chromeRows+4 }

// geometry derives the cell rectangle and raster size from the terminal
// size, density, and cell pixel size, exactly as the shaders gallery does.
func (m *model) geometry() tea.Cmd {
	m.cols, m.rows = max(1, m.width), max(1, m.height)
	if m.showChrome() {
		m.rows = max(1, m.height-chromeRows)
	}
	resize := m.pic.SetSize(m.cols, m.rows)
	cw, ch := m.pic.CellPixelSize()
	factor := math.Min(1, float64(m.density)/float64(ch))
	factor = math.Min(factor, math.Min(2048/float64(m.cols*cw), 1536/float64(m.rows*ch)))
	factor = math.Min(factor, math.Sqrt((1536*1024)/float64(m.cols*m.rows*cw*ch)))
	factor = math.Min(1, math.Nextafter(factor, math.Inf(1)))
	resolution := m.pic.SetResolution(factor)
	g := m.pic.RasterGeometry()
	m.rasterW, m.rasterH = g.Pixels.X, g.Pixels.Y
	return tea.Batch(resize, resolution)

}

func (m *model) render() tea.Cmd {
	if m.closed || m.busy || m.pic.Status().Busy || m.width < 1 || m.height < 1 || (!m.playing && !m.dirty) || m.err != nil {
		return nil
	}
	geometry := m.geometry()
	if m.rasterW < 1 || m.rasterH < 1 || m.rasterW > 2048 || m.rasterH > 1536 {
		return nil
	}
	now := time.Now()
	if m.playing && !m.lastClock.IsZero() {
		m.yaw += math.Min(now.Sub(m.lastClock).Seconds(), 0.25) * 0.6
	}
	m.lastClock = now
	m.frameStarted = now
	m.wakeGeneration++
	m.busy = true
	m.dirty = false
	epoch := m.epoch
	req := frameRequest{width: m.rasterW, height: m.rasterH, yaw: m.yaw, pitch: m.pitch, dist: m.dist, pointSize: m.pointSize}
	renderer := m.renderer
	return tea.Batch(geometry, func() tea.Msg {
		start := time.Now()
		img, st, err := renderer.Render(req)
		return renderedMsg{epoch, img, st, time.Since(start), err}
	})
}

func smooth(old, next float64) float64 {
	if old == 0 {
		return next
	}
	return old*0.8 + next*0.2
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

func (m *model) Update(msg tea.Msg) (model tea.Model, cmd tea.Cmd) {
	before := m.pic.RasterGeometry()
	var pictureCmd tea.Cmd
	for _, p := range m.pictures {
		pictureCmd = tea.Batch(pictureCmd, p.Update(msg))
	}
	defer func() { cmd = tea.Batch(pictureCmd, cmd) }()
	if m.closed {
		return m, nil
	}
	if !m.forceGlyph && m.pic.Mode() == picture.PictureGlyph && m.session.Snapshot().Kitty == picture.KittyCapabilitySupported {
		pictureCmd = tea.Batch(pictureCmd, m.pic.SetMode(picture.PictureKitty))
	}
	if before != m.pic.RasterGeometry() {
		m.epoch++
		m.dirty = true
		pictureCmd = tea.Batch(pictureCmd, m.geometry())
	}

	switch msg := msg.(type) {
	case quitMsg:
		return m, tea.Sequence(m.Dispose(), tea.Quit)
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.epoch++
		m.dirty = true
		return m, tea.Batch(m.geometry(), m.render())
	case tickMsg:
		if msg.generation != m.wakeGeneration {
			return m, nil
		}
		return m, m.render()
	case renderedMsg:
		m.busy = false
		if msg.err != nil {
			m.err = msg.err
			m.busy = false
			return m, tea.Sequence(m.Dispose(), tea.Quit)
		}
		if msg.epoch != m.epoch {
			m.busy = false
			return m, m.render()
		}
		m.frames++
		m.renderMS = smooth(m.renderMS, ms(msg.elapsed))
		m.drawMS = smooth(m.drawMS, ms(msg.stats.Draw))
		m.submitMS = smooth(m.submitMS, ms(msg.stats.Submit))
		m.mapMS = smooth(m.mapMS, ms(msg.stats.Map))
		m.copyMS = smooth(m.copyMS, ms(msg.stats.Copy))
		m.prepareStarted = time.Now()
		return m, m.pic.SetImageImmutable(msg.image)
	case picture.ErrorMsg:
		if msg.Owner != m.pic || msg.Generation != m.pic.Status().Generation {
			return m, nil
		}
		m.err = msg.Err
		return m, tea.Sequence(m.Dispose(), tea.Quit)
	case picture.PresentedMsg:
		if msg.Owner != m.pic || msg.Generation != m.pic.Status().Generation || m.busy {
			return m, nil
		}
		status := m.pic.Status()
		m.transport = "glyph"
		if m.pic.Mode() == picture.PictureKitty {
			m.transport = status.Format.String()
			if status.Medium == picture.KittyMediumSharedMemory {
				m.transport = "shm"
			}
			m.encodedFrames[m.transport]++
		}
		if !m.prepareStarted.IsZero() {
			m.presentMS = smooth(m.presentMS, ms(time.Since(m.prepareStarted)))
			m.prepareStarted = time.Time{}
		}
		now := time.Now()
		if m.playing && !m.lastPresented.IsZero() {
			m.fps = smooth(m.fps, 1/now.Sub(m.lastPresented).Seconds())
		}
		m.lastPresented = now
		m.busy = false
		if m.frames%30 == 1 {
			debugf("frame %d mode=%d path=%s raster=%dx%d fps=%.1f R=%.2f (draw %.2f submit %.2f map %.2f copy %.2f) P=%.2f kitty=%d",
				m.frames, m.pic.Mode(), m.transport, m.rasterW, m.rasterH, m.fps, m.renderMS, m.drawMS, m.submitMS, m.mapMS, m.copyMS, m.presentMS, m.session.Snapshot().Kitty)
		}
		if m.dirty {
			return m, m.render()
		}
		if !m.playing {
			return m, nil
		}
		delay := max(time.Duration(0), time.Second/time.Duration(m.targetFPS)-time.Since(m.frameStarted))
		generation := m.wakeGeneration
		return m, tea.Tick(delay, func(time.Time) tea.Msg { return tickMsg{generation} })
	case tea.KeyPressMsg:
		var extra tea.Cmd
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Sequence(m.Dispose(), tea.Quit)
		case "left":
			m.yaw -= 0.15
		case "right":
			m.yaw += 0.15
		case "up":
			m.pitch = math.Min(1.5, m.pitch+0.1)
		case "down":
			m.pitch = math.Max(-1.5, m.pitch-0.1)
		case "+", "=":
			m.dist = math.Max(1.2, m.dist*0.9)
		case "-", "_":
			m.dist = math.Min(10, m.dist*1.1)
		case "space":
			m.playing = !m.playing
			m.lastClock = time.Now()
			m.lastPresented = time.Time{}
			m.fps = 0
		case "f", "ctrl+f":
			m.fullscreen = !m.fullscreen
			extra = tea.Batch(extra, m.geometry())
			m.epoch++
		case "esc":
			m.fullscreen = false
			extra = tea.Batch(extra, m.geometry())
			m.epoch++
		case "r":
			m.yaw, m.pitch, m.dist = 0.6, 0.35, 3
			m.epoch++
		case "t":
			format := picture.KittyFormatRGBA
			if m.format == picture.KittyFormatRGBA {
				format = picture.KittyFormatPNG
			}
			m.format = format
			extra = m.selectPicture()
			m.presentMS, m.fps = 0, 0
			m.lastPresented = time.Time{}
		case "m":
			medium := picture.KittyMediumSharedMemory
			if m.medium == picture.KittyMediumSharedMemory {
				medium = picture.KittyMediumDirect
			}
			m.medium = medium
			extra = m.selectPicture()
			m.presentMS, m.fps = 0, 0
			m.lastPresented = time.Time{}
		case "g":
			m.forceGlyph = !m.forceGlyph
			mode := picture.PictureGlyph
			if !m.forceGlyph && m.session.Snapshot().Kitty == picture.KittyCapabilitySupported {
				mode = picture.PictureKitty
			}
			extra = tea.Batch(m.pic.SetMode(mode), m.geometry())
			m.epoch++
		case ",":
			m.density = max(2, m.density-2)
			extra = tea.Batch(extra, m.geometry())
			m.epoch++
		case ".":
			m.density = min(40, m.density+2)
			extra = tea.Batch(extra, m.geometry())
			m.epoch++
		default:
			return m, nil
		}
		m.dirty = true
		return m, tea.Batch(extra, m.render())
	}
	if m.dirty {
		return m, m.render()
	}
	return m, nil
}

var accent = lipgloss.NewStyle().Foreground(lipgloss.Color("#70ead1")).Bold(true)
var muted = lipgloss.NewStyle().Foreground(lipgloss.Color("#8b91aa"))

func (m *model) View() tea.View {
	if m.width < 1 || m.height < 1 {
		return tea.NewView("Starting GPU point cloud…")
	}
	content := m.pic.View()
	if content == "" {
		content = lipgloss.NewStyle().Width(m.cols).Height(m.rows).Render("Rendering…")
	}
	if m.showChrome() {
		state := "PLAY"
		if !m.playing {
			state = "PAUSED"
		}
		title := accent.Render("BOOBA-SHIM / WGPU POINT CLOUD") + "   " + muted.Render(m.renderer.Name()) + "   " + muted.Render(state)
		stats := fmt.Sprintf("%s · %.0f app fps · %d×%d · R %.1f / P %.1f ms",
			m.transport, m.fps, m.rasterW, m.rasterH, m.renderMS, m.presentMS)
		breakdown := fmt.Sprintf("draw %.2f · submit %.2f · map %.2f · copy %.2f ms · yaw %.2f pitch %.2f dist %.2f · density %d",
			m.drawMS, m.submitMS, m.mapMS, m.copyMS, m.yaw, m.pitch, m.dist, m.density)
		help := "←→↑↓ orbit · +- zoom · space pause · f full · t png/rgba · m shm · g glyph · ,. density · r reset · q quit"
		clip := func(s string) string { return ansi.Truncate(s, m.width, "") }
		content = clip(title) + "\n" + clip(muted.Render(stats)) + "\n" + clip(muted.Render(breakdown)) + "\n" + content + "\n" + clip(muted.Render(help))
	}
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}
