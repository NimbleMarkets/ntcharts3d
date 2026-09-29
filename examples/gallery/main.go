// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package main

import (
	tea "charm.land/bubbletea/v2"
	"flag"
	"fmt"
	booba "github.com/NimbleMarkets/go-booba"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/NimbleMarkets/ntcharts3d"
	"github.com/NimbleMarkets/ntcharts3d/math3d"
	"github.com/aquilax/go-perlin"
	zone "github.com/lrstanley/bubblezone/v2"
	"math"
	"math/rand/v2"
	"os"
	"runtime"
	"time"
)

type gallery struct {
	mapTextures     [2]*ntcharts3d.Texture
	mapStyle        int
	flatMap, mapLit bool
	scene           *ntcharts3d.Model
	zones           *zone.Manager
	tab             int
	points          int
	duration        time.Duration
	pick            string
}

type quitMsg struct{}

func (g *gallery) load() tea.Cmd {
	g.pick = ""
	clear := g.scene.Clear()
	axes := g.scene.SetAxes(ntcharts3d.Axes{})
	aspect := g.scene.SetPlotAspect(math3d.Vec3{})
	domain := g.scene.SetColorDomain(&ntcharts3d.Range{Min: -1, Max: 1})
	switch g.tab {
	case 0:
		d := ntcharts3d.Scatter{}
		rng := rand.New(rand.NewPCG(2, 7))
		for i := 0; i < g.points; i++ {
			z := rng.Float64()*2 - 1
			t := rng.Float64() * 2 * math.Pi
			r := math.Sqrt(1-z*z) * (1 + .08*rng.NormFloat64())
			d.X = append(d.X, float32(r*math.Cos(t)))
			d.Y = append(d.Y, float32(r*math.Sin(t)))
			d.Z = append(d.Z, float32(z))
			d.ColorValue = append(d.ColorValue, float32(z))
		}
		return tea.Batch(clear, axes, aspect, domain, g.scene.SetScatter("noisy sphere", d))
	case 1:
		p := perlin.NewPerlin(2, 2, 3, 1)
		axes = g.scene.SetAxes(ntcharts3d.Axes{Z: ntcharts3d.Axis{Name: "Height (µm)", Range: &ntcharts3d.Range{Min: -1, Max: 1}}})
		return tea.Batch(clear, axes, aspect, domain, g.scene.SetSurface("Perlin surface", ntcharts3d.Surface{NX: 96, NY: 96, Sampler: func(x, y float64) float64 { return p.Noise2D(x*2, y*2) }, Wireframe: false}))
	case 3:
		return tea.Batch(clear, axes, aspect, domain, g.loadTerrain())
	default:
		d := ntcharts3d.Bars{NX: 12, NY: 7}
		var hours []string
		for x := 0; x < 12; x++ {
			hours = append(hours, fmt.Sprintf("%02d:00", x*2))
		}
		axes = g.scene.SetAxes(ntcharts3d.Axes{
			X: ntcharts3d.Axis{Name: "Hour", Categories: hours},
			Y: ntcharts3d.Axis{Name: "Day", Categories: []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}},
			Z: ntcharts3d.Axis{Name: "Value", Range: &ntcharts3d.Range{Min: 0, Max: 4}},
		})
		aspect = g.scene.SetPlotAspect(math3d.Vec3{X: 1, Y: .7, Z: .6})
		domain = g.scene.SetColorDomain(&ntcharts3d.Range{Min: 0, Max: 4})
		for y := 0; y < 7; y++ {
			for x := 0; x < 12; x++ {
				v := .3 + 2.3*math.Exp(-math.Pow(float64(x-6), 2)/9)*(1+float64(y%3)*.3)
				d.Values = append(d.Values, float32(v))
			}
		}
		return tea.Batch(clear, axes, aspect, domain, g.scene.SetBars("hour × weekday", d))
	}
}

func (g *gallery) Init() tea.Cmd {
	var quit tea.Cmd
	if g.duration > 0 {
		quit = tea.Tick(g.duration, func(time.Time) tea.Msg { return quitMsg{} })
	}
	return tea.Batch(g.scene.Init(), quit)
}

