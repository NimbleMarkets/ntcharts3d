// ntcharts3d - Copyright (c) 2026 Neomantra Corp.
package ntcharts3d

import (
	"bytes"
	"errors"
	"image"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"nimbleterminal.dev/nt3d"
)

// Deliberately unsafe engine: the adapter must serialize even older engines.
type gatedSoftware struct {
	entered chan struct{}
	release chan struct{}
	calls   atomic.Int32
	closes  atomic.Int32
	active  atomic.Int32
	overlap atomic.Bool
}

func (g *gatedSoftware) Render(f nt3d.Frame) (image.Image, error) {
	if g.active.Add(1) != 1 {
		g.overlap.Store(true)
	}
	defer g.active.Add(-1)
	if g.calls.Add(1) == 1 {
		close(g.entered)
		<-g.release
	}
	result, err := nt3d.RenderSoftware(f)
	return result.Image, err
}
func (g *gatedSoftware) Close() error {
	if g.active.Load() != 0 {
		g.overlap.Store(true)
	}
	g.closes.Add(1)
	return nil
}
func gate() *gatedSoftware {
	return &gatedSoftware{entered: make(chan struct{}), release: make(chan struct{})}
}

func TestSoftwareSnapshotOverlapsScheduledRender(t *testing.T) {
	m := bars(t, WithRenderMode(Software))
	g := gate()
	m.software.engine = g
	m.started = true
	cmd := m.schedule()
	done := make(chan struct{})
	go func() { cmd(); close(done) }()
	<-g.entered
	// Snapshot is on the event-loop caller, while the render command is running.
	// Release that command from a worker; Snapshot must wait on the same adapter.
	go func() { time.Sleep(10 * time.Millisecond); close(g.release) }()
	img, mode, err := m.Snapshot(120, 80)
	<-done
	if err != nil || mode != Software || img == nil || g.overlap.Load() || g.calls.Load() != 2 {
		t.Fatalf("overlap: calls=%d err=%v", g.calls.Load(), err)
	}
	pixels := slices.Clone(img.(*image.RGBA).Pix)
	m.Snapshot(80, 120)
	m.Close()
	m.Close()
	if !bytes.Equal(img.(*image.RGBA).Pix, pixels) || g.closes.Load() != 1 {
		t.Fatal("result ownership or idempotent Close failed")
	}
}
func TestSoftwareCloseWaitsAndRejectsQueuedWork(t *testing.T) {
	m := bars(t, WithRenderMode(Software))
	g := gate()
	m.software.engine = g
	m.started = true
	queued := m.schedule()
	f := m.frameSized(100, 80, 1.25, Software)
	done := make(chan struct{})
	go func() { m.software.Render(f); close(done) }()
	<-g.entered
	go func() { time.Sleep(10 * time.Millisecond); close(g.release) }()
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	<-done
	msg := queued().(rendered)
	if msg.err == nil || msg.frame != nil || g.calls.Load() != 1 || g.overlap.Load() {
		t.Fatal("queued work accessed closed renderer")
	}
	if _, cmd := m.Update(msg); cmd != nil {
		t.Fatal("late result escaped Dispose")
	}
	if err := m.Close(); err != nil || g.closes.Load() != 1 {
		t.Fatal("repeated Close")
	}
}
func TestSoftwareSurvivesGPUFallbackCleanup(t *testing.T) {
	m := bars(t, WithRenderer(&recorder{err: errors.New("GPU failure")}))
	m.started = true
	msg := m.schedule()().(rendered)
	if msg.renderMode != Software || msg.frame == nil {
		t.Fatal("no software fallback")
	}
	engine := m.software.engine
	m.Update(msg)
	if m.renderer != nil || m.software.closed || m.software.engine != engine {
		t.Fatal("GPU cleanup disposed software renderer")
	}
	if _, _, err := m.Snapshot(120, 80); err != nil {
		t.Fatal(err)
	}
	if _, err := m.software.Render(Frame{}); err == nil {
		t.Fatal("invalid frame accepted")
	}
	if _, _, err := m.Snapshot(80, 120); err != nil {
		t.Fatal(err)
	}
}

func TestSoftwareRetainedChartPixels(t *testing.T) {
	for _, work := range []string{"camera", "bars"} {
		p, _ := presentationFixture(t, "glyph", work, 80, 22)
		m := p.m
		for i, size := range []image.Point{{160, 80}, {80, 160}, {320, 160}, {80, 40}} {
			m.camera.Beta += float64(i) * 25
			frame := m.frameSized(size.X, size.Y, float32(size.X)/float32(size.Y), Software)
			frame.complete = true
			fresh, err := renderSoftware(frame)
			if err != nil {
				t.Fatal(err)
			}
			retained, mode, err := m.Snapshot(size.X, size.Y)
			if err != nil || mode != Software || !same(fresh, retained) {
				t.Fatal("retained chart differs", work, size, err)
			}
		}
	}
}
