# ntcharts3d API

`ntcharts3d` provides Z-up 3D charts for Bubble Tea. It renders through `nimbleterminal.dev/nt3d`
and displays the result through `nimbleterminal.dev/ntpicture.Picture`.

## Terminology and composition

A `Model` is one chart with named series, a shared coordinate space, and one
camera. `Scatter`, `Surface`, `Bars`, `Lines`, and `VectorField` can coexist in a chart. Series setters
replace a matching name in place or append a new name. Custom series implement
`Series` and supply `Geometry` for rendering and picking.

A `Palette` contains color stops. A color domain is the numeric interval mapped
onto those colors. `ColorLegend` displays that mapping. Domains are per series
by default; `WithSharedColorDomain(true)` shares one domain within a chart.

`RenderMode` selects WebGPU, software rasterization, or canvas wireframe.
`RequestedRenderMode()` reports the last explicit selection, which defaults to
WebGPU. `RenderMode()` reports the active mode after any fallback. Kitty/glyph
output and Kitty transport are separate settings.

## Usage

```go
// import "github.com/NimbleMarkets/ntcharts3d"
chart := ntcharts3d.New(100, 36)
defer chart.Close()
chart.SetScatter("measurements", ntcharts3d.Scatter{
    X: xs, Y: ys, Z: zs, ColorValue: temperatures,
})
chart.SetColorLegend("temperature")
if err := chart.Err(); err != nil {
    // Handle invalid data.
}
// Run chart as a tea.Model, or forward messages from a parent model.
```

Setters return `tea.Cmd`. Return that command from your parent's `Update`
method when changing a chart after `Init`. Before `Init`, setters configure
it without scheduling frames. Data validation errors are available through
`Err()` and leave the previous series intact.

Built-in series setters copy their slices. Surface samplers run synchronously
when geometry is compiled: on data, palette, or color-domain changes, rather
than camera movement. A sampler may be called more than once during an update.
`Surface.Sampler` accepts `func(x, y float64) float64`, as heatpicture does.

## Coordinates and appearance

Import `nimbleterminal.dev/nt3d/math` (package `nt3dmath`) for geometry types.
The former `github.com/NimbleMarkets/ntcharts3d/math3d` package has been removed.
`nt3dmath.Vec3` provides `Add`, `Sub`, `Scale`, `Dot`, `Cross`, and `Normalize` methods.
`nt3dmath.Mat4` provides `Mul`, `Transform`, and `Project`. These methods return values
without changing their inputs. `a.Mul(b)` computes `a * b`, applying `b` first
when transforming a point. Both types use `float32` to match WebGPU;
`Normalize` uses `float64` intermediates and leaves a zero vector unchanged.
`nt3dmath.Identity()` creates an identity matrix. `nt3dmath.Ray` holds an origin and
direction. `nt3dmath.AABB.Include` expands bounds; `Normalization` returns the
matrix, center, and scale used to fit them around the origin.

X/Y are horizontal; Z is up. Surface rows advance Y and columns advance X.
Bars occupy integer X/Y category positions, with width and depth 0.8, and
extend from zero to their signed Z value.

The plot is centered at the origin and uniformly scaled so its longest extent
spans two units. `SetPlotAspect(nt3dmath.Vec3{X: 1, Y: 1, Z: .5})` sets relative
plot lengths, useful when dimensions use different units. `WithPlotAspect` sets
the initial proportions. All components must be positive and finite, with a
ratio no greater than 1,000,000; the zero vector restores data proportions.
Picking and lighting follow the same scale. `Camera.Target` and `Camera.Distance` use these normalized
units. `Alpha` is elevation and `Beta` is azimuth, in degrees. Orthographic
projection is the default; perspective is also available. Orthographic is
useful for comparing bar heights.

Meshes use backface culling, a depth buffer, and ambient and directional
lighting. Points are unlit circles with radii clamped to 1–32 pixels on the GPU.
Frames are opaque RGBA8. Packed point colors use `0xRRGGBBAA`; alpha is ignored.
`WithBackground` sets the background, which defaults to light gray.

Axis lines, names, and tick labels appear inside the chart. Status and hover
details appear as terminal text below or above it. The picture uses
`KittyZ=-1` and `FitFill`. Projection accounts for terminal cell dimensions.

