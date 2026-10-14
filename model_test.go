package ntcharts3d

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	nt3dmath "nimbleterminal.dev/nt3d/math"
	picture "nimbleterminal.dev/ntpicture"
)

type failedRenderer struct{ closed bool }

func TestContinuousColorLegendShowsMappedRange(t *testing.T) {
	m := New(80, 24, WithRenderMode(Software), WithColorLegend("temperature"))
	defer m.Close()
	m.SetScatter("measurements", Scatter{
		X: []float32{0, 1}, Y: []float32{0, 1}, Z: []float32{-2, 4},
	})
	f := m.frame()
	if len(f.ColorLegends) != 1 || f.ColorLegends[0].Label != "temperature" || f.ColorLegends[0].Min != -2 || f.ColorLegends[0].Max != 4 {
		t.Fatalf("wrong color legend metadata: %+v", f.ColorLegends)
	}
	m.SetColorLegendVisible(false)
	if got := m.frame().ColorLegends; len(got) != 0 {
		t.Fatalf("disabled color legend is still in the frame: %+v", got)
	}
}

func TestSharedColorDomainMapsSeriesAndLegend(t *testing.T) {
	m := New(80, 24, WithRenderMode(Software), WithSharedColorDomain(true))
	m.SetScatter("low", Scatter{X: []float32{0, 1}, Y: []float32{0, 1}, Z: []float32{0, 1}, ColorValue: []float32{0, 10}})
	m.SetScatter("high", Scatter{X: []float32{0, 1}, Y: []float32{0, 1}, Z: []float32{2, 3}, ColorValue: []float32{10, 20}})
	if got := m.series[0].geometry.Points[1].Color; got != m.colors.At(.5) {
		t.Fatalf("low-series max should map to middle of shared range, got %v", got)
	}
	if got := m.series[1].geometry.Points[0].Color; got != m.colors.At(.5) {
		t.Fatalf("high-series min should map to middle of shared range, got %v", got)
	}
	f := m.frame()
	if len(f.ColorLegends) != 1 || f.ColorLegends[0].Min != 0 || f.ColorLegends[0].Max != 20 {
		t.Fatalf("expected one shared 0..20 legend, got %+v", f.ColorLegends)
	}
	m.SetSharedColorDomain(false)
	if got := m.series[0].geometry.Points[1].Color; got != m.colors.At(1) {
		t.Fatalf("per-series max should map to palette max when disabled, got %v", got)
	}
}

func TestSoftwareRendererDrawsColorLegendInsideFrame(t *testing.T) {
	palette := defaultPalette()
	f := Frame{Width: 300, Height: 200, Background: color.RGBA{235, 238, 242, 255}, ColorLegends: []ColorLegend{{Label: "signal", Min: -1, Max: 1, Palette: palette}}}
	img := softwareRender(f)
	for _, sample := range []struct {
		x, y int
		want color.RGBA
	}{{276, 29, palette.At(1)}, {276, 83, palette.At(0)}} {
		if got := color.RGBAModel.Convert(img.At(sample.x, sample.y)).(color.RGBA); got != sample.want {
			t.Fatalf("legend pixel (%d,%d) = %v, want %v", sample.x, sample.y, got, sample.want)
		}
	}
}

func TestColorLegendModeToggle(t *testing.T) {
	m := New(80, 24, WithRenderMode(Software))
	defer m.Close()
	m.SetScatter("measurements", Scatter{X: []float32{0, 1}, Y: []float32{0, 1}, Z: []float32{-2, 4}})
	if len(m.frame().ColorLegends) != 1 {
		t.Fatal("image legend should be the default")
	}
	m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	if m.ColorLegendMode() != ColorLegendText || len(m.frame().ColorLegends) != 0 {
		t.Fatal("v did not switch to the terminal legend")
	}
	if view := m.View().Content; !strings.Contains(view, "measurements") {
		t.Fatalf("terminal legend missing: %q", view)
	}
	m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	if m.ColorLegendMode() != ColorLegendImage || len(m.frame().ColorLegends) != 1 {
		t.Fatal("v did not restore the in-image legend")
	}
}

