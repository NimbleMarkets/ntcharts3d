// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"fmt"
	"image/color"

	"github.com/NimbleMarkets/ntcharts3d/math3d"
)

type Point struct {
	Position math3d.Vec3
	Radius   float32
	Color    color.RGBA
	Datum    int
}

type Vertex struct {
	Position, Normal math3d.Vec3
	Color            color.RGBA
}

type Box struct {
	Min, Size math3d.Vec3
	Color     color.RGBA
	Datum     int
}

// Geometry contains points, indexed triangles, boxes, line pairs, and arrow instances.
// Custom Series must not mutate geometry after returning it to the model.
type Geometry struct {
	UV            []UV      // One per vertex when textured.
	Material      *Material // Applies to indexed triangles only.
	Points        []Point
	Vertices      []Vertex
	Indices       []uint32
	Boxes         []Box
	Lines         []Vertex
	LineWidth     float32 // Render pixels for Lines; zero defaults to 1.
	Arrows        []Arrow
	Labels        []string
	Bounds        math3d.AABB
	ColorMin      float32
	ColorMax      float32
	HasColorRange bool
}

// MaxMeshIndices is the most indices a series' mesh may have, three to a
// triangle. The GPU renderer keeps a 48-byte record for each index, vertices
// shared or not, in a storage buffer; 128 MiB is the largest binding that
// WebGPU promises, and so the largest mesh that can be drawn wherever WebGPU
// can. MaxMeshTriangles is the same limit, counted in faces.
const (
	MaxMeshTriangles = (128 << 20) / 48 / 3
	MaxMeshIndices   = 3 * MaxMeshTriangles
)

func validateGeometry(g Geometry) error {
	if !validWidth(g.LineWidth) {
		return fmt.Errorf("ntcharts3d: invalid line width")
	}
	if len(g.Arrows) > MaxVectors {
		return fmt.Errorf("ntcharts3d: arrow count exceeds cap")
	}
	for _, a := range g.Arrows {
		if !finiteVec(a.Start) || !finiteVec(a.End) || !validWidth(a.Width) || !validFloat(a.HeadSize) || a.HeadSize < 0 || a.HeadSize > 64 {
			return fmt.Errorf("ntcharts3d: invalid arrow")
		}
	}

	if len(g.UV) != 0 && len(g.UV) != len(g.Vertices) {
		return fmt.Errorf("ntcharts3d: UV length mismatch")
	}
	if textured(g) && (!g.Material.Texture.valid() || len(g.UV) != len(g.Vertices)) {
		return fmt.Errorf("ntcharts3d: invalid texture or missing UVs")
	}
	for _, uv := range g.UV {
		if !validFloat(uv.U, uv.V) {
			return fmt.Errorf("ntcharts3d: non-finite UV")
		}
	}

	if g.HasColorRange && (!validFloat(g.ColorMin, g.ColorMax) || g.ColorMax < g.ColorMin) {
		return fmt.Errorf("ntcharts3d: invalid color range")
	}
	for _, p := range g.Points {
		if !validFloat(p.Position.X, p.Position.Y, p.Position.Z, p.Radius) || p.Radius < 1 || p.Radius > 32 {
			return fmt.Errorf("ntcharts3d: invalid point geometry")
		}
	}
	for _, v := range g.Vertices {
		if !validFloat(v.Position.X, v.Position.Y, v.Position.Z, v.Normal.X, v.Normal.Y, v.Normal.Z) {
			return fmt.Errorf("ntcharts3d: non-finite vertex")
		}
	}
	for _, v := range g.Lines {
		if !validFloat(v.Position.X, v.Position.Y, v.Position.Z) {
			return fmt.Errorf("ntcharts3d: non-finite line")
		}
	}
	for _, b := range g.Boxes {
		if !validFloat(b.Min.X, b.Min.Y, b.Min.Z, b.Size.X, b.Size.Y, b.Size.Z) || min(b.Size.X, b.Size.Y, b.Size.Z) < 0 {
			return fmt.Errorf("ntcharts3d: invalid box")
		}
	}
	if g.Bounds.IsValid() && (!validFloat(g.Bounds.Min.X, g.Bounds.Min.Y, g.Bounds.Min.Z, g.Bounds.Max.X, g.Bounds.Max.Y, g.Bounds.Max.Z) || g.Bounds.Max.X < g.Bounds.Min.X || g.Bounds.Max.Y < g.Bounds.Min.Y || g.Bounds.Max.Z < g.Bounds.Min.Z) {
		return fmt.Errorf("ntcharts3d: invalid bounds")
	}

	// A mesh may give each face vertices of its own, so it may have as many
	// vertices as indices.
	if len(g.Points) > MaxPoints || len(g.Vertices) > MaxMeshIndices || len(g.Indices) > MaxMeshIndices || len(g.Boxes) > 512*512 || len(g.Lines) > 2*512*512 {
		return fmt.Errorf("ntcharts3d: geometry exceeds cap")
	}
	if len(g.Indices)%3 != 0 || len(g.Lines)%2 != 0 {
		return fmt.Errorf("ntcharts3d: incomplete triangle or line")
	}
	for _, i := range g.Indices {
		if int(i) >= len(g.Vertices) {
			return fmt.Errorf("ntcharts3d: invalid vertex index")
		}
	}
	return nil
}

// boxVertices uses outward counterclockwise faces, matching the GPU box shader.
func boxVertices(b Box) []Vertex {
	corners := []math3d.Vec3{{X: 0, Y: 0, Z: 0}, {X: 1, Y: 0, Z: 0}, {X: 1, Y: 1, Z: 0}, {X: 0, Y: 1, Z: 0}, {X: 0, Y: 0, Z: 1}, {X: 1, Y: 0, Z: 1}, {X: 1, Y: 1, Z: 1}, {X: 0, Y: 1, Z: 1}}
	faces := [][4]int{{0, 3, 2, 1}, {4, 5, 6, 7}, {0, 1, 5, 4}, {3, 7, 6, 2}, {0, 4, 7, 3}, {1, 2, 6, 5}}
	out := make([]Vertex, 0, 36)
	for _, f := range faces {
		n := corners[f[1]].Sub(corners[f[0]]).Cross(corners[f[2]].Sub(corners[f[0]])).Normalize()
		for _, j := range []int{0, 1, 2, 0, 2, 3} {
			p := corners[f[j]]
			p = b.Min.Add(math3d.Vec3{X: p.X * b.Size.X, Y: p.Y * b.Size.Y, Z: p.Z * b.Size.Z})
			out = append(out, Vertex{p, n, b.Color})
		}
	}
	return out
}
