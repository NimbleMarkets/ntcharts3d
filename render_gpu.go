// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"context"
	_ "embed"
	"encoding/binary"
	"errors"
	gpuimage "github.com/NimbleMarkets/go-gpuimage"
	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
	"image"
	"image/color"
	"math"
	"sync"
)

//go:embed scene.wgsl
var sceneShader string

type gpuBatch struct {
	buffer *wgpu.Buffer
	bind   *wgpu.BindGroup
	count  uint32
}

type gpuRenderer struct {
	mu         sync.Mutex
	closed     bool
	r          *gpuimage.Renderer
	shader     *wgpu.ShaderModule
	bindLayout *wgpu.BindGroupLayout
	layout     *wgpu.PipelineLayout
	uniform    *wgpu.Buffer
	pipelines  [6]*wgpu.RenderPipeline
	batches    [6]gpuBatch // scatter, triangles, boxes, lines, screen overlay, grid
	revision   uint64
	uploads    uint64
}

func (g *gpuRenderer) setup() (err error) {
	g.r, err = gpuimage.New(context.Background(), gpuimage.Options{Width: 1, Height: 1, Depth: true, Label: "ntcharts3d", PowerPreference: gputypes.PowerPreferenceHighPerformance})
	if err != nil {
		return err
	}
	if g.r.AdapterInfo().DeviceType == gputypes.DeviceTypeCPU {
		_ = g.r.Close()
		g.r = nil
		return gpuimage.ErrNoAdapter
	}
	return g.r.Do(func(dev *wgpu.Device) error {
		g.shader, err = dev.CreateShaderModule(&wgpu.ShaderModuleDescriptor{Label: "ntcharts3d", WGSL: sceneShader})
		if err != nil {
			return err
		}
		g.bindLayout, err = dev.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{Entries: []gputypes.BindGroupLayoutEntry{{Binding: 0, Visibility: wgpu.ShaderStageVertex, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform}}, {Binding: 1, Visibility: wgpu.ShaderStageVertex, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeReadOnlyStorage}}}})
		if err != nil {
			return err
		}
		g.layout, err = dev.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{BindGroupLayouts: []*wgpu.BindGroupLayout{g.bindLayout}})
		if err != nil {
			return err
		}
		for i, entry := range []string{"vs_point", "vs_mesh", "vs_box", "vs_line", "vs_ui", "vs_grid"} {
			primitive := gputypes.PrimitiveState{Topology: gputypes.PrimitiveTopologyTriangleList, CullMode: gputypes.CullModeBack}
			if i == 0 {
				primitive.CullMode = gputypes.CullModeNone
			}
			if i == 3 {
				primitive.CullMode = gputypes.CullModeNone
			}
			if i >= 4 {
				primitive.CullMode = gputypes.CullModeNone
			}
			g.pipelines[i], err = dev.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{Label: entry, Layout: g.layout, Vertex: wgpu.VertexState{Module: g.shader, EntryPoint: entry}, Primitive: primitive, DepthStencil: &wgpu.DepthStencilState{Format: gputypes.TextureFormatDepth24Plus, DepthWriteEnabled: true, DepthCompare: gputypes.CompareFunctionLessEqual}, Multisample: gputypes.DefaultMultisampleState(), Fragment: &wgpu.FragmentState{Module: g.shader, EntryPoint: "fs_main", Targets: []gputypes.ColorTargetState{{Format: gputypes.TextureFormatRGBA8Unorm, WriteMask: gputypes.ColorWriteMaskAll}}}})
			if err != nil {
				return err
			}
		}
		g.uniform, err = dev.CreateBuffer(&wgpu.BufferDescriptor{Size: 112, Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst})
		return err
	})
}

func floats(values ...float32) []byte {
	out := make([]byte, len(values)*4)
	for i, v := range values {
		binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(v))
	}
	return out
}

func (g *gpuRenderer) upload(dev *wgpu.Device, f Frame) error {
	var data [4][]byte
	vertex := func(kind int, v Vertex, radius float32) {
		data[kind] = append(data[kind], floats(v.Position.X, v.Position.Y, v.Position.Z, radius, v.Normal.X, v.Normal.Y, v.Normal.Z, 0, float32(v.Color.R)/255, float32(v.Color.G)/255, float32(v.Color.B)/255, float32(v.Color.A)/255)...)
	}
	for _, geom := range f.Geometry {
		for _, p := range geom.Points {
			vertex(0, Vertex{Position: p.Position, Color: p.Color}, p.Radius)
		}
		for _, i := range geom.Indices {
			vertex(1, geom.Vertices[i], 1)
		}
		for _, b := range geom.Boxes {
			vertex(2, Vertex{b.Min, b.Size, b.Color}, 1)
		}
		for _, v := range geom.Lines {
			vertex(3, v, 1)
		}
	}
	for i, d := range data {
		old := &g.batches[i]
		if old.bind != nil {
			old.bind.Release()
		}
		if old.buffer != nil {
			old.buffer.Release()
		}
		*old = gpuBatch{}
		if len(d) == 0 {
			continue
		}
		var err error
		old.buffer, err = dev.CreateBuffer(&wgpu.BufferDescriptor{Size: uint64(len(d)), Usage: wgpu.BufferUsageStorage | wgpu.BufferUsageCopyDst})
		if err != nil {
			return err
		}
		if err = dev.Queue().WriteBuffer(old.buffer, 0, d); err != nil {
			return err
		}
		old.bind, err = dev.CreateBindGroup(&wgpu.BindGroupDescriptor{Layout: g.bindLayout, Entries: []wgpu.BindGroupEntry{{Binding: 0, Buffer: g.uniform, Size: 112}, {Binding: 1, Buffer: old.buffer, Size: uint64(len(d))}}})
		if err != nil {
			return err
		}
		old.count = uint32(len(d) / 48)
	}
	g.revision = f.Revision
	g.uploads++
	return nil
}

