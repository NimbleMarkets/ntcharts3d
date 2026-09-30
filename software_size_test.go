// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"image"
	"testing"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/picture"
)

// kitty is a chart drawn in software for a terminal that shows pictures.
// At 100 by 40 cells of 8 by 16 pixels, its plot is 800 by 608.
func kitty(t *testing.T) *Model {
	t.Helper()
	old := picture.KittySupported()
	picture.ForceKittyCapability(picture.KittyCapabilitySupported)
	t.Cleanup(func() { picture.ForceKittyCapability(old) })
	m := New(100, 40, WithRenderMode(Software))
	t.Cleanup(func() { m.Close() })
	m.togglePicture()
	if m.pic.Mode() != picture.PictureKitty {
		t.Fatal("not in Kitty mode")
	}
	return m
}

// took tells the chart that the frame it asked for took this long to draw.
func took(m *Model, elapsed time.Duration) {
	f := m.frame()
	m.Update(rendered{id: m.id, seq: m.seq, frame: image.NewRGBA(image.Rect(0, 0, f.Width, f.Height)), renderMode: Software, elapsed: elapsed})
}

func area(m *Model) int { f := m.frame(); return f.Width * f.Height }

func TestSoftwareDrawsPicturesAtFullSize(t *testing.T) {
	m := kitty(t)
	if f := m.frame(); f.Width != 800 || f.Height != 608 {
		t.Fatalf("a Kitty frame is %d×%d, want the plot's 800×608", f.Width, f.Height)
	}
	// Glyphs are two pixels to a cell: more would be thrown away.
	glyph := New(100, 40, WithRenderMode(Software))
	defer glyph.Close()
	if f := glyph.frame(); f.Width > 320 || f.Height > 200 {
		t.Fatalf("a glyph frame is %d×%d", f.Width, f.Height)
	}
}

func TestSlowSoftwareFramesShrinkBeforeGivingUp(t *testing.T) {
	m := kitty(t)
	full := area(m)
	took(m, 200*time.Millisecond)
	if got := area(m); got >= full || m.RenderMode() != Software {
		t.Fatalf("after a slow frame: area %d of %d, mode %v", got, full, m.RenderMode())
	}
	// A frame five times too slow is cut to about a fifth, not by a step.
	if got := area(m); got > full/3 || got < full/8 {
		t.Fatalf("area %d of %d", got, full)
	}
	for range 20 {
		took(m, 200*time.Millisecond)
		if f := m.frame(); f.Width <= 320 && f.Height <= 200 {
			break
		}
	}
	f := m.frame()
	if f.Width > 320 || f.Height > 200 || f.Width < 200 || m.RenderMode() != Software {
		t.Fatalf("the smallest frame is %d×%d, mode %v", f.Width, f.Height, m.RenderMode())
	}
	// Only at its smallest does a slow frame count toward wireframe.
	took(m, 200*time.Millisecond)
	took(m, 200*time.Millisecond)
	if m.RenderMode() != Software {
		t.Fatal("gave up after two slow frames")
	}
	took(m, 200*time.Millisecond)
	if m.RenderMode() != Wireframe || m.RequestedRenderMode() != Software {
		t.Fatalf("mode %v, requested %v", m.RenderMode(), m.RequestedRenderMode())
	}
}

func TestFastSoftwareFramesGrowBack(t *testing.T) {
	m := kitty(t)
	full := area(m)
	took(m, 400*time.Millisecond)
	small := area(m)
	m.dirty = false // As once the frame has been shown.
	took(m, 30*time.Millisecond)
	if area(m) != small || m.dirty {
		t.Fatal("a frame that took neither long nor no time at all changed the size")
	}
	took(m, 5*time.Millisecond)
	if got := area(m); got <= small || !m.dirty {
		t.Fatalf("a fast frame: area %d after %d, dirty=%v: the picture should be drawn again, larger", got, small, m.dirty)
	}
	for range 20 {
		took(m, 5*time.Millisecond)
	}
	if got := area(m); got != full {
		t.Fatalf("area %d, want the full %d", got, full)
	}
	// At full size there is nothing more to draw.
	m.dirty = false
	took(m, 5*time.Millisecond)
	if m.dirty || area(m) != full {
		t.Fatal("a chart at rest went on drawing")
	}
}

func TestChoosingSoftwareStartsAtFullSize(t *testing.T) {
	m := kitty(t)
	full := area(m)
	took(m, 400*time.Millisecond)
	m.SetRenderMode(Software)
	if got := area(m); got != full {
		t.Fatalf("area %d, want %d", got, full)
	}
}
