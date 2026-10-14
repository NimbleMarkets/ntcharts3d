// ntcharts3d - Copyright (c) 2026 Neomantra Corp.
package ntcharts3d

import (
	"image"
	"nimbleterminal.dev/nt3d"
	"sync"
)

// gpuRenderer adapts chart frames; nt3d owns GPU resources and upload caching.
type gpuRenderer struct {
	mu     sync.Mutex
	engine nt3d.Renderer
	closed bool
}

func (r *gpuRenderer) Render(f Frame) (image.Image, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, nt3d.ErrGPUUnavailable
	}
	if r.engine == nil {
		r.engine = nt3d.NewGPURenderer()
	}
	return r.engine.Render(engineFrame(f))
}
func (r *gpuRenderer) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	if r.engine != nil {
		return r.engine.Close()
	}
	return nil
}
