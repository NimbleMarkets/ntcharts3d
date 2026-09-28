package main

import "math"

// mat4 is column-major, as WGSL expects: element (row r, col c) is m[c*4+r].
type mat4 [16]float32

type vec3 struct{ X, Y, Z float32 }

func identity() mat4 {
	var m mat4
	m[0], m[5], m[10], m[15] = 1, 1, 1, 1
	return m
}

// mul returns a*b.
func mul(a, b mat4) mat4 {
	var m mat4
	for c := range 4 {
		for r := range 4 {
			var s float32
			for k := range 4 {
				s += a[k*4+r] * b[c*4+k]
			}
			m[c*4+r] = s
		}
	}
	return m
}

// perspective builds a right-handed projection with WebGPU's clip-space
// depth range [0, 1].
func perspective(fovy, aspect, near, far float32) mat4 {
	f := float32(1 / math.Tan(float64(fovy)/2))
	var m mat4
	m[0] = f / aspect
	m[5] = f
	m[10] = far / (near - far)
	m[11] = -1
	m[14] = near * far / (near - far)
	return m
}

func sub(a, b vec3) vec3    { return vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func dot(a, b vec3) float32 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func cross(a, b vec3) vec3  { return vec3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X} }
func normalize(a vec3) vec3 {
	l := float32(math.Sqrt(float64(dot(a, a))))
	if l == 0 {
		return a
	}
	return vec3{a.X / l, a.Y / l, a.Z / l}
}

// lookAt builds a view matrix that places eye at the origin looking down -Z.
func lookAt(eye, center, up vec3) mat4 {
	f := normalize(sub(center, eye))
	s := normalize(cross(f, up))
	u := cross(s, f)
	var m mat4
	m[0], m[4], m[8] = s.X, s.Y, s.Z
	m[1], m[5], m[9] = u.X, u.Y, u.Z
	m[2], m[6], m[10] = -f.X, -f.Y, -f.Z
	m[12] = -dot(s, eye)
	m[13] = -dot(u, eye)
	m[14] = dot(f, eye)
	m[15] = 1
	return m
}

// orbitEye converts yaw/pitch (radians) and distance into a camera position
// orbiting the origin. yaw 0 looks down +Z toward the origin.
func orbitEye(yaw, pitch, dist float64) vec3 {
	cp := math.Cos(pitch)
	return vec3{
		X: float32(dist * cp * math.Sin(yaw)),
		Y: float32(dist * math.Sin(pitch)),
		Z: float32(dist * cp * math.Cos(yaw)),
	}
}
