// ntcharts3d - Copyright (c) 2026 Neomantra Corp.
package ntcharts3d

import (
	"image/color"
	"nimbleterminal.dev/nt3d"
	nt3dmath "nimbleterminal.dev/nt3d/math"
)

// engineGeometry borrows immutable chart buffers. Labels and color domains stay
// in the chart layer; they are not rendering concerns.
func engineGeometry(g Geometry) nt3d.Geometry {
	out := nt3d.Geometry{Points: g.Points, Vertices: g.Vertices, Indices: g.Indices,
		Boxes: g.Boxes, Lines: g.Lines, LineWidth: g.LineWidth, Arrows: g.Arrows,
		UV: g.UV, Bounds: g.Bounds}
	if g.Material != nil {
		out.Material = &nt3d.Material{Texture: g.Material.Texture, Unlit: g.Material.Unlit}
	}
	return out
}

// engineFrame preserves the chart Frame contract: positions are data-space,
// Matrix includes plot normalization, and normals already reflect plot aspect.
// Empty Instances deliberately selects identity transforms in nt3d.
func engineFrame(f Frame) nt3d.Frame {
	out := nt3d.Frame{Width: f.Width, Height: f.Height, Matrix: f.Matrix,
		Revision: f.Revision, TextureRevision: f.TextureRevision, Background: f.Background,
		Lighting: engineLighting(f.Light), Complete: f.complete, DynamicLines: f.GridLines}
	out.Anisotropy = textureAnisotropy(f.Anisotropy)
	out.Geometry = make([]nt3d.Geometry, len(f.Geometry))
	for i, g := range f.Geometry {
		out.Geometry[i] = engineGeometry(g)
	}
	for _, r := range frameOverlayRects(f) {
		out.Overlays = append(out.Overlays, nt3d.OverlayRect{X: r.x, Y: r.y, Width: r.w, Height: r.h, Color: r.c})
	}
	return out
}

// engineLighting preserves the chart's ambient + (1-ambient)*diffuse model.
// An intensity of zero means one in nt3d, so omit lights at the endpoints.
func engineLighting(l Light) nt3d.Lighting {
	var out nt3d.Lighting
	white := color.RGBA{255, 255, 255, 255}
	ambient := max(float32(0), min(float32(1), l.Ambient))
	if ambient > 0 {
		out.Ambient = nt3d.Ambient{Sky: white, Ground: white, Intensity: ambient}
	}
	// A zero direction contributed no diffuse light in the chart API.
	if ambient < 1 && l.Direction != (nt3dmath.Vec3{}) {
		out.Lights = []nt3d.Light{{Kind: nt3d.DirectionalLight, Direction: l.Direction, Color: white, Intensity: 1 - ambient}}
	}
	return out
}
