// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"sync"
	"unicode"
)

const textWidth, textHeight = 7, 14

var labelFont = func() *opentype.Font {
	f, err := opentype.Parse(gomono.TTF)
	if err != nil {
		panic(err)
	} // The bundled font is static.
	return f
}()
var labelFace = func() font.Face {
	f, err := opentype.NewFace(labelFont, &opentype.FaceOptions{Size: 11, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		panic(err)
	}
	return f
}()
var labelMu sync.Mutex // font.Face reuses scratch buffers.

// displayText bounds label length and replaces unsupported or control characters.
// Go Mono includes Latin, Greek, Cyrillic, and common scientific symbols.
func displayText(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	out := make([]rune, 0, min(limit, 64))
	for _, r := range text {
		if len(out) == limit {
			out[len(out)-1] = '…'
			break
		}
		index, err := labelFont.GlyphIndex(nil, r)
		if unicode.IsControl(r) || unicode.Is(unicode.Mn, r) || err != nil || index == 0 {
			r = '?'
		}
		out = append(out, r)
	}
	return string(out)
}

func bitmapText(text string, x, y int, c color.RGBA, scale int) []overlayRect {
	if text == "" {
		return nil
	}
	labelMu.Lock()
	defer labelMu.Unlock()
	mask := image.NewAlpha(image.Rect(0, 0, font.MeasureString(labelFace, text).Ceil(), textHeight))
	d := font.Drawer{Dst: mask, Src: image.NewUniform(color.White), Face: labelFace, Dot: fixed.P(0, 11)}
	d.DrawString(text)
	var rects []overlayRect
	for row := 0; row < mask.Bounds().Dy(); row++ {
		for col := 0; col < mask.Bounds().Dx(); {
			if mask.AlphaAt(col, row).A < 96 {
				col++
				continue
			}
			start := col
			for col < mask.Bounds().Dx() && mask.AlphaAt(col, row).A >= 96 {
				col++
			}
			rects = append(rects, overlayRect{x + start*scale, y + row*scale, (col - start) * scale, scale, c})
		}
	}
	return rects
}
