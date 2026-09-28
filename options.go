// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"image/color"

	"github.com/NimbleMarkets/ntcharts/v2/picture"
	zone "github.com/lrstanley/bubblezone/v2"
)

type Option func(*Model)

func WithKittyID(id int) Option          { return func(m *Model) { m.kittyID = id } }
func WithOrtho() Option                  { return func(m *Model) { m.camera.Projection = Orthographic } }
func WithProjection(p Projection) Option { return func(m *Model) { m.camera.Projection = p } }
func WithAutoRotate(on bool) Option      { return func(m *Model) { m.camera.AutoRotate = on } }
func WithBackground(c color.Color) Option {
	return func(m *Model) { m.background = color.RGBAModel.Convert(c).(color.RGBA); m.background.A = 255 }
}

func WithMaxPoints(n int) Option { return func(m *Model) { m.maxPoints = max(1, min(MaxPoints, n)) } }

// WithRenderMode selects the initial requested and active rendering mode.
// WebGPU is the default; invalid modes are ignored.
func WithRenderMode(l RenderMode) Option {
	return func(m *Model) {
		if l <= Wireframe {
			m.renderMode = l
			m.requestedRenderMode = l
		}
	}
}

func WithRenderer(r Renderer) Option                    { return func(m *Model) { m.renderer = r } }
func WithKittyMedium(medium picture.KittyMedium) Option { return func(m *Model) { m.medium = medium } }

// WithColorLegend sets the legend title. An empty title uses series names,
// or "value" for a shared domain.
func WithColorLegend(title string) Option { return func(m *Model) { m.colorLegendTitle = title } }

// WithSharedColorDomain maps series with color ranges against their combined
// minimum and maximum. The domain updates when data or palettes change.
func WithSharedColorDomain(enabled bool) Option {
	return func(m *Model) { m.sharedColorDomain = enabled }
}

func WithColorLegendMode(mode ColorLegendMode) Option {
	return func(m *Model) {
		if mode <= ColorLegendText {
			m.colorLegendMode = mode
		}
	}
}

// WithZoneManager uses a parent's zone manager. The parent must Scan its full
// View. By default, the chart owns a manager and starts at the terminal origin.
func WithZoneManager(z *zone.Manager) Option { return func(m *Model) { m.zones = z } }
