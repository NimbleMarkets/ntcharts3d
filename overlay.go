// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"image"
	"image/color"
	"image/draw"
)

type overlayRect struct {
	x, y, w, h int
	c          color.RGBA
}

func drawOverlays(img *image.RGBA, f Frame) {
	for _, r := range frameOverlayRects(f) {
		draw.Draw(img, image.Rect(r.x, r.y, r.x+r.w, r.y+r.h), image.NewUniform(r.c), image.Point{}, draw.Src)
	}
}

func frameOverlayRects(f Frame) []overlayRect {
	out := append(axisRects(f), emphasisRects(f)...)
	return append(out, colorLegendRects(f)...)
}
