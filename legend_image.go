package ntcharts3d

import (
	"image"
	"image/color"
	"strconv"
)

func colorLegendScale(f Frame) int {
	if f.Width >= 480 && f.Height >= 300 {
		return 2
	}
	return 1
}

func colorLegendRects(f Frame) []overlayRect {
	var out []overlayRect
	scale := colorLegendScale(f)
	for i, panel := range colorLegendPanels(f) {
		legend := f.ColorLegends[i]
		x, y, panelW, panelH := panel.Min.X, panel.Min.Y, panel.Dx(), panel.Dy()
		bg := color.RGBA{235, 238, 242, 255}
		ink := color.RGBA{35, 40, 48, 255}
		out = append(out, overlayRect{x, y, panelW, panelH, ink})
		out = append(out, overlayRect{x + scale, y + scale, panelW - 2*scale, panelH - 2*scale, bg})
		label := displayText(legend.Label, 16)
		out = append(out, bitmapText(label, x+6*scale, y+6*scale, ink, scale)...)
		out = append(out, bitmapText(numberLabel(legend.Max), x+6*scale, y+21*scale, ink, scale)...)
		out = append(out, bitmapText(numberLabel(legend.Min), x+6*scale, y+panelH-(textHeight+2)*scale, ink, scale)...)

		barX, barY, barW, barH := x+panelW-18*scale, y+21*scale, 12*scale, panelH-35*scale
		for row := range barH {
			t := 1 - float64(row)/float64(barH-1)
			c := legend.Palette.At(t)
			out = append(out, overlayRect{barX, barY + row, barW, 1, c})
		}
	}
	return out
}

func numberLabel(v float32) string { return strconv.FormatFloat(float64(v), 'g', 3, 32) }

func colorLegendPanels(f Frame) []image.Rectangle {
	if f.Width < 176 || f.Height < 112 {
		return nil
	}
	scale := colorLegendScale(f)
	panelW, panelH := 120*scale, 90*scale
	var panels []image.Rectangle
	for i := range f.ColorLegends {
		x, y := max(4, f.Width-panelW-12), 8+i*(panelH+6)
		if y+panelH > f.Height-4 {
			break
		}
		panels = append(panels, image.Rect(x, y, x+panelW, y+panelH))
	}
	return panels
}
