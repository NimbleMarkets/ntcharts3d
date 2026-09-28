// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"fmt"
	"image/color"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts3d/math3d"
)

// Series supplies a named dataset as geometry. Geometry is compiled when data,
// palettes, or shared-domain settings change, never on camera movement.
type Series interface {
	Name() string
	Geometry(Palette) (Geometry, error)
}

// ColorDomainSeries lets a custom Series map values against a shared color domain.
type ColorDomainSeries interface {
	GeometryWithColorDomain(Palette, float32, float32) (Geometry, error)
}

type series struct {
	name     string
	geometry Geometry
	source   Series
}

func validColumns(n int, optional ...int) error {
	if n < 1 || n > MaxPoints {
		return fmt.Errorf("ntcharts3d: datum count must be 1..%d", MaxPoints)
	}
	for _, v := range optional {
		if v != 0 && v != n {
			return fmt.Errorf("ntcharts3d: column length %d != %d", v, n)
		}
	}
	return nil
}

func validFloat(v ...float32) bool {
	for _, x := range v {
		if !math3d.IsFinite(float64(x)) {
			return false
		}
	}
	return true
}

// SetPalette copies color stops and recompiles series using the new palette.
// Empty palettes are ignored; invalid colors leave the existing data intact.
// Execute the returned command when calling this after Init.
func (m *Model) SetPalette(cs []color.Color) tea.Cmd {
	if len(cs) == 0 {
		return nil
	}
	converted := make(Palette, len(cs))
	for i, c := range cs {
		if c == nil {
			m.err = fmt.Errorf("ntcharts3d: nil scale color")
			return nil
		}
		converted[i] = color.RGBAModel.Convert(c).(color.RGBA)
	}
	ss := slices.Clone(m.series)
	for i := range ss {
		g, err := ss[i].source.Geometry(converted)
		if err == nil {
			err = validateGeometry(g)
		}
		if err != nil {
			m.err = err
			return nil
		}
		ss[i].geometry = g
	}
	if m.sharedColorDomain || m.colorDomain != nil {
		if err := compileColors(ss, converted, m.sharedColorDomain, m.colorDomain); err != nil {
			m.err = err
			return nil
		}
	}
	m.colors, m.series = converted, ss
	m.err = nil
	_ = m.resizePicture()
	return m.changed(true)
}

// SetSeries compiles geometry synchronously. Call it before Init or from Update.
// A matching name replaces the existing series in place; a new name appends it.
// Providers must not mutate returned geometry. Err reports validation errors.
func (m *Model) SetSeries(s Series) tea.Cmd {
	if s == nil {
		m.err = fmt.Errorf("ntcharts3d: nil series")
		return nil
	}
	g, err := s.Geometry(m.colors)
	if err == nil {
		err = validateGeometry(g)
	}
	if err != nil {
		m.err = err
		return nil
	}
	next := slices.Clone(m.series)
	index := -1
	count := len(g.Points)
	for i, old := range next {
		if old.name == s.Name() {
			index = i
		} else {
			count += len(old.geometry.Points)
		}
	}
	if count > m.maxPoints {
		m.err = fmt.Errorf("ntcharts3d: point cap %d exceeded", m.maxPoints)
		return nil
	}
	entry := series{s.Name(), g, s}
	if index < 0 {
		next = append(next, entry)
	} else {
		next[index] = entry
	}
	if err := m.axes.validate(next); err != nil {
		m.err = err
		return nil
	}
	if m.sharedColorDomain || m.colorDomain != nil {
		if err := compileColors(next, m.colors, m.sharedColorDomain, m.colorDomain); err != nil {
			m.err = err
			return nil
		}
	}
	m.series = next
	_ = m.resizePicture()
	m.err = nil
	m.hover = nil
	return m.changed(true)
}

func compileColors(ss []series, cs Palette, shared bool, domain *Range) error {
	lo, hi, found := float32(0), float32(0), false
	base := make([]Geometry, len(ss))
	for i, s := range ss {
		g, err := s.source.Geometry(cs)
		if err != nil {
			return err
		}
		if err := validateGeometry(g); err != nil {
			return err
		}
		base[i] = g
		if g.HasColorRange {
			if !found {
				lo, hi, found = g.ColorMin, g.ColorMax, true
			} else {
				lo, hi = min(lo, g.ColorMin), max(hi, g.ColorMax)
			}
		}
	}
	if domain != nil {
		lo, hi, found = domain.Min, domain.Max, true
	}
	if (!shared && domain == nil) || !found {
		for i := range ss {
			ss[i].geometry = base[i]
		}
		return nil
	}
	for i, s := range ss {
		if !base[i].HasColorRange {
			s.geometry = base[i]
			ss[i] = s
			continue
		}
		if ds, ok := s.source.(ColorDomainSeries); ok {
			g, err := ds.GeometryWithColorDomain(cs, lo, hi)
			if err != nil {
				return err
			}
			if err = validateGeometry(g); err != nil {
				return err
			}
			s.geometry = g
		} else {
			return fmt.Errorf("ntcharts3d: series %q has a color range but does not implement ColorDomainSeries", s.name)
		}
		ss[i] = s
	}
	return nil
}

// SetSharedColorDomain maps series with color ranges against their combined
// minimum and maximum. Custom series must implement ColorDomainSeries.
func (m *Model) SetSharedColorDomain(enabled bool) tea.Cmd {
	if m.sharedColorDomain == enabled {
		return nil
	}
	ss := slices.Clone(m.series)
	if err := compileColors(ss, m.colors, enabled, m.colorDomain); err != nil {
		m.err = err
		return nil
	}
	m.err = nil
	m.sharedColorDomain, m.series = enabled, ss
	return m.changed(true)
}

func (m *Model) Clear() tea.Cmd {
	m.series = nil
	m.hover = nil
	_ = m.resizePicture()
	return m.changed(true)
}

func (m *Model) bounds() math3d.AABB {
	var b math3d.AABB
	for _, s := range m.series {
		if s.geometry.Bounds.IsValid() {
			b.Include(s.geometry.Bounds.Min)
			b.Include(s.geometry.Bounds.Max)
		}
	}
	return b
}

// Bounds returns the combined data bounds of the current series.
func (m *Model) Bounds() math3d.AABB { return m.bounds() }
