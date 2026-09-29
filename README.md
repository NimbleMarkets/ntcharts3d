# ntcharts3d

3D charts for Bubble Tea terminals and go-booba browser applications.

```go
import "github.com/NimbleMarkets/ntcharts3d"
```

One `ntcharts3d.Model` combines named scatter, surface, bar, line, vector-field, or custom series
with a shared camera. Charts support orbit, pan, zoom, picking, color legends,
hover highlights, fixed axis and color ranges, plot proportions, and in-chart
axis labels with optional back-plane grids.
Rendering uses go-gpuimage v0.1.0, with software and wireframe fallbacks.
ntcharts handles terminal images and glyph output.

The initial release is **v0.1.0**. Install it with:

```sh
go get github.com/NimbleMarkets/ntcharts3d@v0.1.0
```

See the [changelog](CHANGELOG.md), [API guide](API.md),
[gallery instructions](examples/gallery/README.md),
[transport diagnostic](examples/transport/README.md), and [demo site](web/README.md).

## Run locally

```sh
go test -race ./...
cd examples/gallery
go run .
# Shared memory requires a compatible local terminal:
go run . -medium shm
```

The gallery has scatter, surface, bar, textured map, and vector-field tabs.
Run `go -C examples/gallery run . -tab 4 -rotate=false` from the root to try the
map: `t` swaps imagery, `f` toggles flat/terrain, and `l` toggles lighting. Press `v` to switch legend modes,
`g` to switch Kitty/glyph output, `o` to switch projection, and `b` for grid lines. Use PNG over SSH.
Run with `-tab 5 -rotate=false` for swirling arrows and streamlines: `n` normalizes
lengths, `s` toggles streamlines, and `w` changes stroke widths.
See the gallery instructions for browser builds and all controls.

## Development tasks

Install [Task](https://taskfile.dev/docs/installation), then run tasks from the
repository root. `task` builds the library and native examples into `bin/`.

| Command | Purpose |
| --- | --- |
| `task list` | List all tasks |
| `task ci` | Check formatting, modules, race tests, vet, and native/WASM builds |
| `task test-gpu` | Run hardware GPU tests without the race detector |
| `task gallery -- -render-mode software` | Run the gallery with flags |
| `task transport -- -medium shm` | Run the transport diagnostic with flags |
| `task go-tidy` | Update module files in all three modules |
| `task build-web` | Compile both WASM examples and generate browser assets |
| `task serve-gallery` / `task serve-transport` | Serve generated browser examples |
| `task build-wasm-site` / `task serve-wasm-site` | Build and serve the Go/TinyGo demo page |
| `task clean` | Remove compiled outputs, keeping browser assets |

Builds and CI check module files without updating them.
`task build-web` generates the browser assets bundled with go-booba v0.7.0,
including Kitty shared-memory support. No local terminal asset overlay is needed.

GitHub Actions runs `task ci` on Linux and macOS for pushes and pull requests.
Hardware GPU tests run separately with `task test-gpu`. WASM compilation does
not verify browser rendering. A separate Pages workflow builds the gallery with
Go and TinyGo and deploys from `main`. See the [demo site instructions](web/README.md).
The Pages gallery uses Kitty shared memory with the bundled terminal assets.

## Dependencies

The library requires Go 1.26; the examples require Go 1.26.8. The library and
both examples pin ntcharts to upstream
commit [`4777228ed363`](https://github.com/NimbleMarkets/ntcharts/commit/4777228ed363ab15755715840089133a1a362c83),
which includes the Kitty transport APIs. Go records this unreleased commit as
`v2.3.1-0.20260927013859-4777228ed363`. No local ntcharts checkout is needed.

The gallery's `ntcharts3d => ../..` replacement uses this repository directly.
Both examples use go-booba v0.7.0 and its Bubble Tea WASM fork at `1b36865b418a`.
The bundled ghostty-web assets support browser shared memory.

## Ownership

- ntcharts3d: chart geometry, interactions, legends, and rendering.
- go-gpuimage: GPU execution, textures, and readback.
- ntcharts: terminal image transport and canvas.
- go-booba and the browser terminal: WASM hosting and Kitty image display.

Extracted from ntcharts `wgpu-shared-memory` at `2ed8872`.
MIT license; existing copyright notices are preserved.
