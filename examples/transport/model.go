package main

import (
	"fmt"
	"image"
	"math"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	gpuimage "github.com/NimbleMarkets/go-gpuimage"
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
type encodedMsg struct {
	frame   tea.Msg
	elapsed time.Duration
}
type presentedMsg struct{}

const chromeRows = 4 // title, stats, breakdown, help

type model struct {
	pic      picture.Model
	renderer sceneRenderer

	width, height, cols, rows int
	rasterW, rasterH          int
	density, targetFPS        int

	yaw, pitch, dist float64
	pointSize        float32

	lastClock, frameStarted, lastPresented time.Time
	renderMS, encodeMS, fps                float64
	drawMS, submitMS, mapMS, copyMS        float64
	bytes                                  int
	transport                              string
	frames                                 int
	encodedFrames                          map[string]int

	playing, fullscreen, forceGlyph, busy, dirty bool
	epoch, wakeGeneration                        uint64
	duration                                     time.Duration
	err                                          error
}

func newModel(r sceneRenderer, format picture.KittyFormat, medium picture.KittyMedium, fps, density int) *model {
	return &model{
		renderer: r, density: density, targetFPS: fps,
		yaw: 0.6, pitch: 0.35, dist: 3, pointSize: 2,
		playing: true, dirty: true, transport: "probing",
		encodedFrames: make(map[string]int),
		pic: picture.NewWithConfig(picture.Config{
			Fit: picture.FitFill, KittyFormat: format, KittyMedium: medium, CellPixelWidth: 8, CellPixelHeight: 16,
		}),
	}
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
func (m *model) geometry() {
	m.cols, m.rows = max(1, m.width), max(1, m.height)
	if m.showChrome() {
		m.rows = max(1, m.height-chromeRows)
	}
	m.pic.SetSize(m.cols, m.rows)
	cw, ch := m.pic.CellPixelSize()
	factor := math.Min(1, float64(m.density)/float64(ch))
	factor = math.Min(factor, math.Min(2048/float64(m.cols*cw), 1536/float64(m.rows*ch)))
	factor = math.Min(factor, math.Sqrt((1536*1024)/float64(m.cols*m.rows*cw*ch)))
	factor = math.Min(1, math.Nextafter(factor, math.Inf(1)))
	m.pic.SetKittyResolutionFactor(factor)
	m.rasterW = m.cols * max(1, int(float64(cw)*factor))
	m.rasterH = m.rows * max(1, int(float64(ch)*factor))
	if m.pic.Mode() == picture.PictureGlyph {
		m.rasterW = min(m.cols, 2048)
		m.rasterH = min(m.rows*2, 1536)
	}
}

func (m *model) render() tea.Cmd {
	if m.busy || m.width < 1 || m.height < 1 || (!m.playing && !m.dirty) || m.err != nil {
		return nil
	}
	m.geometry()
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
	return func() tea.Msg {
		start := time.Now()
		img, st, err := renderer.Render(req)
		return renderedMsg{epoch, img, st, time.Since(start), err}
	}
}

func smooth(old, next float64) float64 {
	if old == 0 {
		return next
	}
	return old*0.8 + next*0.2
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case quitMsg:
		return m, tea.Quit
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.epoch++
		m.dirty = true
		m.geometry()
		return m, m.render()
	case uv.CellSizeEvent:
		m.pic.SetCellPixelSize(msg.Width, msg.Height)
		m.epoch++
		m.dirty = true
		m.geometry()
		return m, m.render()
	case tickMsg:
		if msg.generation != m.wakeGeneration {
			return m, nil
		}
		return m, m.render()
	case renderedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.busy = false
			return m, tea.Quit
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
		cmd := m.pic.SetImage(msg.image)
		if cmd == nil {
			m.bytes, m.encodeMS, m.transport = 0, 0, "glyph"
			return m, func() tea.Msg { return presentedMsg{} }
		}
		return m, func() tea.Msg { start := time.Now(); frame := cmd(); return encodedMsg{frame, time.Since(start)} }
	case encodedMsg:
		m.encodeMS = smooth(m.encodeMS, ms(msg.elapsed))
		if frame, ok := msg.frame.(picture.KittyFrameMsg); ok {
			// With shared memory the APC contains a name, not the pixels.
			m.transport = frame.Format.String()
			if frame.Medium == picture.KittyMediumSharedMemory {
				m.transport = "shm"
			}
			m.encodedFrames[m.transport]++
			m.bytes = len(frame.APC)
		}
		return m, tea.Sequence(m.pic.Update(msg.frame), func() tea.Msg { return presentedMsg{} })
	case presentedMsg:
		now := time.Now()
		if m.playing && !m.lastPresented.IsZero() {
			m.fps = smooth(m.fps, 1/now.Sub(m.lastPresented).Seconds())
		}
		m.lastPresented = now
		m.busy = false
		if m.frames%30 == 1 {
			debugf("frame %d mode=%d path=%s raster=%dx%d fps=%.1f R=%.2f (draw %.2f submit %.2f map %.2f copy %.2f) E=%.2f bytes=%d kitty=%d",
				m.frames, m.pic.Mode(), m.transport, m.rasterW, m.rasterH, m.fps, m.renderMS, m.drawMS, m.submitMS, m.mapMS, m.copyMS, m.encodeMS, m.bytes, picture.KittySupported())
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
			return m, tea.Quit
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
			m.geometry()
			m.epoch++
		case "esc":
			m.fullscreen = false
			m.geometry()
			m.epoch++
		case "r":
			m.yaw, m.pitch, m.dist = 0.6, 0.35, 3
			m.epoch++
		case "t":
			format := picture.KittyFormatRGBA
			if m.pic.KittyFormat() == picture.KittyFormatRGBA {
				format = picture.KittyFormatPNG
			}
			extra = m.pic.SetKittyFormat(format)
			m.encodeMS, m.fps = 0, 0
			m.lastPresented = time.Time{}
		case "m":
			medium := picture.KittyMediumSharedMemory
			if m.pic.KittyMedium() == picture.KittyMediumSharedMemory {
				medium = picture.KittyMediumDirect
			}
			extra = m.pic.SetKittyMedium(medium)
			m.encodeMS, m.fps = 0, 0
			m.lastPresented = time.Time{}
		case "g":
			m.forceGlyph = !m.forceGlyph
			if m.forceGlyph && m.pic.Mode() == picture.PictureKitty || !m.forceGlyph && m.pic.Mode() == picture.PictureGlyph {
				extra = m.pic.Toggle()
			}
			m.geometry()
			m.epoch++
		case ",":
			m.density = max(2, m.density-2)
			m.geometry()
			m.epoch++
		case ".":
			m.density = min(40, m.density+2)
			m.geometry()
			m.epoch++
		default:
			return m, nil
		}
		m.dirty = true
		return m, tea.Batch(extra, m.render())
	}
	cmd := m.pic.Update(msg)
	if !m.forceGlyph && m.pic.Mode() == picture.PictureGlyph && picture.KittySupported() == picture.KittyCapabilitySupported {
		debugf("kitty probe answered: switching picture to Kitty mode")
		toggle := m.pic.Toggle()
		m.epoch++
		m.dirty = true
		return m, tea.Batch(cmd, toggle, m.render())
	}
	return m, cmd
}

var accent = lipgloss.NewStyle().Foreground(lipgloss.Color("#70ead1")).Bold(true)
var muted = lipgloss.NewStyle().Foreground(lipgloss.Color("#8b91aa"))

func (m *model) View() tea.View {
	if m.width < 1 || m.height < 1 {
		return tea.NewView("Starting GPU point cloud…")
	}
	content := m.pic.String()
	if content == "" {
		content = lipgloss.NewStyle().Width(m.cols).Height(m.rows).Render("Rendering…")
	}
	if m.showChrome() {
		state := "PLAY"
		if !m.playing {
			state = "PAUSED"
		}
		title := accent.Render("BOOBA-SHIM / WGPU POINT CLOUD") + "   " + muted.Render(m.renderer.Name()) + "   " + muted.Render(state)
		stats := fmt.Sprintf("%s · %.0f app fps · %d×%d · R %.1f / E %.1f ms · %s",
			m.transport, m.fps, m.rasterW, m.rasterH, m.renderMS, m.encodeMS, formatBytes(m.bytes))
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

func formatBytes(n int) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KiB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.2f MiB", float64(n)/(1024*1024))
	}
}