func (*failedRenderer) Render(Frame) (image.Image, error) { return nil, errors.New("no adapter") }
func (r *failedRenderer) Close() error                    { r.closed = true; return nil }
func TestAutomaticDemotionAndIdle(t *testing.T) {
	r := &failedRenderer{}
	m := New(40, 20, WithRenderer(r))
	defer m.Close()
	m.SetScatter("p", Scatter{X: []float32{0}, Y: []float32{0}, Z: []float32{0}})
	m.started = true
	cmd := m.schedule()
	msg := cmd().(rendered)
	if msg.renderMode != Software || msg.frame == nil {
		t.Fatal("no fallback")
	}
	_, present := m.Update(msg)
	drain(t, m, present)
	if !r.closed || m.RenderMode() != Software {
		t.Fatal("failed GPU not released")
	}
	if m.RequestedRenderMode() != WebGPU {
		t.Fatal("GPU fallback lost the requested mode")
	}
	_, next := m.Update(picture.PresentedMsg{Owner: m.pic, Generation: m.pic.Status().Generation})
	if next != nil {
		t.Fatal("idle scene scheduled another frame")
	}
	m.SetRenderMode(WebGPU)
	if m.RenderMode() != WebGPU || m.RequestedRenderMode() != WebGPU || m.renderer == nil {
		t.Fatal("explicit WebGPU selection did not prepare a retry")
	}
}

func TestStaleGPUFailureDemotesBeforeScheduling(t *testing.T) {
	r := &failedRenderer{}
	m := New(40, 20, WithRenderer(r))
	defer m.Close()
	m.started = true
	first := m.schedule()
	m.SetCamera(Camera{Alpha: 45, Distance: 3})
	msg := first().(rendered)
	if msg.seq == m.seq || msg.err == nil {
		t.Fatal("expected a stale GPU failure")
	}
	_, next := m.Update(msg)
	if !r.closed || m.renderer != nil || m.RenderMode() != Software {
		t.Fatal("stale GPU failure did not release and demote the renderer")
	}
	if m.Err() != msg.err || m.RequestedRenderMode() != WebGPU {
		t.Fatal("fallback lost the error or requested mode")
	}
	if next == nil {
		t.Fatal("stale failure did not schedule the changed chart")
	}
	frame := next().(rendered)
	if frame.seq != m.seq || frame.renderMode != Software || frame.err != nil || frame.frame == nil {
		t.Fatalf("next frame did not render successfully in software: %+v", frame)
	}
}

func TestSlowSoftwareFallbackPreservesRequestedMode(t *testing.T) {
	m := New(40, 20, WithRenderMode(Software))
	defer m.Close()
	frame := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for range 3 {
		m.Update(rendered{id: m.id, seq: m.seq, frame: frame, renderMode: Software, elapsed: 151 * time.Millisecond})
	}
	if m.RenderMode() != Wireframe || m.RequestedRenderMode() != Software {
		t.Fatalf("slow fallback: active=%v requested=%v", m.RenderMode(), m.RequestedRenderMode())
	}
	m.SetRenderMode(RenderMode(255))
	if m.RenderMode() != Wireframe || m.RequestedRenderMode() != Software {
		t.Fatal("invalid mode changed fallback state")
	}
	m.SetRenderMode(Software)
	if m.RenderMode() != Software || m.RequestedRenderMode() != Software {
		t.Fatal("explicit selection did not reset fallback")
	}
}

func TestDataCopiedAndCameraDoesNotChangeRevision(t *testing.T) {
	m := New(40, 20, WithRenderMode(Software))
	defer m.Close()
	xs := []float32{1, 2}
	m.SetScatter("a", Scatter{X: xs, Y: []float32{0, 0}, Z: []float32{0, 0}})
	xs[0] = 100
	if m.series[0].geometry.Points[0].Position.X != 1 {
		t.Fatal("caller mutation changed scene")
	}
	rev := m.revision
	m.SetCamera(DefaultCamera())
	if m.revision != rev {
		t.Fatal("camera dirtied data")
	}
	m.SetScatter("bad", Scatter{X: []float32{1}})
	if m.Err() == nil || len(m.series) != 1 {
		t.Fatal("invalid data partially applied")
	}
}

func TestPointPickingAndRayBox(t *testing.T) {
	m := New(41, 23, WithRenderMode(Software))
	defer m.Close()
	m.SetScatter("a", Scatter{X: []float32{0}, Y: []float32{0}, Z: []float32{0}, Labels: []string{"center"}})
	p := m.Pick(20, 10)
	if p == nil || p.SeriesIndex != 0 || p.SeriesName != "a" || p.Datum != 0 || p.Label != "center" {
		t.Fatalf("center pick: %+v", p)
	}
	if m.Pick(-1, 0) != nil {
		t.Fatal("outside pick")
	}
	d, ok := rayBox(nt3dmath.Ray{Origin: nt3dmath.Vec3{X: 0, Y: 0, Z: 3}, Direction: nt3dmath.Vec3{X: 0, Y: 0, Z: -1}}, Box{Min: nt3dmath.Vec3{X: -1, Y: -1, Z: -1}, Size: nt3dmath.Vec3{X: 2, Y: 2, Z: 2}})
	if !ok || d != 2 {
		t.Fatal("ray box")
	}
}

