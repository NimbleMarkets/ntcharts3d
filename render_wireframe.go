// ntcharts3d - Copyright (c) 2026 Neomantra Corp.
package ntcharts3d

import (
	"math"
	"nimbleterminal.dev/nt3d"
)

func renderWireframe(f Frame, cols, rows int) (string, error) {
	out := engineFrame(f)
	out.Overlays = nil
	layout := layoutAxes(f, cols, rows, 1, 1, nil)
	emphasis := emphasisLayout(f, cols, rows, 1, 1)
	layout.lines = append(layout.lines, emphasis.lines...)
	layout.labels = append(layout.labels, emphasis.labels...)
	// Axis strokes are screen-space decorations, independent of scene depth.
	for _, segment := range layout.lines {
		segment, ok := clipAxisSegment(segment, cols, rows)
		if !ok {
			continue
		}
		n := max(1, int(math.Ceil(float64(max(abs(segment.b.x-segment.a.x), abs(segment.b.y-segment.a.y))))))
		for i := 0; i <= n; i++ {
			t := float32(i) / float32(n)
			x, y := int(math.Round(float64(segment.a.x+(segment.b.x-segment.a.x)*t))), int(math.Round(float64(segment.a.y+(segment.b.y-segment.a.y)*t)))
			out.CellText = append(out.CellText, nt3d.CellText{X: x, Y: y, Text: "·", Color: segment.color})
		}
	}
	for _, label := range layout.labels {
		out.CellText = append(out.CellText, nt3d.CellText{X: label.rect.Min.X, Y: label.rect.Min.Y, Text: label.text, Color: label.color})
	}
	return nt3d.RenderWireframe(out, cols, rows)
}
