// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"image"
	"image/color"
	"testing"
)

// Hide concrete types to exercise the generic image.Image path.
type textureGenericImage struct{ image.Image }

func TestTextureTypedOpacitySubimages(t *testing.T) {
	for _, name := range []string{"RGBA", "NRGBA"} {
		t.Run(name, func(t *testing.T) {
			var src interface {
				image.Image
				Set(int, int, color.Color)
				SubImage(image.Rectangle) image.Image
			}
			if name == "NRGBA" {
				src = image.NewNRGBA(image.Rect(-5, 7, 19, 29))
			} else {
				src = image.NewRGBA(image.Rect(-5, 7, 19, 29))
			}
			bounds := image.Rect(-2, 9, 13, 21)
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					src.Set(x, y, color.RGBA{R: uint8(x + 5), G: uint8(y), B: 100, A: 255})
				}
			}
			// Pixels outside the crop remain transparent; rows have extra stride.
			sub := src.SubImage(bounds)
			got, err := NewTexture(sub)
			if err != nil {
				t.Fatalf("opaque subimage: %v", err)
			}
			want, err := NewTexture(textureGenericImage{sub})
			if err != nil {
				t.Fatalf("generic subimage: %v", err)
			}
			if got.Bounds() != image.Rect(0, 0, bounds.Dx(), bounds.Dy()) {
				t.Fatalf("snapshot bounds: %v", got.Bounds())
			}
			src.Set(bounds.Min.X, bounds.Min.Y, color.RGBA{A: 254})
			for y := 0; y < bounds.Dy(); y++ {
				for x := 0; x < bounds.Dx(); x++ {
					if got.At(x, y) != want.At(x, y) {
						t.Fatalf("snapshot pixel (%d,%d): got %v, want %v", x, y, got.At(x, y), want.At(x, y))
					}
				}
			}
			for _, input := range []image.Image{sub, textureGenericImage{sub}} {
				if tex, err := NewTexture(input); tex != nil || err == nil || err.Error() != "ntcharts3d: texture must be opaque" {
					t.Fatalf("translucent subimage: texture=%v, error=%v", tex, err)
				}
			}
		})
	}
}

func BenchmarkNewTexture(b *testing.B) {
	for _, name := range []string{"RGBA", "NRGBA"} {
		b.Run(name, func(b *testing.B) {
			var src image.Image
			if name == "NRGBA" {
				img := image.NewNRGBA(image.Rect(0, 0, 512, 512))
				for i := 3; i < len(img.Pix); i += 4 {
					img.Pix[i] = 255
				}
				src = img
			} else {
				img := image.NewRGBA(image.Rect(0, 0, 512, 512))
				for i := 3; i < len(img.Pix); i += 4 {
					img.Pix[i] = 255
				}
				src = img
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := NewTexture(src); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
