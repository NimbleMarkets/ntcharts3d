// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"charm.land/lipgloss/v2"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/charmbracelet/x/ansi"
)

func (m *Model) View() tea.View {
	mode := "glyph"
	if m.pic.Mode() == picture.PictureKitty {
		mode = "Kitty"
		if m.transport != "" {
			mode += "/" + m.transport
		}
	}
	count := 0
	for _, s := range m.series {
		count += len(s.geometry.Points) + len(s.geometry.Boxes) + len(s.geometry.Vertices) + len(s.geometry.Lines)/2 + len(s.geometry.Arrows)
	}
	legendMode := "image"
	if m.colorLegendMode == ColorLegendText || m.renderMode == Wireframe {
		legendMode = "text"
	}
	title := fmt.Sprintf("ntcharts3d · %s · %d data · %s · %s · %s legend", m.camera.Projection, count, m.renderMode, mode, legendMode)
	if m.err != nil {
		title += " · " + m.err.Error()
	}
	body := m.pic.View().Content
	if m.renderMode == Wireframe {
		body = m.wire
	}
	if body == "" {
		plotRows := m.plotHeight()
		body = strings.Repeat(strings.Repeat(" ", m.plotWidth())+"\n", max(0, plotRows-1)) + strings.Repeat(" ", m.plotWidth())
	}
	footer := ""
	if m.hover != nil {
		p := m.hover
		footer += fmt.Sprintf(" %s #%d x=%.3g y=%.3g z=%.3g %s", p.SeriesName, p.Datum, p.Position.X, p.Position.Y, p.Position.Z, p.Label)
	} else {
		footer += "  drag orbit · shift pan · wheel zoom · r rotate · o projection · g glyph · v legend"
	}
	clip := func(s string) string { return ansi.Truncate(s, m.width, "") }
	if m.plotWidth() < m.width {
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, " ", m.colorLegendView(m.plotHeight()))
	}
	content := clip(title) + "\n" + m.zones.Mark(m.zoneID, body) + "\n"
	content += clip(footer)
	if m.ownZones {
		content = m.zones.Scan(content)
	}
	v := tea.NewView(content)
	v.MouseMode = tea.MouseModeAllMotion
	return v
}
