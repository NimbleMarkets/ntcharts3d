# ntcharts3d gallery

The gallery runs natively or in a go-booba browser terminal. It has five tabs:

- Scatter: a 16,000-point noisy sphere.
- Surface: a 96×96 Perlin grid with a Height (µm) axis and fixed −1…1 range.
- Bars: a 12×7 hour/day chart with fixed 0…4 limits and X:Y:Z proportions of 1:.7:.6.
- Map: a textured island with street/topographic imagery, terrain/flat views, and optional lighting.
- Vectors: a swirling 3D flow with 147 arrows colored by speed and six analytic streamlines.

Data is generated locally. The first three tabs have a fixed color domain and legend.
The map uses image colors and omits the elevation color legend.
Back-plane grids and in-chart hover labels are enabled.
Axes are labeled inside the chart. The bar tab uses Hour, Day, and Value, with
hour and weekday categories shared by tick labels and pick descriptions.
The numeric range caption outside the chart is provided by this example.
See the [dependency pins](../../README.md#dependencies).

From the repository root, use `task gallery -- <flags>` to run the gallery.
`task build-web` prepares both browser examples; `task serve-gallery` serves
this one. go-booba v0.7.0 bundles the required terminal assets.

## Native

Run from this directory:

```sh
go run .
go run . -medium shm             # compatible local Kitty/Ghostty terminal
go run . -render-mode software
go run . -render-mode wireframe
go run . -points 500000 -rotate=false
```

Press 1/2/3/4/5 or Tab to change tabs. Drag to orbit, Shift-drag to pan, and scroll
to zoom. Press `b` to toggle grid lines, `v` for legend mode, `g` for Kitty/glyph output, `o` for projection,
and `q` to quit. Hover shows coordinates and labels; clicks show the series name
and datum in the header. See [all chart controls](../../API.md#interaction-and-embedding).

`-tab 2` starts on the surface. `-duration 10s` exits after ten seconds.

## Textured map demo

From the repository root:

```sh
go -C examples/gallery run . -tab 4 -rotate=false
# Compare the software fallback:
go -C examples/gallery run . -tab 4 -rotate=false -render-mode software
```

Use a Kitty graphics-compatible terminal for image detail, or run the browser
gallery and press `4`. Drag to tilt/orbit and scroll to zoom.

- `t`: swap street/topographic imagery using `SetSeriesTexture`, retaining the mesh.
- `f`: rebuild the surface with flat/terrain elevations.
- `l`: toggle unlit colors/terrain lighting.
- `o`: switch perspective/orthographic projection.
- `r`: toggle auto-rotation.

The 768×576 map and 96×72 elevation grid are generated locally for the fictional
Cedar Island. North is at the image top; the north marker and harbor labels make
orientation easy to check. This is synthetic imagery, not fetched OSM data.
See [terrain.go](terrain.go) for the complete textured surface example. A real
composited map image can replace `terrainImage`; projection and tile fetching
remain outside this library.

## Vector field demo

From the repository root:

```sh
go -C examples/gallery run . -tab 5 -rotate=false
# Compare raster fallback:
go -C examples/gallery run . -tab 5 -rotate=false -render-mode software
```

- `n`: switch magnitude-scaled/equal-length arrows; color always indicates speed.
- `s`: show/hide the streamlines.
- `w`: switch thin/wide shafts and lines.
- Drag to orbit, scroll to zoom, `o` for projection, and `r` for auto-rotation.

Arrowheads face the camera. Hover or click a stroke to inspect its vector or
streamline label. The gray helices are analytic streamlines of the same vector
field. The example uses `SetVectorField` and `SetLines`; see
[vector_field.go](vector_field.go). It works in the browser gallery under tab `5`.

## Browser

For the stock Go/TinyGo comparison page, use `task build-wasm-site` and
`task serve-wasm-site` from the root. See [demo site instructions](../../web/README.md).
Both variants use Kitty shared memory. The commands below build and serve the
standalone gallery with the same bundled terminal assets.

```sh
GOOS=js GOARCH=wasm go build -o web/app.wasm .
go tool booba-assets web/
python3 -m http.server 8766 --bind 127.0.0.1 -d web
```

Open http://127.0.0.1:8766. This build uses go-booba v0.7.0 and its pinned
Bubble Tea WASM fork. The header shows `Kitty/shm` when shared memory is active.

## Rendering and transport

`-medium auto` uses PNG natively. In the browser, it requests shared memory and
falls back to PNG when the terminal registry is unavailable. Native shared
memory requires a shared local namespace; use `-medium direct` over SSH.
Embedded charts select transport with `ntcharts3d.WithKittyMedium`.

If WebGPU is unavailable, the chart falls back to software. Kitty/glyph output
is independent of that choice. See the [API limits](../../API.md#limits-and-fallback)
and [demo site build instructions](../../web/README.md).
