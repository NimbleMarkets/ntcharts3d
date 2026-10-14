// ntcharts3d - Copyright (c) 2026 Neomantra Corp.
package ntcharts3d

import (
	"image"
	"nimbleterminal.dev/nt3d"
)

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func renderSoftware(f Frame) (image.Image, error) {
	return nt3d.NewSoftwareRenderer().Render(engineFrame(f))
}