## Axes

`Axes` contains the shared X, Y, and Z `Axis` settings. `WithAxes` configures them
at construction; `SetAxes` replaces them and returns a command. `Axes()` returns
a copy. The default names are X, Y, and Z, with numeric ticks from the data bounds.
An axis name describes a dimension; tick labels describe individual coordinates.

```go
chart.SetAxes(ntcharts3d.Axes{
    X: ntcharts3d.Axis{Name: "Hour", Categories: hours},
    Y: ntcharts3d.Axis{Name: "Day", Categories: weekdays},
    Z: ntcharts3d.Axis{Name: "Value", TickCount: 5},
})
```

`Categories` names coordinates 0, 1, and so on. X/Y categories must match every
bar series' NX/NY dimensions; mismatches leave the previous settings or series
intact and set `Err()`. Up to 512 categories per axis are supported. Setters copy
the slices. Bar pick descriptions use these same names, or numeric indices when
no categories are supplied. Charts containing only bars use integer X/Y ticks.

Numeric ticks use intervals of 1, 2, or 5 times a power of ten. `TickCount` is a
target from 2 to 64; zero uses five. `Format func(float64) string` customizes
numeric labels, for example percentages or units. It runs synchronously while
preparing frames and may run again after camera or size changes. It does not
change coordinates or geometry. A constant numeric range gets one tick.

`Hidden` hides an axis. `HideName`, `HideTicks`, and `HideLabels` hide individual
parts. Axes choose outer edges as the camera moves. Text stays upright and its
size does not change with zoom. Names take priority over tick labels; crowded
labels are omitted and long labels are truncated. Labels avoid the image legend.
Axis lines and text are drawn over the scene and do not participate in picking.

GPU and software rendering use the same bitmap text layout and embedded Go Mono
font. It includes Latin, Greek, Cyrillic, and symbols such as `°`, `µ`, and `−`.
Unsupported characters and controls display as `?`; text shaping is not supported. Wireframe draws text
inside its canvas. Glyph image output has less text detail than Kitty images.
Image legends use a smaller panel on small frames to leave room for the plot.

`Bounds()` returns the combined data bounds for an optional external summary,
as shown in the gallery. Axis labels do not change data bounds.

`Axis.Range` fixes numeric bounds. Nil fits the data. Ranges are copied, must be
finite with `Min < Max`, and cannot be combined with categories on the same axis.

```go
chart.SetAxes(ntcharts3d.Axes{
    Z: ntcharts3d.Axis{Name: "Temperature (°C)", Range: &ntcharts3d.Range{Min: -20, Max: 40}},
})
```

`PlotBounds()` applies these ranges to the data bounds. Rendering clips points,
triangles, lines, and bars to the plot; clipped bars have closed faces. Point
centers determine inclusion, so their circles can extend past a boundary.
Picking excludes clipped geometry and reports original data values. Fixed ranges
keep charts comparable when replacing data. An empty chart can still show axes
when all three ranges are fixed.

`WithGrid(Grid{Show: true})` or `SetGrid` adds tick-aligned lines on the three
back planes. The grid follows the camera and is hidden behind data in GPU and
software rendering. It is off by default; `Grid.Color` overrides the muted gray.
Hidden axes omit their grid ticks. Dense categories are sampled to at most 64
grid ticks per dimension. Wireframe has no depth buffer.

## Colors and legends

`SetPalette([]color.Color)` accepts heatmap/heatpicture palettes and uses the
same interpolation. `ColorValue` defaults to Z for scatter points and the bar
value for bars. Surfaces use Z.

`WithSharedColorDomain(true)` maps all series with color ranges against their
combined minimum and maximum. `SetSharedColorDomain` changes this at runtime.
The automatic domain is recomputed when data or palettes change.

`WithColorDomain(Range{Min: 0, Max: 100})` fixes one palette domain for all mapped
series. `SetColorDomain(&Range{Min: 0, Max: 100})` changes it; values outside the
range use the endpoint colors. `SetColorDomain(nil)` restores automatic domains
and the current shared-domain setting. Explicit point colors are unaffected.
The legend shows the fixed domain. `ColorDomain()` returns a copy or nil.

