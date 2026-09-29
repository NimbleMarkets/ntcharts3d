// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"fmt"
	"image"
)

// Snapshot draws the chart as it stands into an image of width by height
// pixels. It needs no terminal: the program need not be running, and the
// chart's size in cells plays no part. The picture's own shape sets the
// camera's aspect. Its area may not exceed 4096×2160 pixels.
//
// The chart is drawn in its requested render mode, and the mode returned is
// the one that drew it. WebGPU falls back to software if the GPU is
// unavailable or fails, as it does for the terminal, but a snapshot's
// fallback does not change how the chart is drawn on screen. A software
// snapshot draws every point, triangle, bar, line, and arrow; frames for the
// terminal sample large series to stay prompt. Wireframe draws text, and has
// no image to give.
//
// Snapshot returns when the image is drawn. Like the chart's other methods,
// it must not be called while another goroutine changes the chart.
func (m *Model) Snapshot(width, height int) (image.Image, RenderMode, error) {
	mode := m.requestedRenderMode
	switch {
	case m.closed:
		return nil, mode, fmt.Errorf("ntcharts3d: chart closed")
	case width < 1 || height < 1 || width > maxFramePixels || height > maxFramePixels/width:
		return nil, mode, fmt.Errorf("ntcharts3d: snapshot must be 1×1 to 4096×2160 pixels in area")
	case mode == Wireframe:
		return nil, mode, fmt.Errorf("ntcharts3d: wireframe has no image")
	}
	f := m.frameSized(width, height, float32(width)/float32(height), mode)
	f.complete = true
	if mode == WebGPU && m.renderer != nil {
		if img, err := m.renderer.Render(f); err == nil {
			return img, WebGPU, nil
		}
	}
	return softwareRender(f), Software, nil
}
