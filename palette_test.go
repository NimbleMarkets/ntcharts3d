// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"image/color"
	"math"
	"testing"
)

func TestPaletteAt(t *testing.T) {
	a := color.RGBA{10, 20, 30, 40}
	b := color.RGBA{110, 120, 130, 140}
	for _, tc := range []struct {
		name string
		at   float64
		want color.RGBA
	}{
		{"NaN", math.NaN(), a},
		{"negative NaN", math.Float64frombits(0xfff8000000000001), a},
		{"negative infinity", math.Inf(-1), a},
		{"below zero", -1, a},
		{"zero", 0, a},
		{"midpoint", .5, color.RGBA{60, 70, 80, 90}},
		{"one", 1, b},
		{"above one", 2, b},
		{"positive infinity", math.Inf(1), b},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := (Palette{a, b}).At(tc.at); got != tc.want {
				t.Errorf("At(%v) = %v, want %v", tc.at, got, tc.want)
			}
			if got := (Palette{a}).At(tc.at); got != a {
				t.Errorf("single stop At(%v) = %v, want %v", tc.at, got, a)
			}
			if got := (Palette{}).At(tc.at); got != (color.RGBA{}) {
				t.Errorf("empty palette At(%v) = %v", tc.at, got)
			}
		})
	}
}
