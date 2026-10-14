// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"image"
	"image/color"

	nt3dmath "nimbleterminal.dev/nt3d/math"
)

// RenderMode selects WebGPU, software, or wireframe rendering.
// Kitty/glyph output is a separate setting.
type RenderMode uint8

const (
	WebGPU RenderMode = iota
	Software
	Wireframe
)

func (l RenderMode) String() string {
	switch l {
	case WebGPU:
		return "WebGPU"
	case Software:
		return "software"
	default:
		return "wireframe"
	}
}

type Light struct {
	Direction nt3dmath.Vec3
	Ambient   float32
}

// Frame is a read-only snapshot. Mesh data changes only when Revision changes; materials may also change
// when TextureRevision changes.
type Frame struct {
	Width, Height   int
	Matrix          nt3dmath.Mat4
	Revision        uint64
	TextureRevision uint64
	Bounds          nt3dmath.AABB // Plot bounds after fixed axis ranges.
	Axes            [3]AxisFrame  // X, Y, Z; separate from cached series geometry.
	Geometry        []Geometry    // Clipped geometry; mesh normals account for plot proportions.
	GridLines       []Vertex      // Dynamic pairs of back-plane grid endpoints.
	Emphasis        *EmphasisFrame
	ColorLegends    []ColorLegend
	Background      color.RGBA
	Light           Light

	// complete asks the software renderer to draw everything. For the
	// terminal it samples large series to keep its frames prompt.
	complete bool
}

// Renderer draws a Frame into a Go image. Implementations must serialize Render
// and Close and reuse mesh data while Frame.Revision is unchanged. Texture bindings
// must also be refreshed when Frame.TextureRevision changes.
type Renderer interface {
	Render(Frame) (image.Image, error)
	Close() error
}
