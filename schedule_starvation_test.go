package ntcharts3d

import (
	"image/color"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestCameraChangesPresentInFlightFrames(t *testing.T) {
	for _, input := range []string{"set-camera", "mouse", "key", "rotate"} {
		t.Run(input, func(t *testing.T) {
			m := New(40, 20, WithRenderMode(Software))
			defer m.Close()
			drain(t, m, m.Init())
			for range 4 {
				before := m.PictureStatus().Presented
				c := m.Camera()
				c.Beta++
				render := m.SetCamera(c)
				if render == nil {
					t.Fatal("missing render")
				}
				for range 3 {
					var cmd tea.Cmd
					switch input {
					case "set-camera":
						c.Beta++
						cmd = m.SetCamera(c)
					case "mouse":
						m.drag = true
						cmd = m.mouse(tea.MouseMotionMsg{X: m.lastX + 1, Y: m.lastY, Button: tea.MouseLeft})
					case "key":
						_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
					case "rotate":
						m.camera.AutoRotate = true
						_, cmd = m.Update(tick{id: m.id, generation: m.wake})
					}
					if cmd != nil {
						t.Fatal("camera change escaped single-flight rendering")
					}
				}
				m.camera.AutoRotate = false
				result := render().(rendered)
				if result.seq == m.seq {
					t.Fatal("expected an older camera pose")
				}
				_, present := m.Update(result)
				if !m.PictureStatus().Busy || m.busy || !m.dirty {
					t.Fatal("older camera frame was discarded instead of presented")
				}
				drain(t, m, present)
				status := m.PictureStatus()
				if status.Busy || status.Presented < before+2 || m.dirty || m.busy {
					t.Fatalf("newest pose did not follow the completed pose: %+v", status)
				}
			}
		})
	}
}

func TestSceneChangesRejectInFlightFrames(t *testing.T) {
	for _, change := range []string{"geometry", "background", "resize-back", "renderer-back", "cell-aspect"} {
		t.Run(change, func(t *testing.T) {
			m := New(40, 20, WithRenderMode(Software))
			defer m.Close()
			drain(t, m, m.Init())
			c := m.Camera()
			c.Beta++
			render := m.SetCamera(c)
			switch change {
			case "geometry":
				drain(t, m, m.SetScatter("new", Scatter{X: []float32{1}, Y: []float32{2}, Z: []float32{3}}))
			case "background":
				drain(t, m, m.SetBackground(color.RGBA{R: 200, A: 255}))
			case "resize-back":
				drain(t, m, m.SetSize(41, 20))
				drain(t, m, m.SetSize(40, 20))
			case "renderer-back":
				drain(t, m, m.SetRenderMode(Wireframe))
				drain(t, m, m.SetRenderMode(Software))
			case "cell-aspect":
				drain(t, m, m.session.SetCellSize(3, 2))
			}
			_, next := m.Update(render())
			if m.PictureStatus().Busy || !m.busy {
				t.Fatalf("invalidated frame was submitted: busy=%v picture=%+v", m.busy, m.PictureStatus())
			}
			drain(t, m, next)
			if m.Err() != nil || m.PictureStatus().Busy {
				t.Fatalf("replacement did not settle: %v %+v", m.Err(), m.PictureStatus())
			}
		})
	}
}

func TestClosedChartIgnoresInFlightCameraFrame(t *testing.T) {
	m := New(40, 20, WithRenderMode(Software))
	defer m.Close()
	drain(t, m, m.Init())
	c := m.Camera()
	c.Beta++
	render := m.SetCamera(c)
	m.SetCamera(c)
	drain(t, m, m.Dispose())
	_, cmd := m.Update(render())
	if cmd != nil || m.PictureStatus().Busy {
		t.Fatal("closed chart presented late work")
	}
}
