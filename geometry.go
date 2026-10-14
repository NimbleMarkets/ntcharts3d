// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"fmt"

	"nimbleterminal.dev/nt3d"
	nt3dmath "nimbleterminal.dev/nt3d/math"
)

type Point = nt3d.Point
type Vertex = nt3d.Vertex
type Box = nt3d.Box

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
	Bounds        nt3dmath.AABB
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
	if g.HasColorRange && (!validFloat(g.ColorMin, g.ColorMax) || g.ColorMax < g.ColorMin) {
		return fmt.Errorf("ntcharts3d: invalid color range")
	}
	return engineGeometry(g).Validate()
}
