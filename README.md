# ntcharts3d - Nimble Terminal 3D Charts

<p>
    <a href="https://github.com/NimbleMarkets/ntcharts3d/actions/workflows/go.yml"><img src="https://github.com/NimbleMarkets/ntcharts3d/actions/workflows/go.yml/badge.svg" alt="Build Status"></a>
    <a href="https://github.com/NimbleMarkets/ntcharts3d/actions/workflows/pages.yml"><img src="https://github.com/NimbleMarkets/ntcharts3d/actions/workflows/pages.yml/badge.svg" alt="Deploy to GitHub Pages Status"></a>
    <a href="https://github.com/NimbleMarkets/ntcharts3d/tags"><img src="https://img.shields.io/github/tag/NimbleMarkets/ntcharts3d.svg" alt="Latest Release"></a>
    <a href="https://pkg.go.dev/github.com/NimbleMarkets/ntcharts3d?tab=doc"><img src="https://godoc.org/github.com/NimbleMarkets/ntcharts3d?status.svg" alt="GoDoc"></a>
    <a href="https://github.com/NimbleMarkets/ntcharts3d/blob/main/CODE_OF_CONDUCT.md"><img src="https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg"  alt="Code Of Conduct"></a>
</p>

3D charts for Bubble Tea terminals and go-booba browser applications.

<p align="center"><a href="https://nimblemarkets.github.io/ntcharts3d"/><b>Live Demo</b></a> (<a href="./examples/gallery/main.go">source</a>)<br><a href="https://nimblemarkets.github.io/ntcharts3d"/><img src="examples/gallery/demo.gif" alt="gallery demo" width="75%"/></a></p>

```go
import "github.com/NimbleMarkets/ntcharts3d"
```

One `ntcharts3d.Model` combines named scatter, surface, bar, line, vector-field, or custom series
with a shared camera. Charts support orbit, pan, zoom, picking, color legends,
hover highlights, fixed axis and color ranges, plot proportions, and in-chart
axis labels with optional back-plane grids.
Rendering uses `nimbleterminal.dev/nt3d` for WebGPU, software, and wireframe.
`nimbleterminal.dev/ntpicture` owns terminal sessions, image transport, and glyph output.
The chart module remains `github.com/NimbleMarkets/ntcharts3d`.

The latest released chart API before this migration is **v0.2.2**:

```sh
go get github.com/NimbleMarkets/ntcharts3d@v0.2.2
```

See the [changelog](CHANGELOG.md), [API guide](API.md),
[gallery instructions](examples/gallery/README.md),
[transport diagnostic](examples/transport/README.md), and [demo site](web/README.md).

## Run locally

```sh
bash scripts/bootstrap-deps.sh  # extract committed pins from ../nt3d and ../ntpicture
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
| `task deps` | Extract pinned engine/picture sources from sibling repositories |
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
`task build-web` generates the browser assets bundled with go-booba v0.9.2,
including Kitty shared-memory support. No local terminal asset overlay is needed.

GitHub Actions runs `task ci` on Linux and macOS for pushes and pull requests.
Hardware GPU tests run separately with `task test-gpu`. WASM compilation does
not verify browser rendering. A separate Pages workflow builds the gallery with
Go and TinyGo and deploys from `main`. See the [demo site instructions](web/README.md).
The Pages gallery uses Kitty shared memory with the bundled terminal assets.

## Dependencies

The library and examples require Go 1.26.8. This development migration pins
`nt3d` at `972f2a4220a4` and `ntpicture` at `1a8609901740`; `ntcanvas v0.1.0`
and `ntgpuimage v0.2.0` resolve as normal Go modules. NTCharts is no longer a dependency.

Until the engine pin is published, module replacements point into ignored
`.deps/` directories. Run `task deps` (or the shell script above) to extract
**committed** snapshots from sibling repositories. It does not copy working-tree
edits or modify either sibling. `NT3D_SOURCE` and `NTPICTURE_SOURCE` override
those repository paths. CI checks out the same commits into `.deps/`.
The nt3d pin was not available upstream when this migration was validated;
GitHub Actions requires that commit to be published. Before a library release,
publish the dependency pins, remove these local replacements, and regenerate
module checksums in all three modules.

The gallery's `ntcharts3d => ../..` replacement uses this repository directly.
Both examples use go-booba v0.9.2 and its Bubble Tea WASM fork at `c397c32a5074`.
The bundled ghostty-web assets support browser shared memory.

The bundled ghostty-web graphics fixes restore frame updates in Safari and
prevent grey placeholder cells while dragging. Regenerate browser assets
with `task web-assets` after changing the booba pin.

## Ownership

- ntcharts3d: series, chart frames, axes, palettes, clipping, picking, hover, and layout.
- nt3d: geometry validation, shared math and textures, GPU/software/wireframe rendering.
- ntpicture: explicit terminal sessions, encoding, placement IDs, and presentation lifecycle.
- ntcanvas: terminal wireframe surfaces through nt3d.
- ntgpuimage: GPU execution and readback through nt3d (directly in the transport diagnostic).
- go-booba and the browser terminal: WASM hosting and Kitty image display.

The chart uses nt3d's frame/render interfaces directly. Its custom `Renderer`,
`Frame`, picking, and camera controls remain chart APIs; `nt3d/widget` is not
embedded. This preserves data-space surface-vertex picking, bar values, and
chart viewport behavior without coupling the generic widget to chart metadata.

Use `nimbleterminal.dev/nt3d/math` (package `nt3dmath`) directly for `Vec3`,
`Mat4`, `Ray`, and `AABB`; the local `math3d` package has been removed. Chart
primitive types still forward to engine types. Transport options now use
`nimbleterminal.dev/ntpicture`. Replace `WithKittyID` with
`WithImageIDAllocator`; share `WithTerminalSession` between charts on one
connection. Execute `Dispose()` before quitting, then `Close()` after the event
loop exits. See [lifecycle details](API.md#terminal-session-and-cleanup).

Extracted from ntcharts `wgpu-shared-memory` at `2ed8872`.
MIT license; existing copyright notices are preserved.
