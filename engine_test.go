package ntcharts3d

import (
	"image"
	"image/color"
	"nimbleterminal.dev/nt3d"
	"testing"
)

type engineRecorder struct {
	frames []nt3d.Frame
	closed bool
}

func (r *engineRecorder) Render(f nt3d.Frame) (image.Image, error) {
	r.frames = append(r.frames, f)
	return image.NewRGBA(image.Rect(0, 0, f.Width, f.Height)), nil
}
func (r *engineRecorder) Close() error { r.closed = true; return nil }

func TestEngineAdapterPreservesIndependentRevisions(t *testing.T) {
	r := &engineRecorder{}
	m := New(40, 20, WithRenderer(&gpuRenderer{engine: r}))
	defer m.Close()
	first := testTexture(t, color.RGBA{R: 255, A: 255})
	next := testTexture(t, color.RGBA{G: 255, A: 255})
	m.SetSurface("terrain", Surface{NX: 3, NY: 3, Sampler: func(x, y float64) float64 { return x + y }, Material: &Material{Texture: first, Unlit: true}})
	snapshot := func() {
		t.Helper()
		if _, mode, err := m.Snapshot(100, 100); err != nil || mode != WebGPU {
			t.Fatalf("snapshot: %v %v", mode, err)
		}
	}
	snapshot()
	c := m.Camera()
	c.Beta += 30
	m.SetCamera(c)
	snapshot()
	m.SetSeriesTexture("terrain", next)
	snapshot()
	a, b, cframe := r.frames[0], r.frames[1], r.frames[2]
	if a.Revision != b.Revision || b.Revision != cframe.Revision || a.Matrix == b.Matrix {
		t.Fatal("camera/texture changed mesh revision or camera was lost")
	}
	if a.TextureRevision != b.TextureRevision || b.TextureRevision == cframe.TextureRevision {
		t.Fatal("texture revision not forwarded independently")
	}
	if &a.Geometry[0].Vertices[0] != &cframe.Geometry[0].Vertices[0] {
		t.Fatal("adapter copied mesh buffers for a texture change")
	}
	if a.Geometry[0].Material.Texture != first || cframe.Geometry[0].Material.Texture != next {
		t.Fatal("texture replacement mutated an earlier frame")
	}
	if !a.Complete || len(a.Overlays) == 0 {
		t.Fatal("snapshot completeness or chart decorations lost")
	}
	m.Close()
	if !r.closed {
		t.Fatal("engine renderer was not closed")
	}
}

// Test helpers fail immediately on invalid engine frames, rather than silently
// substituting blank images and hiding an adapter validation error.
func softwareRender(f Frame) image.Image {
	img, err := renderSoftware(f)
	if err != nil {
		panic(err)
	}
	return img
}
func wireRender(f Frame, cols, rows int) string {
	text, err := renderWireframe(f, cols, rows)
	if err != nil {
		panic(err)
	}
	return text
}