func TestSingleFlightAndForeignMessages(t *testing.T) {
	m := New(40, 20, WithRenderMode(Software))
	defer m.Close()
	m.started = true
	first := m.schedule()
	if first == nil {
		t.Fatal("no initial render")
	}
	m.SetCamera(Camera{Alpha: 45, Distance: 3})
	if m.schedule() != nil {
		t.Fatal("second frame in flight")
	}
	_, cmd := m.Update(first())
	if cmd == nil {
		t.Fatal("stale completion lost dirty frame")
	}
	m.Update(rendered{id: m.id + 1})
	if !m.busy {
		t.Fatal("foreign completion cleared busy")
	}
}

func TestGlyphToggleDoesNotChangeRenderMode(t *testing.T) {
	m := New(40, 20, WithRenderMode(Software))
	defer m.Close()
	m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if m.RenderMode() != Software {
		t.Fatal("toggle changed renderer")
	}
}

func TestBarNegativeExtentAndCategories(t *testing.T) {
	m := New(40, 20, WithRenderMode(Software))
	defer m.Close()
	m.SetAxes(Axes{X: Axis{Categories: []string{"a", "b"}}, Y: Axis{Categories: []string{"week"}}})
	m.SetBars("bars", Bars{NX: 2, NY: 1, Values: []float32{-2, 3}})
	b := m.bounds()
	if b.Min.Z != -2 || b.Max.Z != 3 {
		t.Fatal("bar baseline omitted")
	}
	if m.datumLabel(0, 1) != "b / week" {
		t.Fatal("category label missing")
	}
}

func TestMousePickAndOrbit(t *testing.T) {
	m := New(41, 23, WithRenderMode(Software))
	defer m.Close()
	m.SetScatter("a", Scatter{X: []float32{0}, Y: []float32{0}, Z: []float32{0}})
	m.View() // scanning is asynchronous: allow the manager to publish its zones
	for i := 0; i < 100 && !m.zones.Get(m.zoneID).InBounds(tea.MouseClickMsg{X: 20, Y: 11, Button: tea.MouseLeft}); i++ {
		time.Sleep(time.Millisecond)
	}
	m.mouse(tea.MouseClickMsg{X: 20, Y: 11, Button: tea.MouseLeft})
	cmd := m.mouse(tea.MouseReleaseMsg{X: 20, Y: 11, Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatal("click produced no pick")
	}
	if p, ok := cmd().(PickMsg); !ok || p.Datum != 0 || p.SeriesName != "a" || p.SeriesIndex != 0 || p.ModelID != m.ID() {
		t.Fatal("wrong pick")
	}
	beta := m.camera.Beta
	m.mouse(tea.MouseClickMsg{X: 20, Y: 11, Button: tea.MouseLeft})
	m.mouse(tea.MouseMotionMsg{X: 23, Y: 11, Button: tea.MouseLeft})
	if m.camera.Beta == beta {
		t.Fatal("drag did not orbit")
	}
	if m.mouse(tea.MouseReleaseMsg{X: 23, Y: 11, Button: tea.MouseLeft}) != nil {
		t.Fatal("drag emitted click")
	}
}

func TestKittyEncoding(t *testing.T) {
	for _, format := range []picture.KittyFormat{picture.KittyFormatZlib, picture.KittyFormatPNG, picture.KittyFormatRGBA} {
		t.Run(format.String(), func(t *testing.T) {
			session := picture.NewTerminalSession(picture.TerminalOptions{Kitty: picture.KittyCapabilitySupported, SharedMemory: picture.KittyCapabilityUnsupported})
			defer session.Close()
			m := New(8, 6, WithRenderMode(Software), WithTerminalSession(session), WithKittyFormat(format), WithKittyMedium(picture.KittyMediumSharedMemory))
			defer m.Close()
			m.started = true
			_, cmd := m.Update(struct{}{})
			messages := drain(t, m, cmd)
			status := m.PictureStatus()
			if status.Busy || status.Presented == 0 || status.Format != format || status.Medium != picture.KittyMediumDirect {
				t.Fatalf("unexpected presentation: %+v", status)
			}
			found := false
			for _, msg := range messages {
				if raw, ok := msg.(tea.RawMsg); ok && strings.Contains(fmt.Sprint(raw.Msg), "\x1b_G") {
					found = true
				}
			}
			if !found {
				t.Fatal("no Kitty output")
			}
		})
	}
}