func (g *gpuRenderer) uploadUI(dev *wgpu.Device, f Frame) error {
	var data []byte
	vertex := func(x, y int, c color.RGBA) {
		data = append(data, floats(float32(x), float32(y), 0, 1, 0, 0, 0, 0, float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255)...)
	}
	for _, r := range frameOverlayRects(f) {
		x0, y0, x1, y1 := r.x, r.y, r.x+r.w, r.y+r.h
		vertex(x0, y0, r.c)
		vertex(x1, y0, r.c)
		vertex(x1, y1, r.c)
		vertex(x0, y0, r.c)
		vertex(x1, y1, r.c)
		vertex(x0, y1, r.c)
	}
	return g.uploadDynamic(dev, 4, data)
}

func (g *gpuRenderer) uploadGrid(dev *wgpu.Device, f Frame) error {
	var data []byte
	for _, v := range f.GridLines {
		data = append(data, floats(v.Position.X, v.Position.Y, v.Position.Z, 1, 0, 0, 0, 0, float32(v.Color.R)/255, float32(v.Color.G)/255, float32(v.Color.B)/255, 1)...)
	}
	return g.uploadDynamic(dev, 5, data)
}

func (g *gpuRenderer) uploadDynamic(dev *wgpu.Device, index int, data []byte) error {
	b := &g.batches[index]
	if b.bind != nil {
		b.bind.Release()
	}
	if b.buffer != nil {
		b.buffer.Release()
	}
	*b = gpuBatch{}
	if len(data) == 0 {
		return nil
	}
	var err error
	b.buffer, err = dev.CreateBuffer(&wgpu.BufferDescriptor{Size: uint64(len(data)), Usage: wgpu.BufferUsageStorage | wgpu.BufferUsageCopyDst})
	if err != nil {
		return err
	}
	if err = dev.Queue().WriteBuffer(b.buffer, 0, data); err != nil {
		return err
	}
	b.bind, err = dev.CreateBindGroup(&wgpu.BindGroupDescriptor{Layout: g.bindLayout, Entries: []wgpu.BindGroupEntry{{Binding: 0, Buffer: g.uniform, Size: 112}, {Binding: 1, Buffer: b.buffer, Size: uint64(len(data))}}})
	if err != nil {
		return err
	}
	b.count = uint32(len(data) / 48)
	return nil
}

func (g *gpuRenderer) Render(f Frame) (image.Image, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return nil, errors.New("ntcharts3d: renderer closed")
	}
	if g.r == nil {
		if err := g.setup(); err != nil {
			return nil, err
		}
	}
	if w, h := g.r.Size(); w != f.Width || h != f.Height {
		if err := g.r.Resize(f.Width, f.Height); err != nil {
			return nil, err
		}
	}
	ub := append(floats(f.Matrix[:]...), floats(float32(f.Width), float32(f.Height), 0, 0, f.Light.Direction.X, f.Light.Direction.Y, f.Light.Direction.Z, f.Light.Ambient, 0, 0, 0, 0)...)
	img, _, err := g.r.Render(context.Background(), func(frame *gpuimage.Frame) error {
		dev := frame.Device
		if g.uploads == 0 || g.revision != f.Revision {
			if err := g.upload(dev, f); err != nil {
				return err
			}
		}
		if err := g.uploadGrid(dev, f); err != nil {
			return err
		}
		if err := g.uploadUI(dev, f); err != nil {
			return err
		}
		if err := dev.Queue().WriteBuffer(g.uniform, 0, ub); err != nil {
			return err
		}
		pass, err := frame.Encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{ColorAttachments: []wgpu.RenderPassColorAttachment{{View: frame.Color, LoadOp: gputypes.LoadOpClear, StoreOp: gputypes.StoreOpStore, ClearValue: gputypes.Color{R: float64(f.Background.R) / 255, G: float64(f.Background.G) / 255, B: float64(f.Background.B) / 255, A: 1}}}, DepthStencilAttachment: &wgpu.RenderPassDepthStencilAttachment{View: frame.Depth, DepthLoadOp: gputypes.LoadOpClear, DepthStoreOp: gputypes.StoreOpStore, DepthClearValue: 1, StencilReadOnly: true}})
		if err != nil {
			return err
		}
		for _, i := range []int{5, 0, 1, 2, 3, 4} {
			b := g.batches[i]
			if b.count == 0 {
				continue
			}
			pass.SetPipeline(g.pipelines[i])
			pass.SetBindGroup(0, b.bind, nil)
			v, n := b.count, uint32(1)
			if i == 0 {
				v, n = 6, b.count
			}
			if i == 2 {
				v, n = 36, b.count
			}
			if i == 3 || i == 5 {
				v, n = 6, b.count/2
			}
			pass.Draw(gputypes.DrawArgs{VertexCount: v, InstanceCount: n})
		}
		return pass.End()
	})
	return img, err
}

func (g *gpuRenderer) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.closed = true
	if g.r == nil {
		return nil
	}
	_ = g.r.Do(func(*wgpu.Device) error {
		for _, b := range g.batches {
			if b.bind != nil {
				b.bind.Release()
			}
			if b.buffer != nil {
				b.buffer.Release()
			}
		}
		for _, p := range g.pipelines {
			if p != nil {
				p.Release()
			}
		}
		if g.uniform != nil {
			g.uniform.Release()
		}
		if g.layout != nil {
			g.layout.Release()
		}
		if g.bindLayout != nil {
			g.bindLayout.Release()
		}
		if g.shader != nil {
			g.shader.Release()
		}
		return nil
	})
	err := g.r.Close()
	g.r = nil
	return err
}