The color legend is visible by default, with a gradient and numeric range in
the upper-right corner of the image. Press `v` or call `SetColorLegendMode` to
switch between the image and terminal text beside the plot. Use
`WithColorLegendMode(ColorLegendText)` to start in text mode.

`WithColorLegend("temperature")` or `SetColorLegend("temperature")` sets the
title. An empty title uses series names, or "value" for a shared domain.
`SetColorLegendVisible(false)` hides the legend.

Custom series publish their color range through `Geometry.ColorMin`,
`ColorMax`, and `HasColorRange`. They must also implement `ColorDomainSeries`
to support shared or fixed domains.

## Interaction and embedding

| Input | Action |
| --- | --- |
| Drag; arrows or h/j/k/l | Orbit |
| Shift-drag | Pan |
| Wheel; +/- | Zoom |
| r | Toggle idle rotation |
| o | Switch projection |
| g | Switch Kitty/glyph output |
| v | Switch image/text legend |
| q or Ctrl-C | Quit |

Hover shows the datum, coordinates, and labels in the footer. It also draws a
foreground point ring or bar outline. `WithEmphasis(Emphasis{ShowLabel: true})`
adds a short label inside the chart. `SetEmphasis` changes this at runtime;
`Hidden` disables the highlight and `Color` overrides its color. Labels avoid
axes and the legend; crowded labels are omitted. Clicking emits a `PickMsg` with
`ModelID`, `SeriesIndex`, `SeriesName`, `Datum`, `Position`, and `Label`.
`SeriesIndex` is zero-based; `SeriesName` is the name passed to the setter or
returned by `Series.Name()`. Dragging does not emit a pick.

Picking tests projected point radii with a minimum tolerance based on cell
size, projected strokes for lines/arrows, ray/box intersections for bars,
and nearby grid vertices for surfaces.
Overlapping hits select the nearest depth.

By default, the chart owns a BubbleZone manager and starts at the terminal
origin. For a parent layout, pass `WithZoneManager(manager)` and call
`manager.Scan` once on the complete parent `View`. Only the plot rectangle
accepts mouse input. `SetSize` includes two rows for the title and hover details.
Execute `Dispose()` before quitting, then call `Close()` after the program exits
to release GPU resources.

## Lines and vector fields

`SetLines` creates independent line segments. To make a polyline or streamline,
repeat adjacent endpoints in the `Start` and `End` arrays:

```go
cmd := chart.SetLines("trajectory", ntcharts3d.Lines{
    Start: []nt3dmath.Vec3{{X: 0}, {X: 1, Y: 1}},
    End:   []nt3dmath.Vec3{{X: 1, Y: 1}, {X: 2, Y: 1, Z: 1}},
    Width: 3,
    Color: []uint32{0x4488ccff, 0x4488ccff},
})
```

Start/End lengths must match, with 1–262,144 segments. `Width` is in render pixels,
1–32; zero defaults to 1. Segments use butt caps and independent ends, without
polyline joins or dashes. `Color`, `ColorValue`, and `Labels` are optional columns
with one entry per segment. Without packed `Color`, the palette maps `ColorValue`
or the midpoint Z of each segment. Existing custom `Geometry.Lines` keeps its
endpoint-pair format; set `Geometry.LineWidth` to change its width.

`SetVectorField` creates arrows from paired world positions and vectors:

```go
cmd = chart.SetVectorField("velocity", ntcharts3d.VectorField{
    Origins: positions, // []nt3dmath.Vec3
    Vectors: velocities, // []nt3dmath.Vec3
    Scale:   0.25,
    Width:   2,
    HeadSize: 9,
    Normalize: false,
})
```

Origins/Vectors must have equal lengths, with 1–100,000 entries. The tip is
`origin + vector * Scale`; zero Scale defaults to 1, and negative/nonfinite scales
are invalid. `Normalize: true` makes all nonzero vectors `Scale` data units long.
The palette maps the **original vector magnitude**, so normalization changes
length but preserves the color meaning. Optional `ColorValue` overrides that
scalar; packed `Color` bypasses palette mapping and its legend. Optional `Labels`
identify vectors during picking. Zero vectors contribute to the domain and data
bounds but produce no arrow. All setter slices are copied, and invalid input
leaves the previous series intact. Both series support shared/fixed color domains.

