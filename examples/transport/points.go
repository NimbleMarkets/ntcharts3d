package main

import (
	"encoding/binary"
	"math"
)

type point struct{ X, Y, Z, R, G, B float32 }

// fibonacciSphere spreads n points evenly over the unit sphere, sweeping
// from the north pole (y=1) to the south pole, and colors them by latitude
// with a three-stop gradient (blue → green → yellow).
func fibonacciSphere(n int) []point {
	pts := make([]point, n)
	golden := math.Pi * (3 - math.Sqrt(5))
	for i := range pts {
		y := 1 - 2*(float64(i)+0.5)/float64(n)
		r := math.Sqrt(1 - y*y)
		theta := golden * float64(i)
		x, z := r*math.Cos(theta), r*math.Sin(theta)
		cr, cg, cb := latitudeColor((1 - y) / 2)
		pts[i] = point{float32(x), float32(y), float32(z), cr, cg, cb}
	}
	return pts
}

// latitudeColor maps t in [0,1] through blue (0.20,0.35,0.95) → green
// (0.15,0.80,0.45) → yellow (0.98,0.85,0.20).
func latitudeColor(t float64) (r, g, b float32) {
	stops := [3][3]float64{{0.20, 0.35, 0.95}, {0.15, 0.80, 0.45}, {0.98, 0.85, 0.20}}
	if t <= 0.5 {
		return lerp3(stops[0], stops[1], t*2)
	}
	return lerp3(stops[1], stops[2], (t-0.5)*2)
}

func lerp3(a, b [3]float64, t float64) (float32, float32, float32) {
	return float32(a[0] + (b[0]-a[0])*t), float32(a[1] + (b[1]-a[1])*t), float32(a[2] + (b[2]-a[2])*t)
}

func putF32(b []byte, i int, v float32) { binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(v)) }

// packPoints lays points out as the WGSL struct Point { pos: vec4<f32>,
// color: vec4<f32> } — 32 bytes each, both w components set to 1.
func packPoints(pts []point) []byte {
	b := make([]byte, 32*len(pts))
	for i, p := range pts {
		o := i * 8
		putF32(b, o+0, p.X)
		putF32(b, o+1, p.Y)
		putF32(b, o+2, p.Z)
		putF32(b, o+3, 1)
		putF32(b, o+4, p.R)
		putF32(b, o+5, p.G)
		putF32(b, o+6, p.B)
		putF32(b, o+7, 1)
	}
	return b
}

// uniformBytes lays out the WGSL Uniforms struct: view_proj mat4x4<f32>,
// viewport vec2<f32>, point_size f32, _pad f32 — 80 bytes.
func uniformBytes(viewProj mat4, w, h int, pointSize float32) []byte {
	b := make([]byte, 80)
	for i, v := range viewProj {
		putF32(b, i, v)
	}
	putF32(b, 16, float32(w))
	putF32(b, 17, float32(h))
	putF32(b, 18, pointSize)
	putF32(b, 19, 0)
	return b
}
