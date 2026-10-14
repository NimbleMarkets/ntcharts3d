// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"image/color"
)

type overlayRect struct {
	x, y, w, h int
	c          color.RGBA
}

func frameOverlayRects(f Frame) []overlayRect {
	out := append(axisRects(f), emphasisRects(f)...)
	return append(out, colorLegendRects(f)...)
}