Arrows are unlit, camera-facing shafts and filled triangular heads anchored to
3D endpoints, not cylindrical/conical meshes. Width defaults to 2 pixels (1–32),
and HeadSize defaults to 8 pixels (1–64). A head shrinks to at most 45% of the
projected segment length; its full base width is the larger of its length and
1.5 times the shaft width. View-aligned arrows appear as width-sized squares.
Widths are measured at the renderer's resolution, so software/glyph upscaling
can make them look thicker than native GPU output.

Custom series can populate `Geometry.Arrows` with `Arrow{Start, End, Width,
HeadSize, Color, Datum}`. Custom Width zero means 1 pixel; custom HeadSize zero
means a shaft without a head. Arrow instances are uploaded once per geometry
revision and expanded on the GPU. Orbit, projection, and viewport changes do not
rebuild or upload instance data. Software renders the same shapes; wireframe
shows centerlines and head outlines with single-rune strokes.

Fixed plot ranges clip centerlines in data space. Camera near/far planes are
also clipped before perspective division. A clipped tip loses its arrowhead;
an unclipped tip retains it. Pixel widths may extend across the plot boundary.
Picking follows visible strokes, returning the original segment index or arrow
Datum and a position along the stroke. Above `MaxPickPoints` combined points,
segments, and arrows, those primitives are excluded from picking; bars and mesh
vertices remain eligible.

See [the vector-field example](examples/gallery/vector_field.go): gallery tab 5
combines a swirling flow with analytic streamlines and magnitude coloring.

## Textured terrain and flat maps

`Surface.Material` and `Geometry.Material` apply to indexed triangles. A texture
replaces vertex/palette colors. `Material.Unlit` bypasses lighting, preserving map
colors and labels; otherwise the scene light shades the texture. Points, lines, arrows,
and boxes retain their existing appearance.

```go
texture, err := ntcharts3d.NewTexture(mapImage)
if err != nil {
    return err
}
cmd := chart.SetSurface("terrain", ntcharts3d.Surface{
    NX: nx, NY: ny, Z: elevations,
    MinX: west, MaxX: east, MinY: south, MaxY: north,
    Material: &ntcharts3d.Material{Texture: texture, Unlit: true},
})
// Execute cmd when updating a running Bubble Tea model.
```

Use zero elevations for a tilted flat map. Projection, tile fetching, imagery
composition, and elevation sourcing belong to the caller.

`UV{U, V}` uses normalized image coordinates with `(0,0)` at the top-left.
Surface defaults place that corner at `(MinX, MaxY)` and `(1,1)` at
`(MaxX, MinY)`. Rows still advance from MinY toward MaxY. Optional `Surface.UV`
values override this mapping, one per grid vertex, for crops or atlases. Custom
`Geometry` requires one finite UV per vertex when textured. Coordinates outside
`[0,1]` clamp to the image edge. One material is supported per geometry.

`NewTexture(image.Image)` copies pixels into immutable storage, accepts nonzero
image bounds, and rejects empty images, dimensions above 8192, or any nonopaque
pixel. Callers may mutate or reuse the original image afterward. Surface setters
also copy UVs and materials. A zero-value `Texture` is invalid. Textures can be
shared by multiple series and charts; each renderer owns and releases its GPU
resources through `Close`, and discards images no longer used by its meshes.

Both raster renderers use bilinear filtering with clamp-to-edge and
perspective-correct UV interpolation. Plot clipping interpolates UVs. Channels
are assumed sRGB-encoded, uploaded as RGBA8Unorm, and filtered/lit directly in
encoded space; no linear-light conversion or color-profile processing occurs.
This preserves unlit image colors. Alpha blending and mipmaps are not supported
in this version. Highly tilted or minified images may alias. Wireframe displays
geometry only.

Textured surfaces do not contribute an elevation palette domain or legend by
default. Set `Surface.ColorLegend: true` to opt in; custom series control this
with `Geometry.HasColorRange`.

```go
// Replace imagery without resampling elevations, clipping, or mesh uploads.
cmd = chart.SetSeriesTexture("terrain", nextTexture)
```

