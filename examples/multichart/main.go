// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts3d"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"
	picture "nimbleterminal.dev/ntpicture"
)

type dashboard struct {
	session      *picture.TerminalSession
	zones        *zone.Manager
	charts       [2]*ntcharts3d.Model
	focus, width int
	duration     time.Duration
}

type quitMsg struct{}

func newDashboard(mode ntcharts3d.RenderMode) *dashboard {
	d := &dashboard{
		session: picture.NewTerminalSession(picture.TerminalOptions{}),
		zones:   zone.New(), width: 80,
	}
	for i := range d.charts {
		d.charts[i] = ntcharts3d.New(80, 10,
			ntcharts3d.WithTerminalSession(d.session),
			ntcharts3d.WithZoneManager(d.zones),
			ntcharts3d.WithRenderMode(mode),
			ntcharts3d.WithAutoRotate(true))
	}
	// Setters run before Init, so no render commands are scheduled yet.
	cloud := ntcharts3d.Scatter{}
	for i := range 800 {
		t := float64(i) * .04
		cloud.X = append(cloud.X, float32(math.Cos(t)))
		cloud.Y = append(cloud.Y, float32(math.Sin(t)))
		cloud.Z = append(cloud.Z, float32(t/16-1))
	}
	d.charts[0].SetScatter("helix", cloud)
	d.charts[1].SetSurface("ripple", ntcharts3d.Surface{NX: 32, NY: 32,
		Sampler: func(x, y float64) float64 { return math.Cos(8*math.Hypot(x, y)) * .4 }})
	return d
}

func (d *dashboard) Init() tea.Cmd {
	var timeout tea.Cmd
	if d.duration > 0 {
		timeout = tea.Tick(d.duration, func(time.Time) tea.Msg { return quitMsg{} })
	}
	// Both Init calls reach the same session; discovery runs only once.
	return tea.Batch(d.charts[0].Init(), d.charts[1].Init(), timeout)
}

func (d *dashboard) dispose() tea.Cmd {
	// Sequence waits for both placement cleanup commands before quitting.
	return tea.Sequence(d.charts[0].Dispose(), d.charts[1].Dispose(), tea.Quit)
}

func (d *dashboard) close() {
	for _, chart := range d.charts {
		_ = chart.Close()
	}
	d.session.Close()
	d.zones.Close()
}

func (d *dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case quitMsg:
		return d, d.dispose()
	case tea.WindowSizeMsg:
		d.width = max(1, v.Width)
		height := max(6, v.Height-3)
		return d, tea.Batch(d.charts[0].SetSize(d.width, height/2), d.charts[1].SetSize(d.width, height-height/2))
	case tea.KeyPressMsg:
		switch v.String() {
		case "q", "ctrl+c":
			return d, d.dispose()
		case "tab":
			d.focus = 1 - d.focus
			return d, nil
		case "1", "2":
			d.focus = int(v.String()[0] - '1')
			return d, nil
		}
		_, cmd := d.charts[d.focus].Update(msg)
		return d, cmd
	}
	// Broadcast terminal replies, session changes, render completions, and
	// mouse events. Each chart filters its own messages and mouse zone.
	// Keep forwarding late completions even after Dispose.
	var cmds []tea.Cmd
	for _, chart := range d.charts {
		_, cmd := chart.Update(msg)
		cmds = append(cmds, cmd)
	}
	return d, tea.Batch(cmds...)
}

func (d *dashboard) View() tea.View {
	state := d.session.Snapshot()
	clip := func(s string) string { return ansi.Truncate(s, d.width, "") }
	parts := []string{clip(fmt.Sprintf("Shared session · cell %d×%d px · focus %d · tab select · q quit", state.CellPixelWidth, state.CellPixelHeight, d.focus+1))}
	for i, chart := range d.charts {
		parts = append(parts, clip(fmt.Sprintf("%d %s", i+1, []string{"Helix scatter", "Ripple surface"}[i])), chart.View().Content)
	}
	v := tea.NewView(d.zones.Scan(strings.Join(parts, "\n")))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion
	return v
}

func run() error {
	mode := flag.String("render-mode", "gpu", "gpu, software, or wireframe")
	duration := flag.Duration("duration", 0, "quit automatically after duration")
	flag.Parse()
	modes := map[string]ntcharts3d.RenderMode{"gpu": ntcharts3d.WebGPU, "software": ntcharts3d.Software, "wireframe": ntcharts3d.Wireframe}
	selected, ok := modes[*mode]
	if !ok || *duration < 0 {
		return fmt.Errorf("invalid render mode or duration")
	}
	d := newDashboard(selected)
	defer d.close()
	d.duration = *duration
	for _, chart := range d.charts {
		if err := chart.Err(); err != nil {
			return err
		}
	}
	_, err := tea.NewProgram(d).Run()
	return err
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
