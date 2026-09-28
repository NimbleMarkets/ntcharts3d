package main

import (
	"context"
	_ "embed"
	"fmt"
	"image"

	gpuimage "github.com/NimbleMarkets/go-gpuimage"
	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
)

//go:embed shader.wgsl
var shaderWGSL string

type frameRequest struct {
	width, height    int
	yaw, pitch, dist float64
	pointSize        float32
}

type sceneRenderer interface {
	Render(req frameRequest) (*image.NRGBA, gpuimage.Stats, error)
	Name() string
	Close()
}

// scene owns the point-cloud pipeline and buffers on top of a
// gpuimage Renderer.
type scene struct {
	r          *gpuimage.Renderer
	shader     *wgpu.ShaderModule
	bindLayout *wgpu.BindGroupLayout
	layout     *wgpu.PipelineLayout
	pipeline   *wgpu.RenderPipeline
	uniform    *wgpu.Buffer
	points     *wgpu.Buffer
	bindGroup  *wgpu.BindGroup
	count      uint32
}

func newScene(ctx context.Context, pts []point, w, h int) (*scene, error) {
	r, err := gpuimage.New(ctx, gpuimage.Options{
		Width: w, Height: h, Depth: true, Label: "points",
		PowerPreference: gputypes.PowerPreferenceHighPerformance,
	})
	if err != nil {
		return nil, err
	}
	s := &scene{r: r, count: uint32(len(pts))}
	if err := r.Do(func(dev *wgpu.Device) error { return s.setup(dev, pts) }); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

func (s *scene) setup(dev *wgpu.Device, pts []point) (err error) {
	s.shader, err = dev.CreateShaderModule(&wgpu.ShaderModuleDescriptor{Label: "points", WGSL: shaderWGSL})
	if err != nil {
		return fmt.Errorf("shader: %w", err)
	}
	s.bindLayout, err = dev.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "points",
		Entries: []gputypes.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageVertex, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform}},
			{Binding: 1, Visibility: wgpu.ShaderStageVertex, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeReadOnlyStorage}},
		},
	})
	if err != nil {
		return fmt.Errorf("bind group layout: %w", err)
	}
	s.layout, err = dev.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{Label: "points", BindGroupLayouts: []*wgpu.BindGroupLayout{s.bindLayout}})
	if err != nil {
		return fmt.Errorf("pipeline layout: %w", err)
	}
	s.pipeline, err = dev.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label:  "points",
		Layout: s.layout,
		Vertex: wgpu.VertexState{Module: s.shader, EntryPoint: "vs_main"},
		Primitive: gputypes.PrimitiveState{
			Topology: gputypes.PrimitiveTopologyTriangleList,
			CullMode: gputypes.CullModeNone,
		},
		DepthStencil: &wgpu.DepthStencilState{
			Format:            gputypes.TextureFormatDepth24Plus,
			DepthWriteEnabled: true,
			DepthCompare:      gputypes.CompareFunctionLess,
		},
		Multisample: gputypes.DefaultMultisampleState(),
		Fragment: &wgpu.FragmentState{
			Module:     s.shader,
			EntryPoint: "fs_main",
			Targets: []gputypes.ColorTargetState{{
				Format:    gputypes.TextureFormatRGBA8Unorm,
				WriteMask: gputypes.ColorWriteMaskAll,
			}},
		},
	})
	if err != nil {
		return fmt.Errorf("render pipeline: %w", err)
	}
	packed := packPoints(pts)
	s.points, err = dev.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "points:storage", Size: uint64(len(packed)),
		Usage: wgpu.BufferUsageStorage | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return fmt.Errorf("points buffer: %w", err)
	}
	if err := dev.Queue().WriteBuffer(s.points, 0, packed); err != nil {
		return fmt.Errorf("write points: %w", err)
	}
	s.uniform, err = dev.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "points:uniform", Size: 80,
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return fmt.Errorf("uniform buffer: %w", err)
	}
	s.bindGroup, err = dev.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "points",
		Layout: s.bindLayout,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: s.uniform, Size: 80},
			{Binding: 1, Buffer: s.points, Size: uint64(len(packed))},
		},
	})
	if err != nil {
		return fmt.Errorf("bind group: %w", err)
	}
	return nil
}

func (s *scene) Name() string { return s.r.AdapterInfo().Name }

func (s *scene) Render(req frameRequest) (*image.NRGBA, gpuimage.Stats, error) {
	if req.width < 1 || req.height < 1 {
		return nil, gpuimage.Stats{}, fmt.Errorf("invalid raster %dx%d", req.width, req.height)
	}
	if w, h := s.r.Size(); w != req.width || h != req.height {
		if err := s.r.Resize(req.width, req.height); err != nil {
			return nil, gpuimage.Stats{}, err
		}
	}
	aspect := float32(req.width) / float32(req.height)
	proj := perspective(0.9, aspect, 0.1, 100)
	view := lookAt(orbitEye(req.yaw, req.pitch, req.dist), vec3{}, vec3{0, 1, 0})
	ub := uniformBytes(mul(proj, view), req.width, req.height, req.pointSize)
	return s.r.Render(context.Background(), func(f *gpuimage.Frame) error {
		if err := f.Device.Queue().WriteBuffer(s.uniform, 0, ub); err != nil {
			return err
		}
		pass, err := f.Encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
			Label: "points",
			ColorAttachments: []wgpu.RenderPassColorAttachment{{
				View: f.Color, LoadOp: gputypes.LoadOpClear, StoreOp: gputypes.StoreOpStore,
				ClearValue: gputypes.Color{R: 0.06, G: 0.07, B: 0.09, A: 1},
			}},
			// StencilReadOnly: gogpu's browser backend emits stencilLoadOp/
			// stencilStoreOp unless the stencil aspect is read-only, and
			// WebGPU rejects those on a depth-only format such as Depth24Plus.
			// Native backends ignore the flag for a depth-only attachment.
			DepthStencilAttachment: &wgpu.RenderPassDepthStencilAttachment{
				View: f.Depth, DepthLoadOp: gputypes.LoadOpClear, DepthStoreOp: gputypes.StoreOpStore, DepthClearValue: 1,
				StencilReadOnly: true,
			},
		})
		if err != nil {
			return err
		}
		pass.SetPipeline(s.pipeline)
		pass.SetBindGroup(0, s.bindGroup, nil)
		pass.Draw(gputypes.DrawArgs{VertexCount: 6, InstanceCount: s.count})
		return pass.End()
	})
}

// Close releases the scene's GPU objects on the renderer thread, then the
// renderer itself. Explicit nil checks per field: a partially built scene
// (setup failed midway) must not dereference nil, and a typed nil stored in
// an interface would defeat a generic `!= nil` loop.
func (s *scene) Close() {
	_ = s.r.Do(func(*wgpu.Device) error {
		if s.bindGroup != nil {
			s.bindGroup.Release()
		}
		if s.uniform != nil {
			s.uniform.Release()
		}
		if s.points != nil {
			s.points.Release()
		}
		if s.pipeline != nil {
			s.pipeline.Release()
		}
		if s.layout != nil {
			s.layout.Release()
		}
		if s.bindLayout != nil {
			s.bindLayout.Release()
		}
		if s.shader != nil {
			s.shader.Release()
		}
		return nil
	})
	_ = s.r.Close()
}