`SetSeriesTexture` requires an existing textured series and a valid nonnil
texture. It preserves UVs and lighting settings, and the replacement survives
palette/domain changes. Unknown names and invalid inputs leave the series intact
and set `Err()`. Use `SetSurface` or `SetSeries` to attach/remove a material or
change UVs. Image updates advance `Frame.TextureRevision`, independently of
`Frame.Revision`. Orbiting uploads neither image pixels nor series geometry;
unchanged shared images remain resident across geometry rebuilds.

## Rendering and scheduling

GPU resources persist across frames. Data and palette changes increment the
geometry revision and trigger uploads. Fixed axis ranges and plot proportions
also prepare new geometry. Camera and light changes update uniforms; grid lines,
axis labels, and hover highlights update separately. Resizing replaces the render
attachments and readback buffer. These changes do not reupload series geometry.

Only one frame is rendered and presented at a time. Changes during a frame are
combined into the next one. Idle charts do not run a ticker. Auto-rotation pauses
for two seconds after input; `Camera.ResumeAfter` sets that delay in seconds.

A custom `Renderer` receives a read-only `Frame` and returns an `image.Image`.
`Frame.Axes` contains formatted ticks and names in X/Y/Z order; `Frame.Bounds`
provides the resolved plot bounds. `Frame.Geometry` is clipped to fixed ranges,
and mesh normals account for plot proportions. `Frame.GridLines` contains pairs
of back-plane endpoints; `Frame.Emphasis` describes the optional hover highlight.
These decorations are separate from `Frame.Geometry` and can change without
changing `Frame.Revision`.
It must serialize `Render` and `Close` and reuse geometry while `Revision` is
unchanged. `Frame.TextureRevision` changes independently when imagery is replaced;
renderers must refresh material bindings while retaining mesh buffers. `Texture`
implements read-only `image.Image` for custom renderers. The default renderer adapts chart frames to nt3d, which owns GPU execution. WASM
uploads and readback copy data across JavaScript calls.

Presentation uses complete frames read back into Go memory. `Geometry.Lines`
supports pairs of vertices with `Geometry.LineWidth` in render pixels.
`Geometry.Arrows` holds compact arrow instances; both are unlit and opaque.

## Snapshots

`Snapshot` draws the chart as it stands into an image of a chosen size, without
a terminal. The program need not be running, and the chart's size in cells
plays no part: the picture's own shape sets the camera's aspect.

```go
chart := ntcharts3d.New(1, 3, ntcharts3d.WithBackground(color.White))
defer chart.Close()
chart.SetSeries(mesh)
chart.SetCamera(ntcharts3d.Camera{Alpha: 30, Beta: 45, Distance: 3})
img, mode, err := chart.Snapshot(1600, 1200)
```

The chart is drawn in its requested render mode, and `mode` is the one that
drew it. WebGPU falls back to software when the GPU is unavailable or fails.
That fallback is the snapshot's alone; it does not change how the chart is
drawn on screen, nor `Err()`. A caller with a renderer of its own can compare
`mode` with `WebGPU` and use that instead.

A snapshot is for keeping, so a software snapshot draws every point, triangle,
bar, line, and arrow; terminal frames may sample points, lines, and arrows.
Terminal raster sizing and adaptive resolution do not apply. The area may not exceed
4096×2160 pixels. Wireframe draws text and has no image; `Snapshot` returns an
error for it, as it does for a closed chart.

The image holds what the renderer draws, axes and image legends among them.
Hide them with `SetAxes` and `SetColorLegendVisible` for a picture of the data
alone. Views of the same data reuse its geometry on the GPU: only the first
uploads it.

`Snapshot` returns when the image is drawn. Like the chart's other methods, it
must not be called while another goroutine changes the chart.

## Limits and fallback

| Render mode | Limits |
| --- | --- |
| WebGPU | 500,000 scatter points total; `WithMaxPoints` can lower this. Surface dimensions 2–512; bar grids up to 512×512. Custom meshes up to `nt3d.MaxMeshTriangles` (932,067) per series, with vertices shared or not. Up to 100,000 arrows and 262,144 line segments per series. Framebuffer area capped at 4096×2160 pixels. |
| Software | Full Kitty raster while prompt, adaptively reduced toward a 320×200 bounding box. All triangles and bars; interactive point, line, and arrow sampling is owned by nt3d. Glyph raster is one pixel wide and two pixels high per cell. |
| Wireframe | nt3d terminal wireframe with depth-aware geometry, chart axis/hover strokes, and cell labels. |

