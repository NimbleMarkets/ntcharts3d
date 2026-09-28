package main

import (
	"math"
	"testing"
)

func near(a, b float32) bool { return math.Abs(float64(a-b)) < 1e-4 }

// apply multiplies m by (x,y,z,1) and returns the clip-space result.
func apply(m mat4, x, y, z float32) (cx, cy, cz, cw float32) {
	cx = m[0]*x + m[4]*y + m[8]*z + m[12]
	cy = m[1]*x + m[5]*y + m[9]*z + m[13]
	cz = m[2]*x + m[6]*y + m[10]*z + m[14]
	cw = m[3]*x + m[7]*y + m[11]*z + m[15]
	return
}

func TestMulIdentity(t *testing.T) {
	m := perspective(1, 1.5, 0.1, 10)
	if mul(identity(), m) != m || mul(m, identity()) != m {
		t.Fatal("identity is not neutral under mul")
	}
}

func TestPerspectiveDepthRange(t *testing.T) {
	p := perspective(math.Pi/2, 1, 1, 10)
	_, _, z, w := apply(p, 0, 0, -1)
	if !near(z/w, 0) {
		t.Fatalf("near plane depth = %v, want 0", z/w)
	}
	_, _, z, w = apply(p, 0, 0, -10)
	if !near(z/w, 1) {
		t.Fatalf("far plane depth = %v, want 1", z/w)
	}
	x, _, _, w := apply(p, 1, 0, -1) // 90° fov: x=1 at z=-1 lands on the right edge
	if !near(x/w, 1) {
		t.Fatalf("right edge = %v, want 1", x/w)
	}
}

func TestLookAtMapsEyeToOriginAndCenterToNegativeZ(t *testing.T) {
	eye, center := vec3{0, 0, 5}, vec3{0, 0, 0}
	v := lookAt(eye, center, vec3{0, 1, 0})
	x, y, z, _ := apply(v, eye.X, eye.Y, eye.Z)
	if !near(x, 0) || !near(y, 0) || !near(z, 0) {
		t.Fatalf("eye maps to (%v,%v,%v), want origin", x, y, z)
	}
	_, _, z, _ = apply(v, center.X, center.Y, center.Z)
	if !near(z, -5) {
		t.Fatalf("center maps to z=%v, want -5", z)
	}
}

func TestOrbitEye(t *testing.T) {
	e := orbitEye(0, 0, 3)
	if !near(e.X, 0) || !near(e.Y, 0) || !near(e.Z, 3) {
		t.Fatalf("orbitEye(0,0,3) = %+v, want (0,0,3)", e)
	}
	e = orbitEye(math.Pi/2, 0, 3)
	if !near(e.X, 3) || !near(e.Z, 0) {
		t.Fatalf("orbitEye(pi/2,0,3) = %+v, want (3,0,0)", e)
	}
	e = orbitEye(0, math.Pi/2, 3)
	if !near(e.Y, 3) {
		t.Fatalf("orbitEye(0,pi/2,3) = %+v, want y=3", e)
	}
}
