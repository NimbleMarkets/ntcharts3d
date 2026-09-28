// ntcharts3d - Copyright (c) 2026 Neomantra Corp.
// Adapted from ntcharts heatpicture. MIT.

package ntcharts3d

import (
	"image/color"
)

// Palette is a sequence of color stops, independent of a numeric color domain.
type Palette []color.RGBA

// At interpolates the palette at t, clamped to [0,1]. An empty palette returns
// transparent black.
func (s Palette) At(t float64) color.RGBA { return colorAt(s, t) }
func defaultPalette() Palette {
	return Palette{{45, 70, 150, 255}, {35, 185, 165, 255}, {245, 205, 70, 255}}
}

func valuesRange(v []float32) (lo, hi float32) {
	if len(v) == 0 {
		return 0, 1
	}
	lo, hi = v[0], v[0]
	for _, x := range v {
		lo = min(lo, x)
		hi = max(hi, x)
	}
	return
}

func mapped(s Palette, v, lo, hi float32) color.RGBA {
	if hi == lo {
		return s.At(.5)
	}
	return s.At((float64(v) - float64(lo)) / (float64(hi) - float64(lo)))
}

// colorAt interpolates adjacent RGBA stops at t, clamped to [0,1].
// SetPalette converts colors once so rendering can use RGBA values directly.
func colorAt(scale []color.RGBA, t float64) color.RGBA {
	if len(scale) == 0 {
		return color.RGBA{0, 0, 0, 0}
	}
	if len(scale) == 1 {
		return scale[0]
	}
	if t <= 0 {
		return scale[0]
	}
	if t >= 1 {
		return scale[len(scale)-1]
	}

	pos := t * float64(len(scale)-1)
	lo := int(pos)
	if lo >= len(scale)-1 {
		return scale[len(scale)-1]
	}
	frac := pos - float64(lo)

	return lerpRGBA(scale[lo], scale[lo+1], frac)
}

// lerpRGBA linearly interpolates between two RGBA colors in straight RGBA.
func lerpRGBA(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
		A: uint8(float64(a.A)*(1-t) + float64(b.A)*t),
	}
}
