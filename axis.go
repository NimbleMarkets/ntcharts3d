// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"fmt"
	"slices"
	"strconv"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts3d/math3d"
)

// Axis configures one shared coordinate axis. The zero value shows a numeric
// axis with automatic ticks. An empty Name uses X, Y, or Z.
type Axis struct {
	Range      *Range // Optional fixed numeric bounds. Geometry outside them is clipped.
	Name       string
	Categories []string // Names at coordinates 0, 1, ...; at most 512 entries.
	TickCount  int      // Target numeric tick count, 2–64; zero uses five.
	// Format formats numeric tick values. It runs synchronously when a frame is
	// prepared and may run again after camera or size changes. Nil uses numbers.
	Format                                  func(float64) string
	Hidden, HideName, HideTicks, HideLabels bool
}

// Axes configures the X, Y, and Z axes shared by all series.
type Axes struct{ X, Y, Z Axis }

// AxisTick pairs a data coordinate with its display label.
type AxisTick struct {
	Value float32
	Label string
}

// AxisFrame is a resolved axis for rendering. Tick labels are already formatted.
// Renderers must not mutate its ticks. Axes in Frame are ordered X, Y, Z.
type AxisFrame struct {
	Name                                    string
	Ticks                                   []AxisTick
	Hidden, HideName, HideTicks, HideLabels bool
}

func (a Axes) list() [3]Axis { return [3]Axis{a.X, a.Y, a.Z} }

func (a Axes) clone() Axes {
	a.X.Range, a.Y.Range, a.Z.Range = cloneRange(a.X.Range), cloneRange(a.Y.Range), cloneRange(a.Z.Range)
	a.X.Categories = slices.Clone(a.X.Categories)
	a.Y.Categories = slices.Clone(a.Y.Categories)
	a.Z.Categories = slices.Clone(a.Z.Categories)
	return a
}

func (a Axes) validate(ss []series) error {
	for i, axis := range a.list() {
		if axis.Range != nil && (!axis.Range.valid() || len(axis.Categories) != 0) {
			return fmt.Errorf("ntcharts3d: %s axis needs a valid numeric range without categories", axisName(i))
		}
		if axis.TickCount != 0 && (axis.TickCount < 2 || axis.TickCount > 64) {
			return fmt.Errorf("ntcharts3d: %s axis tick count must be 2–64 or zero", axisName(i))
		}
		if len(axis.Categories) > 512 {
			return fmt.Errorf("ntcharts3d: %s axis has more than 512 categories", axisName(i))
		}
		if i == 2 || len(axis.Categories) == 0 {
			continue
		}
		for _, s := range ss {
			if b, ok := s.source.(barSeries); ok {
				n := b.data.NX
				if i == 1 {
					n = b.data.NY
				}
				if len(axis.Categories) != n {
					return fmt.Errorf("ntcharts3d: %s axis category count does not match bars %q", axisName(i), s.name)
				}
			}
		}
	}
	return nil
}

// WithAxes sets the initial axes. Category slices are copied by New.
func WithAxes(a Axes) Option { return func(m *Model) { m.SetAxes(a) } }

// SetAxes replaces the shared axes and copies category slices. Invalid settings
// leave the previous axes intact and set Err. Return the command from Update.
func (m *Model) SetAxes(a Axes) tea.Cmd {
	if err := a.validate(m.series); err != nil {
		m.err = err
		return nil
	}
	data := false
	for i, old := range m.axes.list() {
		next := a.list()[i]
		if (old.Range == nil) != (next.Range == nil) || (old.Range != nil && next.Range != nil && *old.Range != *next.Range) {
			data = true
		}
	}
	m.axes = a.clone()
	m.hover = nil
	m.err = nil
	return m.changed(data)
}

// Axes returns the current configuration with copies of its category slices.
func (m *Model) Axes() Axes { return m.axes.clone() }

func axisName(i int) string { return [3]string{"X", "Y", "Z"}[i] }

func (m *Model) axisFrames(b math3d.AABB) [3]AxisFrame {
	var out [3]AxisFrame
	if !b.IsValid() {
		return out
	}
	lo, hi := components(b.Min), components(b.Max)
	for i, a := range m.axes.list() {
		name := a.Name
		if name == "" {
			name = axisName(i)
		}
		f := AxisFrame{Name: name, Hidden: a.Hidden, HideName: a.HideName, HideTicks: a.HideTicks, HideLabels: a.HideLabels}
		if !a.Hidden {
			if len(a.Categories) != 0 {
				for j, label := range a.Categories {
					if float32(j) >= lo[i] && float32(j) <= hi[i] {
						f.Ticks = append(f.Ticks, AxisTick{float32(j), label})
					}
				}
			} else {
				// A chart containing only bars uses integer category positions on X/Y.
				count := 0
				if i < 2 {
					for _, s := range m.series {
						bars, ok := s.source.(barSeries)
						if !ok {
							count = 0
							break
						}
						n := bars.data.NX
						if i == 1 {
							n = bars.data.NY
						}
						count = max(count, n)
					}
				}
				if count > 0 {
					for j := range count {
						if float32(j) >= lo[i] && float32(j) <= hi[i] {
							f.Ticks = append(f.Ticks, AxisTick{float32(j), strconv.Itoa(j)})
						}
					}
				} else {
					f.Ticks = numericTicks(lo[i], hi[i], a.TickCount, a.Format)
				}
			}
		}
		out[i] = f
	}
	return out
}

func (m *Model) datumLabel(si, idx int) string {
	s := m.series[si]
	if b, ok := s.source.(barSeries); ok && idx >= 0 && idx < len(b.data.Values) {
		label := func(a Axis, i int) string {
			if i < len(a.Categories) {
				return a.Categories[i]
			}
			return strconv.Itoa(i)
		}
		return label(m.axes.X, idx%b.data.NX) + " / " + label(m.axes.Y, idx/b.data.NX)
	}
	if idx >= 0 && idx < len(s.geometry.Labels) {
		return s.geometry.Labels[idx]
	}
	return ""
}

func components(v math3d.Vec3) [3]float32 { return [3]float32{v.X, v.Y, v.Z} }
func vector(v [3]float32) math3d.Vec3     { return math3d.Vec3{X: v[0], Y: v[1], Z: v[2]} }
