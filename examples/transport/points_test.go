package main

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestFibonacciSphereIsUnitAndColored(t *testing.T) {
	pts := fibonacciSphere(500)
	if len(pts) != 500 {
		t.Fatalf("len = %d", len(pts))
	}
	for i, p := range pts {
		r := math.Sqrt(float64(p.X*p.X + p.Y*p.Y + p.Z*p.Z))
		if math.Abs(r-1) > 1e-5 {
			t.Fatalf("point %d radius %v", i, r)
		}
		for _, c := range []float32{p.R, p.G, p.B} {
			if c < 0 || c > 1 {
				t.Fatalf("point %d color out of range: %+v", i, p)
			}
		}
	}
	if pts[0].Y <= pts[len(pts)-1].Y {
		t.Fatal("points should sweep from the north pole to the south pole")
	}
	if pts[0].R == pts[len(pts)-1].R && pts[0].G == pts[len(pts)-1].G && pts[0].B == pts[len(pts)-1].B {
		t.Fatal("poles have the same color; latitude gradient is missing")
	}
}

func TestPackPointsLayout(t *testing.T) {
	b := packPoints([]point{{X: 1, Y: 2, Z: 3, R: 0.5, G: 0.25, B: 0.125}})
	if len(b) != 32 {
		t.Fatalf("len = %d, want 32", len(b))
	}
	f := func(i int) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:])) }
	want := []float32{1, 2, 3, 1, 0.5, 0.25, 0.125, 1}
	for i, w := range want {
		if f(i) != w {
			t.Fatalf("float %d = %v, want %v", i, f(i), w)
		}
	}
}

func TestUniformBytesLayout(t *testing.T) {
	m := identity()
	b := uniformBytes(m, 640, 480, 2.5)
	if len(b) != 80 {
		t.Fatalf("len = %d, want 80", len(b))
	}
	f := func(i int) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:])) }
	for i := 0; i < 16; i++ {
		if f(i) != m[i] {
			t.Fatalf("matrix element %d = %v, want %v", i, f(i), m[i])
		}
	}
	if f(16) != 640 || f(17) != 480 || f(18) != 2.5 || f(19) != 0 {
		t.Fatalf("tail = %v %v %v %v", f(16), f(17), f(18), f(19))
	}
}