func (g *gallery) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case quitMsg:
		return g, tea.Quit
	case ntcharts3d.PickMsg:
		g.pick = fmt.Sprintf("Picked %s (series %d) datum %d %s", v.SeriesName, v.SeriesIndex, v.Datum, v.Label)
		return g, nil
	case tea.WindowSizeMsg:
		return g, g.scene.SetSize(v.Width, max(3, v.Height-2))
	case tea.KeyPressMsg:
		switch v.String() {
		case "q", "ctrl+c":
			return g, tea.Quit
		case "tab":
			g.tab = (g.tab + 1) % 4
			return g, g.load()
		case "b":
			grid := g.scene.Grid()
			grid.Show = !grid.Show
			return g, g.scene.SetGrid(grid)
		case "t":
			if g.tab == 3 {
				g.mapStyle = 1 - g.mapStyle
				return g, g.scene.SetSeriesTexture(terrainName, g.mapTextures[g.mapStyle])
			}
		case "f":
			if g.tab == 3 {
				g.flatMap = !g.flatMap
				return g, g.setTerrain()
			}
		case "l":
			if g.tab == 3 {
				g.mapLit = !g.mapLit
				return g, g.setTerrain()
			}
		case "1", "2", "3", "4":
			g.tab = int(v.String()[0] - '1')
			return g, g.load()
		}
	}
	_, cmd := g.scene.Update(msg)
	return g, cmd
}

func (g *gallery) View() tea.View {
	v := g.scene.View()
	title := fmt.Sprintf("1 scatter · 2 surface · 3 bars · 4 map · tab switch · b grid · q quit   %s", g.pick)
	b := g.scene.Bounds()
	caption := fmt.Sprintf("X %.2g…%.2g  Y %.2g…%.2g  Z↑ %.2g…%.2g", b.Min.X, b.Max.X, b.Min.Y, b.Max.Y, b.Min.Z, b.Max.Z)
	if g.tab == 3 {
		caption = g.terrainCaption()
	}
	v.Content = g.zones.Scan(title + "\n" + v.Content + "\n" + caption)
	v.AltScreen = true
	return v
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	holdRuntime()
}

func run() error {
	renderMode := flag.String("render-mode", "gpu", "gpu, software, or wireframe")
	medium := flag.String("medium", "auto", "auto, direct or shm (native shm requires a local terminal)")
	points := flag.Int("points", 16000, "scatter points (1..500000)")
	tab := flag.Int("tab", 1, "initial gallery (1..4)")
	duration := flag.Duration("duration", 0, "quit automatically after duration")
	rotate := flag.Bool("rotate", true, "initial auto-rotation")
	flag.Parse()
	if *points < 1 || *points > ntcharts3d.MaxPoints || *tab < 1 || *tab > 4 || *duration < 0 {
		return fmt.Errorf("invalid points, tab, or duration")
	}
	l := ntcharts3d.WebGPU
	switch *renderMode {
	case "gpu":
	case "software":
		l = ntcharts3d.Software
	case "wireframe":
		l = ntcharts3d.Wireframe
	default:
		return fmt.Errorf("invalid render mode")
	}
	m := picture.KittyMediumDirect
	switch *medium {
	case "auto":
		if runtime.GOOS == "js" {
			m = picture.KittyMediumSharedMemory
		}
	case "direct":
	case "shm":
		m = picture.KittyMediumSharedMemory
	default:
		return fmt.Errorf("invalid medium")
	}
	zones := zone.New()
	defer zones.Close()
	s := ntcharts3d.New(80, 24, ntcharts3d.WithRenderMode(l), ntcharts3d.WithAutoRotate(*rotate), ntcharts3d.WithKittyMedium(m), ntcharts3d.WithSharedColorDomain(true), ntcharts3d.WithGrid(ntcharts3d.Grid{Show: true}), ntcharts3d.WithEmphasis(ntcharts3d.Emphasis{ShowLabel: true}), ntcharts3d.WithZoneManager(zones))
	defer s.Close()
	g := &gallery{scene: s, zones: zones, tab: *tab - 1, points: *points, duration: *duration}
	g.load()
	if s.Err() != nil {
		return s.Err()
	}
	if runtime.GOOS == "js" {
		// TinyGo 0.42 cannot use Bubble Tea's signal handler in the browser.
		return booba.Run(g, tea.WithoutSignalHandler())
	}
	return booba.Run(g)
}
