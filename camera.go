// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"math"

	"nimbleterminal.dev/nt3d"
	nt3dmath "nimbleterminal.dev/nt3d/math"
)

type Projection = nt3d.Projection

const (
	Orthographic = nt3d.Orthographic
	Perspective  = nt3d.Perspective
)

// Camera uses degrees for Alpha (elevation) and Beta (azimuth). Distance and
// Target use normalized units: geometry is centered at the origin and its
// longest extent spans two units.
type Camera struct {
	Alpha, Beta, Distance float64
	Target                nt3dmath.Vec3
	Projection            Projection
	AutoRotate            bool
	ResumeAfter           float64 // idle seconds before rotation resumes after input
}

// DefaultCamera returns a Camera initialized with some typical default values.
func DefaultCamera() Camera {
	return Camera{Alpha: 25, Beta: 40, Distance: 3.4, ResumeAfter: 2}
}

func (c Camera) valid() Camera {
	if !nt3dmath.IsFinite(c.Alpha) {
		c.Alpha = 25
	}
	if !nt3dmath.IsFinite(c.Beta) {
		c.Beta = 40
	}
	c.Alpha = math.Max(-89, math.Min(89, c.Alpha))
	c.Beta = math.Mod(c.Beta, 360)
	if !nt3dmath.IsFinite(c.Distance) || c.Distance <= 0 {
		c.Distance = 3.4
	}
	c.Distance = math.Max(.15, math.Min(100, c.Distance))
	if !nt3dmath.IsFinite(float64(c.Target.X)) || !nt3dmath.IsFinite(float64(c.Target.Y)) || !nt3dmath.IsFinite(float64(c.Target.Z)) {
		c.Target = nt3dmath.Vec3{}
	}
	if c.Projection != Perspective {
		c.Projection = Orthographic
	}
	if !nt3dmath.IsFinite(c.ResumeAfter) || c.ResumeAfter <= 0 {
		c.ResumeAfter = 2
	}
	return c
}

func (c Camera) basis() (eye, right, up, forward nt3dmath.Vec3) {
	c = c.valid()
	a, b := c.Alpha*math.Pi/180, c.Beta*math.Pi/180
	eye = c.Target.Add(nt3dmath.Vec3{X: float32(c.Distance * math.Cos(a) * math.Cos(b)), Y: float32(c.Distance * math.Cos(a) * math.Sin(b)), Z: float32(c.Distance * math.Sin(a))})
	forward = c.Target.Sub(eye).Normalize()
	right = forward.Cross(nt3dmath.Vec3{Z: 1}).Normalize()
	up = right.Cross(forward)
	return
}

func (c Camera) Matrix(aspect float32) nt3dmath.Mat4 { return nt3d.Camera(c.valid()).Matrix(aspect) }

// RayFromCell returns a ray through a cell center, accounting for cell dimensions.
func (c Camera) RayFromCell(x, y, cols, rows, cellW, cellH int) nt3dmath.Ray {
	return c.ray(float32(x)+.5, float32(y)+.5, cols, rows, cellW, cellH)
}

func (c Camera) ray(x, y float32, cols, rows, cw, ch int) nt3dmath.Ray {
	return nt3d.Camera(c.valid()).RayFromPixel(x*float32(cw), y*float32(ch), cols*cw, rows*ch)
}
