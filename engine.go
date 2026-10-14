// ntcharts3d - Copyright (c) 2026 Neomantra Corp.
package ntcharts3d

import "nimbleterminal.dev/nt3d"

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
		Light: nt3d.Light(f.Light), Complete: f.complete, DynamicLines: f.GridLines}
	out.Geometry = make([]nt3d.Geometry, len(f.Geometry))
	for i, g := range f.Geometry {
		out.Geometry[i] = engineGeometry(g)
	}
	for _, r := range frameOverlayRects(f) {
		out.Overlays = append(out.Overlays, nt3d.OverlayRect{X: r.x, Y: r.y, Width: r.w, Height: r.h, Color: r.c})
	}
	return out
}
