package main

import (
	"image"
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
	picture "nimbleterminal.dev/ntpicture"

	gpuimage "nimbleterminal.dev/ntgpuimage"
)

type fakeRenderer struct {
	calls []frameRequest
}

func (f *fakeRenderer) Render(req frameRequest) (*image.NRGBA, gpuimage.Stats, error) {
	f.calls = append(f.calls, req)
	return image.NewNRGBA(image.Rect(0, 0, req.width, req.height)), gpuimage.Stats{Seq: uint64(len(f.calls))}, nil
}
func (f *fakeRenderer) Name() string { return "fake" }
func (f *fakeRenderer) Close()       {}

func sized(t *testing.T, w, h int) (*model, *fakeRenderer) {
	t.Helper()
	fr := &fakeRenderer{}
	m := newModel(fr, picture.KittyFormatPNG, picture.KittyMediumDirect, 60, 16)
	t.Cleanup(func() { m.Dispose() })
	m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return m, fr
}

func TestGeometryReservesChromeRows(t *testing.T) {
	m, _ := sized(t, 80, 24)
	if m.cols != 80 || m.rows != 20 {
		t.Fatalf("cols,rows = %d,%d want 80,20", m.cols, m.rows)
	}
	m.fullscreen = true
	m.geometry()
	if m.cols != 80 || m.rows != 24 {
		t.Fatalf("fullscreen cols,rows = %d,%d want 80,24", m.cols, m.rows)
	}
}

func TestGeometryTinyTerminal(t *testing.T) {
	m, fr := sized(t, 10, 3)
	if m.rows < 1 || m.cols < 1 {
		t.Fatalf("rows,cols = %d,%d; must stay >= 1", m.rows, m.cols)
	}
	// sized() left a render in flight; clear it so render() issues a new one.
	m.busy = false
	cmd := m.render()
	if cmd == nil {
		t.Fatal("render returned nil for a 10x3 terminal; expected a small raster, not a skip")
	}
	cmd()
	if len(fr.calls) == 0 {
		t.Fatal("renderer was never called")
	}
	for _, c := range fr.calls {
		if c.width < 1 || c.height < 1 {
			t.Fatalf("submitted a %dx%d raster", c.width, c.height)
		}
	}
}

func TestRenderIsSingleInFlight(t *testing.T) {
	m, _ := sized(t, 80, 24)
	m.busy = false
	if m.render() == nil {
		t.Fatal("first render returned nil cmd")
	}
	if m.render() != nil {
		t.Fatal("second render while busy should return nil")
	}
}

func TestStaleFrameIsDroppedAndRerendered(t *testing.T) {
	m, fr := sized(t, 80, 24)
	m.busy = true
	m.epoch = 7
	calls := len(fr.calls)
	_, cmd := m.Update(renderedMsg{epoch: 6, image: image.NewNRGBA(image.Rect(0, 0, 4, 4))})
	if cmd == nil {
		t.Fatal("stale frame should schedule a fresh render")
	}
	if m.frames != 0 {
		t.Fatal("stale frame was counted as presented")
	}
	// The rescheduled render is the new in-flight frame and carries the
	// current epoch, so it will be accepted when it lands.
	if !m.busy {
		t.Fatal("rescheduled render did not mark the model busy")
	}
	fresh, ok := cmd().(renderedMsg)
	if !ok || fresh.epoch != 7 {
		t.Fatalf("rescheduled render produced %#v, want renderedMsg with epoch 7", fresh)
	}
	if len(fr.calls) != calls+1 {
		t.Fatalf("renderer called %d times, want %d", len(fr.calls), calls+1)
	}
}

func TestOrbitKeysChangeCamera(t *testing.T) {
	m, _ := sized(t, 80, 24)
	yaw, pitch, dist := m.yaw, m.pitch, m.dist
	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m.Update(tea.KeyPressMsg{Code: '+', Text: "+"})
	if m.yaw == yaw || m.pitch == pitch || m.dist == dist {
		t.Fatalf("camera unchanged: yaw %v pitch %v dist %v", m.yaw, m.pitch, m.dist)
	}
	m.pitch = 10
	m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.pitch > 1.5 {
		t.Fatalf("pitch not clamped: %v", m.pitch)
	}
}

func TestMediumTogglePreservesDirectFormat(t *testing.T) {
	m := newModel(&fakeRenderer{}, picture.KittyFormatRGBA, picture.KittyMediumDirect, 60, 16)
	m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if m.medium != picture.KittyMediumSharedMemory || m.format != picture.KittyFormatRGBA {
		t.Fatal("toggle lost the direct format")
	}
	m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if m.medium != picture.KittyMediumDirect {
		t.Fatal("toggle did not return to direct")
	}
}

// Execute Bubble Tea batches/sequences and route their messages through Update.
// Reflection unwraps Tea's unexported sequence slice without depending on its name.
func drain(t *testing.T, m *model, cmd tea.Cmd) []tea.Msg {
	var messages []tea.Msg
	t.Helper()
	count := 0
	var run func(tea.Cmd)
	run = func(cmd tea.Cmd) {
		if cmd == nil {
			return
		}
		count++
		if count > 200 {
			t.Fatal("command loop did not settle")
		}
		msg := cmd()
		messages = append(messages, msg)
		if msg == nil {
			return
		}
		v := reflect.ValueOf(msg)
		if v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeOf(tea.Cmd(nil)) {
			for i := 0; i < v.Len(); i++ {
				run(v.Index(i).Interface().(tea.Cmd))
			}
			return
		}
		if _, ok := msg.(tea.RawMsg); ok {
			return
		}
		_, next := m.Update(msg)
		run(next)
	}
	run(cmd)
	return messages
}

func TestPictureLifecycleAndTransportSwitching(t *testing.T) {
	t.Setenv("NTPICTURE_KITTY", "supported")
	m := newModel(&fakeRenderer{}, picture.KittyFormatPNG, picture.KittyMediumDirect, 60, 16)
	defer m.Dispose()
	m.playing = false
	_, cmd := m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	drain(t, m, cmd)
	if s := m.pic.Status(); s.Busy || s.Presented == 0 || m.transport != "png" {
		t.Fatalf("initial frame: %+v path=%s", s, m.transport)
	}
	for _, key := range []rune{'t', 'm', 't', 'm', 't', 'm'} {
		_, cmd = m.Update(tea.KeyPressMsg{Code: key, Text: string(key)})
		drain(t, m, cmd)
		if s := m.pic.Status(); s.Busy || s.Presented == 0 || s.Format != m.format {
			t.Fatalf("switch %c: %+v", key, s)
		}
		for _, p := range m.pictures {
			if p != m.pic && p.View() != "" {
				t.Fatal("hidden transport retained its placement")
			}
		}
	}
	if len(m.pictures) > 4 {
		t.Fatal("transport toggles retained unbounded pictures")
	}
	drain(t, m, m.Dispose())
	for _, p := range m.pictures {
		if p.View() != "" || p.Status().Busy {
			t.Fatal("dispose did not settle")
		}
	}
}
