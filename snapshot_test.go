// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"errors"
	"image"
	"image/color"
	"testing"
)

// recorder is a renderer that notes the frames it is given.
type recorder struct {
	frames []Frame
	err    error
}

func (r *recorder) Render(f Frame) (image.Image, error) {
	r.frames = append(r.frames, f)
	if r.err != nil {
		return nil, r.err
	}
	return image.NewRGBA(image.Rect(0, 0, f.Width, f.Height)), nil
}

func (r *recorder) Close() error { return nil }

func bars(t *testing.T, opts ...Option) *Model {
	t.Helper()
	m := New(80, 24, opts...)
	t.Cleanup(func() { m.Close() })
	m.SetBars("bars", Bars{NX: 2, NY: 2, Values: []float32{.2, .8, .4, .6}})
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
	return m
}

func drawn(img image.Image, background color.RGBA) int {
	n := 0
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if color.RGBAModel.Convert(img.At(x, y)).(color.RGBA) != background {
				n++
			}
		}
	}
	return n
}

func same(a, b image.Image) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			if color.RGBAModel.Convert(a.At(x, y)) != color.RGBAModel.Convert(b.At(x, y)) {
				return false
			}
		}
	}
	return true
}

func TestSnapshotNeedsNoTerminal(t *testing.T) {
	m := bars(t, WithRenderMode(Software))
	seq, dirty := m.seq, m.dirty
	img, mode, err := m.Snapshot(640, 360)
	if err != nil || mode != Software {
		t.Fatalf("mode=%v err=%v", mode, err)
	}
	if img.Bounds() != image.Rect(0, 0, 640, 360) {
		t.Fatalf("bounds %v: a snapshot is the size asked for, not 320×200", img.Bounds())
	}
	if drawn(img, m.background) < 1000 {
		t.Fatal("nothing was drawn")
	}
	// The program was never started, and the chart on screen is left alone.
	if m.started || m.busy || m.seq != seq || m.dirty != dirty || m.width != 80 || m.height != 24 {
		t.Fatalf("the snapshot disturbed the chart: %+v", m)
	}
	// Its size in cells is no part of the picture.
	small := bars(t, WithRenderMode(Software))
	small.SetSize(20, 8)
	if other, _, _ := small.Snapshot(640, 360); !same(img, other) {
		t.Fatal("the terminal's size changed the snapshot")
	}
	camera := m.Camera()
	camera.Beta += 60
	m.SetCamera(camera)
	if turned, _, _ := m.Snapshot(640, 360); same(img, turned) {
		t.Fatal("turning the camera did not change the snapshot")
	}
}

func TestSnapshotAsksTheRendererForItsSize(t *testing.T) {
	r := &recorder{}
	m := bars(t, WithRenderer(r))
	img, mode, err := m.Snapshot(900, 300)
	if err != nil || mode != WebGPU || img.Bounds() != image.Rect(0, 0, 900, 300) || len(r.frames) != 1 {
		t.Fatalf("mode=%v err=%v frames=%d", mode, err, len(r.frames))
	}
	f := r.frames[0]
	if f.Width != 900 || f.Height != 300 || len(f.Geometry) != 1 || f.Background != m.background || f.Revision != m.revision {
		t.Fatalf("%+v", f)
	}
	// The picture's own shape sets the camera's, not the terminal's cells.
	if want := m.camera.Matrix(3).Mul(m.plotTransform().matrix); f.Matrix != want {
		t.Fatalf("matrix %v, want %v", f.Matrix, want)
	}
}

func TestSnapshotFallsBackToSoftware(t *testing.T) {
	r := &recorder{err: errors.New("no adapter")}
	m := bars(t, WithRenderer(r))
	img, mode, err := m.Snapshot(400, 300)
	if err != nil || mode != Software || img.Bounds() != image.Rect(0, 0, 400, 300) || drawn(img, m.background) < 1000 {
		t.Fatalf("mode=%v err=%v", mode, err)
	}
	if len(r.frames) != 1 {
		t.Fatalf("the GPU was tried %d times", len(r.frames))
	}
	// What the chart shows on screen is not changed by how a snapshot fared.
	if m.RenderMode() != WebGPU || m.RequestedRenderMode() != WebGPU || m.Err() != nil {
		t.Fatalf("mode=%v requested=%v err=%v", m.RenderMode(), m.RequestedRenderMode(), m.Err())
	}
}

func TestSoftwareSnapshotDrawsEveryTriangle(t *testing.T) {
	// 150×150 is some 44,000 triangles: for the terminal, software draws
	// one in three, and the surface shows holes.
	m := New(80, 24, WithRenderMode(Software), WithBackground(color.RGBA{255, 255, 255, 255}))
	defer m.Close()
	m.SetSurface("plane", Surface{NX: 150, NY: 150, Sampler: func(x, y float64) float64 { return 0 }})
	m.SetAxes(Axes{X: Axis{Hidden: true}, Y: Axis{Hidden: true}, Z: Axis{Hidden: true}})
	m.SetColorLegendVisible(false)
	m.SetCamera(Camera{Alpha: 89, Beta: 0, Distance: 3.4, Projection: Orthographic})
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
	img, _, err := m.Snapshot(300, 300)
	if err != nil {
		t.Fatal(err)
	}
	sampled := softwareRender(m.frameSized(300, 300, 1, Software))
	holes := func(img image.Image) int {
		n := 0
		for y := 120; y < 180; y++ {
			for x := 120; x < 180; x++ {
				if color.RGBAModel.Convert(img.At(x, y)).(color.RGBA) == m.background {
					n++
				}
			}
		}
		return n
	}
	if holes(sampled) == 0 {
		t.Fatal("the terminal's frame was expected to be sampled; the test proves nothing")
	}
	if n := holes(img); n != 0 {
		t.Fatalf("the snapshot has %d holes in the middle of the surface", n)
	}
}

func TestSnapshotRefusals(t *testing.T) {
	m := bars(t, WithRenderMode(Software))
	for _, size := range [][2]int{{0, 100}, {100, 0}, {-1, -1}, {4097, 2160}, {8192, 8192}, {1 << 40, 1}} {
		if img, _, err := m.Snapshot(size[0], size[1]); err == nil || img != nil {
			t.Errorf("%d×%d was drawn", size[0], size[1])
		}
	}
	if _, _, err := m.Snapshot(4096, 2160); err != nil {
		t.Errorf("the largest frame: %v", err)
	}
	wire := bars(t, WithRenderMode(Wireframe))
	if img, _, err := wire.Snapshot(100, 100); err == nil || img != nil {
		t.Error("wireframe is text, and has no image to give")
	}
	closed := bars(t, WithRenderMode(Software))
	closed.Close()
	if img, _, err := closed.Snapshot(100, 100); err == nil || img != nil {
		t.Error("a closed chart was drawn")
	}
}
