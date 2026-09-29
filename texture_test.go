// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

package ntcharts3d

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/NimbleMarkets/ntcharts3d/math3d"
)

func testTexture(t *testing.T, c color.RGBA) *Texture {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := range 2 {
		for x := range 2 {
			img.SetRGBA(x, y, c)
		}
	}
	tex, err := NewTexture(img)
	if err != nil {
		t.Fatal(err)
	}
	return tex
}
func TestTextureOwnershipAndValidation(t *testing.T) {
	src := image.NewNRGBA(image.Rect(5, -3, 7, -1))
	colors := []color.NRGBA{{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}, {255, 255, 255, 255}}
	for i, c := range colors {
		src.SetNRGBA(5+i%2, -3+i/2, c)
	}
	tex, err := NewTexture(src)
	if err != nil {
		t.Fatal(err)
	}
	src.SetNRGBA(5, -3, color.NRGBA{})
	if tex.Bounds() != image.Rect(0, 0, 2, 2) || tex.At(0, 0) != (color.RGBA{255, 0, 0, 255}) {
		t.Fatal("snapshot bounds or ownership")
	}
	for i, uv := range []UV{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		if got := tex.sample(uv); got != (color.RGBA(colors[i])) {
			t.Fatal("orientation", uv, got)
		}
	}
	if got := tex.sample(UV{.5, .5}); got != (color.RGBA{128, 128, 128, 255}) {
		t.Fatal("bilinear", got)
	}
	if tex.sample(UV{-2, 3}) != tex.sample(UV{0, 1}) {
		t.Fatal("clamp")
	}
	semi := image.NewNRGBA64(image.Rect(0, 0, 1, 1))
	semi.SetNRGBA64(0, 0, color.NRGBA64{A: 65534})
	var nilImage *image.RGBA
	for _, src := range []image.Image{nil, nilImage, semi, image.NewRGBA(image.Rect(0, 0, 0, 1)), image.NewRGBA(image.Rect(0, 0, 1, 1))} {
		if _, err := NewTexture(src); err == nil {
			t.Fatal("accepted invalid image")
		}
	}
}
func TestTexturedSurfaceAndUpdates(t *testing.T) {
	tex := testTexture(t, color.RGBA{255, 0, 0, 255})
	next := testTexture(t, color.RGBA{0, 255, 0, 255})
	samples := 0
	material := &Material{Texture: tex, Unlit: true}
	m := New(40, 20)
	defer m.Close()
	m.SetSurface("terrain", Surface{NX: 2, NY: 2, Sampler: func(x, y float64) float64 { samples++; return x + y }, Material: material})
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
	material.Texture = next
	old := m.frame()
	g := old.Geometry[0]
	if g.Material.Texture != tex || g.HasColorRange {
		t.Fatal("material ownership or legend")
	}
	want := []UV{{0, 1}, {1, 1}, {0, 0}, {1, 0}}
	for i, uv := range g.UV {
		if uv != want[i] {
			t.Fatal("default UV", uv)
		}
	}
	count := samples
	m.SetSeriesTexture("terrain", next)
	updated := m.frame()
	if m.Err() != nil || samples != count || updated.Revision != old.Revision || updated.TextureRevision == old.TextureRevision {
		t.Fatal("texture update rebuilt mesh", m.Err())
	}
	if old.Geometry[0].Material.Texture != tex || updated.Geometry[0].Material.Texture != next || &old.Geometry[0].Vertices[0] != &updated.Geometry[0].Vertices[0] {
		t.Fatal("snapshot mutation or mesh rebuild")
	}
	m.SetSeriesTexture("missing", tex)
	if m.Err() == nil {
		t.Fatal("missing series accepted")
	}
	m.SetSeriesTexture("terrain", &Texture{})
	if m.Err() == nil {
		t.Fatal("zero texture accepted")
	}
	m.SetPalette([]color.Color{color.Black, color.White})
	if m.Err() != nil || m.frame().Geometry[0].Material.Texture != next {
		t.Fatal("palette lost replacement", m.Err())
	}
	m.SetSharedColorDomain(true)
	if m.Err() != nil || m.frame().Geometry[0].Material.Texture != next {
		t.Fatal("shared domain lost replacement", m.Err())
	}
	m.SetSurface("legend", Surface{NX: 2, NY: 2, Z: []float32{0, 1, 2, 3}, Material: &Material{Texture: tex}, ColorLegend: true})
	if m.Err() != nil || !m.series[1].geometry.HasColorRange {
		t.Fatal("explicit legend", m.Err())
	}
}
func TestTextureUVValidationAndClipping(t *testing.T) {
	tex := testTexture(t, color.RGBA{255, 255, 255, 255})
	for _, uv := range [][]UV{{{0, 0}}, {{0, 0}, {0, 0}, {0, 0}, {float32(math.NaN()), 0}}} {
		m := New(20, 10)
		m.SetSurface("bad", Surface{NX: 2, NY: 2, Z: make([]float32, 4), UV: uv, Material: &Material{Texture: tex}})
		if m.Err() == nil {
			t.Fatal("invalid UV accepted")
		}
		m.Close()
	}
	g, err := (surfaceSeries{"terrain", Surface{NX: 2, NY: 2, Z: make([]float32, 4), Material: &Material{Texture: tex}}}).Geometry(defaultPalette())
	if err != nil {
		t.Fatal(err)
	}
	var bounds math3d.AABB
	bounds.Include(math3d.Vec3{X: -.5, Y: -.5, Z: -1})
	bounds.Include(math3d.Vec3{X: .5, Y: .5, Z: 1})
	clipped := clipGeometry(g, bounds)
	if len(clipped.Indices) == 0 || len(clipped.UV) != len(clipped.Vertices) {
		t.Fatal("missing clipped mesh")
	}
	for i, v := range clipped.Vertices {
		uv := clipped.UV[i]
		if abs(uv.U-(v.Position.X+1)/2) > 1e-6 || abs(uv.V-(1-v.Position.Y)/2) > 1e-6 {
			t.Fatal("clipping lost UV", v, uv)
		}
	}
}
func textureTestFrame(t *testing.T) Frame {
	t.Helper()
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.SetRGBA(0, 0, color.RGBA{255, 0, 0, 255})
	src.SetRGBA(1, 0, color.RGBA{0, 255, 0, 255})
	src.SetRGBA(0, 1, color.RGBA{0, 0, 255, 255})
	src.SetRGBA(1, 1, color.RGBA{255, 255, 255, 255})
	tex, err := NewTexture(src)
	if err != nil {
		t.Fatal(err)
	}
	g, err := (surfaceSeries{"terrain", Surface{NX: 2, NY: 2, Z: []float32{.5, .5, .5, .5}, Material: &Material{Texture: tex, Unlit: true}}}).Geometry(defaultPalette())
	if err != nil {
		t.Fatal(err)
	}
	return Frame{Width: 100, Height: 100, Revision: 1, Matrix: math3d.Identity(), Geometry: []Geometry{g}, Background: color.RGBA{A: 255}, Light: Light{Direction: math3d.Vec3{Z: -1}, Ambient: .25}}
}
func TestSoftwareTextureOrientationAndLighting(t *testing.T) {
	f := textureTestFrame(t)
	img := softwareRender(f)
	for _, test := range []struct {
		x, y int
		c    color.RGBA
	}{{10, 10, color.RGBA{255, 0, 0, 255}}, {90, 10, color.RGBA{0, 255, 0, 255}}, {10, 90, color.RGBA{0, 0, 255, 255}}, {90, 90, color.RGBA{255, 255, 255, 255}}} {
		if got := img.At(test.x, test.y); got != test.c {
			t.Fatal("texture orientation/color", test, got)
		}
	}
	f.Geometry[0].Material.Unlit = false
	lit := softwareRender(f)
	if got := lit.At(10, 10).(color.RGBA).R; got < 62 || got > 64 {
		t.Fatal("texture lighting", got)
	}
}
func TestSoftwarePerspectiveTexture(t *testing.T) {
	f := textureTestFrame(t)
	// w = 1 + x/2; a right-side pixel must sample the perspective-correct UV.
	f.Matrix[3] = .5
	img := softwareRender(f)
	x, y := 55, 45
	ndcX := 2*(float32(x)+.5)/100 - 1
	worldX := ndcX / (1 - .5*ndcX)
	w := 1 + .5*worldX
	worldY := (1 - 2*(float32(y)+.5)/100) * w
	want := f.Geometry[0].Material.Texture.sample(UV{(worldX + 1) / 2, (1 - worldY) / 2})
	got := img.At(x, y).(color.RGBA)
	if abs(float32(got.R)-float32(want.R)) > 1 || abs(float32(got.G)-float32(want.G)) > 1 || abs(float32(got.B)-float32(want.B)) > 1 {
		t.Fatal("perspective interpolation", got, want)
	}
}
