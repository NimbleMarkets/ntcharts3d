// ntcharts3d - Copyright (c) 2026 Neomantra Corp.
package ntcharts3d

import (
	"errors"
	"image"
	"nimbleterminal.dev/nt3d"
	"sync"
)

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func renderSoftware(f Frame) (image.Image, error) {
	return nt3d.NewSoftwareRenderer().Render(engineFrame(f))
}

// softwareRenderer owns fallback resources independently of the GPU adapter.
// The lock also protects old nt3d versions: Snapshot can overlap a command.
// Model/frame construction remains on the event loop; no callbacks run here.
type softwareRenderer struct {
	mu     sync.Mutex
	engine nt3d.Renderer
	closed bool
}

func (r *softwareRenderer) Render(f Frame) (image.Image, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, errors.New("ntcharts3d: software renderer closed")
	}
	if r.engine == nil {
		r.engine = nt3d.NewSoftwareRenderer()
	}
	return r.engine.Render(engineFrame(f))
}
func (r *softwareRenderer) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	if r.engine != nil {
		err := r.engine.Close()
		r.engine = nil
		return err
	}
	return nil
}
