package main

import (
	"bytes"
	"context"
	"os"
	"testing"
)

// Requires a real adapter. Run without -race (Metal FFI checkptr).
func TestSceneRendersOpaqueAnimatedFrames(t *testing.T) {
	if os.Getenv("NTCHARTS3D_TRANSPORT_GPU_TEST") != "1" {
		t.Skip("set NTCHARTS3D_TRANSPORT_GPU_TEST=1 for real GPU tests")
	}
	s, err := newScene(context.Background(), fibonacciSphere(2000), 96, 64)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.Name() == "" {
		t.Fatal("empty adapter name")
	}
	t.Log("adapter:", s.Name())
	req := frameRequest{width: 96, height: 64, yaw: 0, pitch: 0.3, dist: 3, pointSize: 2}
	first, st, err := s.Render(req)
	if err != nil {
		t.Fatal(err)
	}
	if st.Total <= 0 {
		t.Fatalf("stats = %+v", st)
	}
	req.yaw = 1.0
	second, _, err := s.Render(req)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first.Pix, second.Pix) {
		t.Fatal("rotating the camera did not change the frame")
	}
	varied := false
	for i := 0; i < len(first.Pix); i += 4 {
		if first.Pix[i+3] != 255 {
			t.Fatalf("non-opaque pixel at %d", i/4)
		}
		if !bytes.Equal(first.Pix[i:i+3], first.Pix[:3]) {
			varied = true
		}
	}
	if !varied {
		t.Fatal("frame is a single color; no points were drawn")
	}
	// Resizing through the request must produce the new size.
	req.width, req.height = 40, 30
	third, _, err := s.Render(req)
	if err != nil {
		t.Fatal(err)
	}
	if b := third.Bounds(); b.Dx() != 40 || b.Dy() != 30 {
		t.Fatalf("bounds after resize = %v", b)
	}
}
