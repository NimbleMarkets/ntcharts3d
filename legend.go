// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"charm.land/lipgloss/v2"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// ColorLegendMode selects a legend in the image or as terminal text beside it.
type ColorLegendMode uint8

const (
	ColorLegendImage ColorLegendMode = iota
	ColorLegendText
)

// ColorLegend describes a palette and its numeric range for rendering.
type ColorLegend struct {
	Label   string
	Min     float32
	Max     float32
	Palette Palette
}

const colorLegendPanelWidth = 22

func (m *Model) hasColorRanges() bool {
	if !m.colorLegendVisible {
		return false
	}
	for _, s := range m.series {
		if s.geometry.HasColorRange {
			return true
		}
	}
	return false
}

func (m *Model) plotWidth() int {
	if (m.colorLegendMode == ColorLegendText || m.renderMode == Wireframe) && m.width >= 50 && m.hasColorRanges() {
		return m.width - colorLegendPanelWidth - 1
	}
	return m.width
}

func (m *Model) resizePicture() tea.Cmd {
	if m.width < 1 || m.height < 3 {
		return nil
	}
	return m.pic.SetSize(m.plotWidth(), m.plotHeight())
}

// SetColorLegend sets the legend title. An empty title uses series names,
// or "value" for a shared domain. SetColorLegendVisible controls visibility.
func (m *Model) SetColorLegend(title string) tea.Cmd {
	m.colorLegendTitle = title
	return m.changed(false)
}

func (m *Model) SetColorLegendVisible(visible bool) tea.Cmd {
	if m.colorLegendVisible == visible {
		return nil
	}
	m.colorLegendVisible = visible
	_ = m.resizePicture()
	return m.changed(false)
}

func (m *Model) ColorLegendMode() ColorLegendMode { return m.colorLegendMode }

// SetColorLegendMode switches between image and terminal-text legends.
func (m *Model) SetColorLegendMode(mode ColorLegendMode) tea.Cmd {
	if mode > ColorLegendText || mode == m.colorLegendMode {
		return nil
	}
	m.colorLegendMode = mode
	_ = m.resizePicture()
	return m.changed(false)
}

func (m *Model) colorLegendView(rows int) string {
	var lines []string
	if m.sharedColorDomain || m.colorDomain != nil {
		lo, hi, found := float32(0), float32(0), false
		for _, s := range m.series {
			if g := s.geometry; g.HasColorRange {
				if !found {
					lo, hi, found = g.ColorMin, g.ColorMax, true
				} else {
					lo, hi = min(lo, g.ColorMin), max(hi, g.ColorMax)
				}
			}
		}
		if found {
			label := m.colorLegendTitle
			if label == "" {
				label = "value"
			}
			lines = append(lines, ansi.Truncate(label, colorLegendPanelWidth, ""))
			lines = append(lines, legendRamp(m.colors, lo, hi))
		}
	}
	if !m.sharedColorDomain && m.colorDomain == nil {
		for _, s := range m.series {
			g := s.geometry
			if !g.HasColorRange {
				continue
			}
			label := s.name
			if m.colorLegendTitle != "" {
				label = m.colorLegendTitle
			}
			lines = append(lines, ansi.Truncate(label, colorLegendPanelWidth, ""))
			lines = append(lines, legendRamp(m.colors, g.ColorMin, g.ColorMax))
		}
	}
	for len(lines) < rows {
		lines = append(lines, "")
	}
	if len(lines) > rows {
		lines = lines[:rows]
	}
	return strings.Join(lines, "\n")
}

func legendRamp(scale Palette, lo, hi float32) string {
	minLabel := lipgloss.NewStyle().Width(5).Render(fmt.Sprintf("%.3g", lo))
	maxLabel := lipgloss.NewStyle().Width(5).Align(lipgloss.Right).Render(fmt.Sprintf("%.3g", hi))
	var ramp strings.Builder
	for i := range 8 {
		ramp.WriteString(lipgloss.NewStyle().Foreground(scale.At(float64(i) / 7)).Render("█"))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, minLabel, " ", ramp.String(), " ", maxLabel)
}

func (m *Model) plotHeight() int { return max(1, m.height-2) }
