//go:build webgpu

// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"image"
	"testing"
)

func TestGPUSnapshot(t *testing.T) {
	m := New(80, 24)
	defer m.Close()
	m.SetBars("bars", Bars{NX: 2, NY: 2, Values: []float32{.2, .8, .4, .6}})
	// Larger than any frame drawn for glyphs, and not the terminal's shape.
	img, mode, err := m.Snapshot(1600, 900)
	if err != nil {
		t.Fatal(err)
	}
	if mode != WebGPU {
		t.Fatalf("drawn by %v: no GPU was found", mode)
	}
	if img.Bounds() != image.Rect(0, 0, 1600, 900) {
		t.Fatalf("bounds %v", img.Bounds())
	}
	nonBackground(t, img, m.background)
	// Another view of the same data is drawn without uploading it again.
	r := m.renderer.(*gpuRenderer)
	uploads := r.uploads
	camera := m.Camera()
	camera.Beta += 45
	m.SetCamera(camera)
	turned, _, err := m.Snapshot(1600, 900)
	if err != nil {
		t.Fatal(err)
	}
	if same(img, turned) {
		t.Fatal("turning the camera did not change the snapshot")
	}
	if r.uploads != uploads {
		t.Fatal("a second view uploaded the geometry again")
	}
	// A chart drawn in software on screen still has its GPU for the asking.
	if _, mode, err := m.Snapshot(64, 64); err != nil || mode != WebGPU {
		t.Fatalf("a small snapshot: %v %v", mode, err)
	}
}

type mesh struct{ geometry Geometry }

func (m mesh) Name() string                       { return "mesh" }
func (m mesh) Geometry(Palette) (Geometry, error) { return m.geometry, nil }

func TestGPUDrawsTheLargestMesh(t *testing.T) {
	m := New(80, 24)
	defer m.Close()
	m.SetSeries(mesh{triangles(MaxMeshTriangles)})
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
	img, mode, err := m.Snapshot(1024, 1024)
	if err != nil || mode != WebGPU {
		t.Fatalf("mode=%v err=%v", mode, err)
	}
	nonBackground(t, img, m.background)
}
