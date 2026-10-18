// ntcharts3d - Copyright (c) 2026 Neomantra Corp.
package ntcharts3d

import (
	"fmt"
	"image"
	"reflect"
	"slices"

	tea "charm.land/bubbletea/v2"
	"nimbleterminal.dev/nt3d"
)

type UV = nt3d.UV
type Texture = nt3d.Texture

// Material controls chart triangle appearance. Texture replaces vertex colors.
type Material struct {
	Texture *Texture
	Unlit   bool
}

// NewTexture copies opaque sRGB image pixels into immutable engine storage.
// Both raster renderers filter the texture in linear light using mipmaps.
func NewTexture(src image.Image) (*Texture, error) {
	if src == nil || (reflect.ValueOf(src).Kind() == reflect.Ptr && reflect.ValueOf(src).IsNil()) {
		return nil, fmt.Errorf("ntcharts3d: nil texture image")
	}
	b := src.Bounds()
	if b.Dx() < 1 || b.Dy() < 1 || b.Dx() > 8192 || b.Dy() > 8192 {
		return nil, fmt.Errorf("ntcharts3d: texture dimensions must be 1..8192")
	}
	// Concrete image opacity checks avoid boxing a color for every pixel.
	// Keep the generic path for other image implementations.
	switch img := src.(type) {
	case *image.RGBA:
		if !img.Opaque() {
			return nil, fmt.Errorf("ntcharts3d: texture must be opaque")
		}
		return nt3d.NewTexture(src)
	case *image.NRGBA:
		if !img.Opaque() {
			return nil, fmt.Errorf("ntcharts3d: texture must be opaque")
		}
		return nt3d.NewTexture(src)
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			if a != 65535 {
				return nil, fmt.Errorf("ntcharts3d: texture must be opaque")
			}
		}
	}
	return nt3d.NewTexture(src)
}
func validTexture(t *Texture) bool { return t != nil && !t.Bounds().Empty() }
func cloneMaterial(m *Material) *Material {
	if m == nil {
		return nil
	}
	c := *m
	return &c
}
func textured(g Geometry) bool { return g.Material != nil && g.Material.Texture != nil }

// SetSeriesTexture replaces the image of an already textured series without
// resampling or rebuilding its mesh. Invalid input leaves the series intact.
// Execute the returned command after Init. Err reports errors.
func (m *Model) SetSeriesTexture(name string, texture *Texture) tea.Cmd {
	if !validTexture(texture) {
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
