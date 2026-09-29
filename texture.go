// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"reflect"
	"slices"

	tea "charm.land/bubbletea/v2"
)

// UV is a normalized image coordinate; (0,0) is the top-left corner.
type UV struct{ U, V float32 }

// Material controls triangle appearance. Texture replaces vertex colors.
// Unlit preserves colors without applying the scene light.
type Material struct {
	Texture *Texture
	Unlit   bool
}

// Texture owns an immutable snapshot of opaque image pixels. The zero value is invalid.
// GPU resources are owned by each renderer, not by Texture.
type Texture struct {
	width, height int
	pixels        []byte
}

// NewTexture snapshots src, including images with nonzero bounds. Images must be
// opaque and at most 8192 pixels per side. Channels are treated as sRGB-encoded
// values, with filtering and lighting performed directly in that encoded space.
func NewTexture(src image.Image) (*Texture, error) {
	if src == nil || (reflect.ValueOf(src).Kind() == reflect.Ptr && reflect.ValueOf(src).IsNil()) {
		return nil, fmt.Errorf("ntcharts3d: nil texture image")
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 || w > 8192 || h > 8192 {
		return nil, fmt.Errorf("ntcharts3d: texture dimensions must be 1..8192")
	}
	t := &Texture{width: w, height: h, pixels: make([]byte, w*h*4)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			pixel := src.At(b.Min.X+x, b.Min.Y+y)
			_, _, _, alpha := pixel.RGBA()
			if alpha != 65535 {
				return nil, fmt.Errorf("ntcharts3d: texture must be opaque")
			}
			c := color.NRGBAModel.Convert(pixel).(color.NRGBA)
			i := (y*w + x) * 4
			copy(t.pixels[i:i+4], []byte{c.R, c.G, c.B, 255})
		}
	}
	return t, nil
}
func (t *Texture) valid() bool {
	return t != nil && t.width > 0 && t.height > 0 && len(t.pixels) == t.width*t.height*4
}
func cloneMaterial(m *Material) *Material {
	if m == nil {
		return nil
	}
	c := *m
	return &c
}
func textured(g Geometry) bool { return g.Material != nil && g.Material.Texture != nil }

// sample uses the same texel-center convention as a bilinear clamp-to-edge sampler.
func (t *Texture) sample(uv UV) color.RGBA {
	x := float64(max(0, min(1, uv.U)))*float64(t.width) - .5
	y := float64(max(0, min(1, uv.V)))*float64(t.height) - .5
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(x0), y-float64(y0)
	pixel := func(x, y, c int) float64 {
		return float64(t.pixels[(max(0, min(t.height-1, y))*t.width+max(0, min(t.width-1, x)))*4+c])
	}
	var c [3]uint8
	for i := range c {
		c[i] = uint8((pixel(x0, y0, i)*(1-fx)+pixel(x0+1, y0, i)*fx)*(1-fy) + (pixel(x0, y0+1, i)*(1-fx)+pixel(x0+1, y0+1, i)*fx)*fy + .5)
	}
	return color.RGBA{c[0], c[1], c[2], 255}
}

// SetSeriesTexture replaces the image of an already textured series without
// resampling or rebuilding its mesh. Invalid input leaves the series intact.
// Execute the returned command after Init. Err reports errors.
func (m *Model) SetSeriesTexture(name string, texture *Texture) tea.Cmd {
	if !texture.valid() {
		m.err = fmt.Errorf("ntcharts3d: invalid texture")
		return nil
	}
	for i, s := range m.series {
		if s.name != name {
			continue
		}
		if !textured(s.geometry) {
			m.err = fmt.Errorf("ntcharts3d: series %q is not textured", name)
			return nil
		}
		if s.geometry.Material.Texture == texture {
			m.err = nil
			return nil
		}
		material := cloneMaterial(s.geometry.Material)
		material.Texture = texture
		s.geometry.Material = material
		source := s.source
		if old, ok := source.(textureSeries); ok {
			source = old.Series
		}
		s.source = textureSeries{source, texture}
		m.series = slices.Clone(m.series)
		m.series[i] = s
		// Copy the snapshot headers so an in-flight frame keeps its old material.
		if m.preparedRevision == m.revision && m.prepared != nil {
			m.prepared = slices.Clone(m.prepared)
			m.prepared[i].Material = material
		}
		m.textureRevision++
		m.err = nil
		return m.changed(false)
	}
	m.err = fmt.Errorf("ntcharts3d: unknown series %q", name)
	return nil
}

type textureSeries struct {
	Series
	texture *Texture
}

func (s textureSeries) apply(g Geometry, err error) (Geometry, error) {
	if err == nil {
		g.Material = cloneMaterial(g.Material)
		if g.Material == nil {
			return g, fmt.Errorf("ntcharts3d: textured series lost its material")
		}
		g.Material.Texture = s.texture
	}
	return g, err
}
func (s textureSeries) Geometry(p Palette) (Geometry, error) { return s.apply(s.Series.Geometry(p)) }
func (s textureSeries) GeometryWithColorDomain(p Palette, lo, hi float32) (Geometry, error) {
	if ds, ok := s.Series.(ColorDomainSeries); ok {
		return s.apply(ds.GeometryWithColorDomain(p, lo, hi))
	}
	return Geometry{}, fmt.Errorf("ntcharts3d: series %q does not implement ColorDomainSeries", s.Name())
}

// Bounds returns the snapshot dimensions with a zero origin.
func (t *Texture) Bounds() image.Rectangle {
	if t == nil {
		return image.Rectangle{}
	}
	return image.Rect(0, 0, t.width, t.height)
}

// ColorModel implements image.Image without exposing mutable storage.
func (t *Texture) ColorModel() color.Model { return color.RGBAModel }

// At implements image.Image. Pixels outside Bounds are transparent black.
func (t *Texture) At(x, y int) color.Color {
	if t == nil || x < 0 || y < 0 || x >= t.width || y >= t.height {
		return color.RGBA{}
	}
	i := (y*t.width + x) * 4
	return color.RGBA{t.pixels[i], t.pixels[i+1], t.pixels[i+2], 255}
}
