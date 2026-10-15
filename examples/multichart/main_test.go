package main

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts3d"
	uv "github.com/charmbracelet/ultraviolet"
	picture "nimbleterminal.dev/ntpicture"
)

func TestSharedDiscoveryAndIndependentControls(t *testing.T) {
	t.Setenv("TERM", "xterm-kitty")
	t.Setenv("NTPICTURE_KITTY", "")
	t.Setenv("TMUX", "")
	d := newDashboard(ntcharts3d.Wireframe)
	defer d.close()
	queries, notifications := 0, 0
	var visit func(tea.Cmd)
	visit = func(cmd tea.Cmd) {
		if cmd == nil {
			return
		}
		switch msg := cmd().(type) {
		case tea.BatchMsg:
			for _, child := range msg {
				visit(child)
			}
		case tea.RawMsg:
			// Count direct Kitty support queries, excluding the shared-memory probe.
			queries += strings.Count(fmt.Sprint(msg.Msg), "a=q,t=d")
		case picture.TerminalSessionMsg:
			if msg.Session != d.session {
				t.Fatal("chart initialized a different session")
			}
			notifications++
		}
	}
	visit(d.Init())
	if queries != 1 || notifications != 1 {
		t.Fatalf("want one shared discovery, got %d Kitty queries and %d session notifications", queries, notifications)
	}
	before := d.charts[0].Camera()
	d.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	d.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if d.charts[0].Camera() != before || d.charts[1].Camera().Beta == before.Beta {
		t.Fatal("keyboard input did not stay with the focused chart")
	}
	d.Update(uv.CellSizeEvent{Width: 10, Height: 20})
	if state := d.session.Snapshot(); state.CellPixelWidth != 10 || state.CellPixelHeight != 20 {
		t.Fatal("terminal geometry was not forwarded to the shared session")
	}
	// Disposing one chart must leave the host-owned session usable by the other.
	d.charts[0].Dispose()
	d.Update(uv.CellSizeEvent{Width: 12, Height: 24})
	if d.session.Snapshot().CellPixelWidth != 12 {
		t.Fatal("chart disposal closed the shared session")
	}
}