The mesh limit is what fits the largest storage buffer binding that WebGPU
promises, 128 MiB, at 48 bytes for each index. Native adapters often allow
more, and a browser may allow less for several large series together, which
consume device memory across separate geometry buffers.

GPU initialization or rendering failure switches to software. A CPU adapter
also selects software. Software draws at full size while it is prompt. A frame
that takes over 60 ms is followed by one reduced to take about 40 ms, and one
under 20 ms by one half as large again, drawn at once, so that a chart at rest
comes to its full size. Three consecutive frames over 150 ms at the smallest
size switch to wireframe. `WithRenderMode` and `SetRenderMode` select a mode explicitly;
`SetRenderMode(WebGPU)` retries GPU initialization after fallback.

Glyph output uses exactly one pixel wide and two pixels high per terminal cell,
including when WebGPU is active. Projection uses the physical display aspect,
not the glyph raster aspect. Software surfaces retain all triangles and nt3d
clips triangles crossing the near plane.

Point, line, and arrow picking is disabled above 200,000 combined scatter points,
line segments, and arrows. Bars and surface vertices remain pickable.

## Build and test

The library and examples use the upstream ntcharts commit pinned in their
`go.mod` files. See the [README](README.md#dependencies) for dependencies.
`task ci` checks all three modules and builds both examples. `task test-gpu`
runs hardware tests. The equivalent library commands are:

```sh
go test -race ./...
go vet ./...
go test -tags webgpu . -run TestGPU -count=1 -v
go test -tags webgpu . -run '^$' -bench BenchmarkScatterGPU -benchmem
```

Normal tests need no GPU. Tests and benchmarks tagged `webgpu` require a
hardware adapter. Do not use `-race` with native GPU tests across the FFI boundary.

See the [gallery](examples/gallery/README.md) for native and browser commands,
and the [Go/TinyGo demo site](web/README.md) for GitHub Pages builds.

### Kitty frame encoding

Direct Kitty frames default to `picture.KittyFormatZlib`, which avoids PNG's
per-row filter trials during interactive camera movement. `WithKittyFormat`
selects `KittyFormatPNG` or `KittyFormatRGBA` instead. With
`WithKittyMedium(picture.KittyMediumSharedMemory)`, supported terminals receive
raw pixels through shared buffers; the selected encoding is the fallback when
shared memory is unavailable.

## Terminal session and cleanup

Transport types come from `nimbleterminal.dev/ntpicture` (usually imported as
`picture`). `WithTerminalSession(session)` shares discovery and cell geometry
between charts on one terminal connection. The host owns supplied sessions;
otherwise the chart creates and closes its own explicit session.

`WithImageIDAllocator(func() (int, error))` replaces `WithKittyID`. Allocations
must return fresh positive 24-bit IDs. By default ntpicture's shared allocator
coordinates placements; streaming frames reuse their placement internally.

Execute every command returned by setters and `Update`, including resize and
legend changes. `PictureStatus()` exposes presentation generations, busy/error
state, and actual encoding/medium. Presented means ordered output completion,
not an acknowledgement from the terminal display.

Quit from a parent model with `tea.Sequence(chart.Dispose(), tea.Quit)` and
retain `defer chart.Close()` around the event loop. Continue forwarding late
messages to `Update` after disposal so ntpicture can retire pending work.
Dispose all charts before closing a host-owned terminal session.

The chart module path is unchanged. `Frame`, custom `Renderer`, chart clipping,
and picking semantics remain here. Math types come directly from `nt3d/math`;
update old `math3d` imports and qualifiers to `nt3dmath`. Chart primitive types
remain aliases to nt3d; use keyed primitive literals. The chart's
opaque `NewTexture` contract is retained over immutable engine texture storage.

Geometry limits are provided directly by `nimbleterminal.dev/nt3d`: use
`nt3d.MaxPoints`, `nt3d.MaxVectors`, `nt3d.MaxMeshTriangles`, and
`nt3d.MaxMeshIndices`. The corresponding ntcharts3d constants have been removed.
