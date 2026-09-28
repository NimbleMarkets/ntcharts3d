# Transport diagnostic

Uses released `go-gpuimage v0.1.0` and ntcharts transport commit `4777228`,
pinned as `v2.3.1-0.20260927013859-4777228ed363`. No local module replacements
are needed for these libraries. The existing Bubble Tea browser fork remains.

This standalone diagnostic measures GPU rendering/readback separately from
terminal transport. It does not import the ntcharts3d chart implementation,
so it can provide a baseline alongside the gallery while chart code changes.

A 3D point cloud (4 000 points on a sphere, depth-tested, instanced quads)
rendered by the GPU through [`go-gpuimage`](https://github.com/NimbleMarkets/go-gpuimage) and shown inside
a Bubble Tea layout by ntcharts `picture.Model`. The same source runs
natively in a Kitty-capable terminal and in the browser under go-booba.

The header reports the image path (`png`, `rgba`, or `shm` for Kitty frames,
`glyph` for the half-block fallback), application FPS, raster size, **R** (GPU render +
readback, `Stats.Total`) and **E** (picture preparation and encoding) in ms, and the Kitty
APC payload size; the second line breaks R into `draw / submit / map / copy`.
Application FPS measures frame submission, not the terminal's display refresh.

## Run

From the repository root, use `task transport -- <flags>` to run this example.
`task build-web` prepares both browser examples; `task serve-transport` serves
this one. go-booba v0.7.0 bundles the required terminal assets.

This example is its own Go module. The go-booba pin sets
`TERM_PROGRAM=ghostty` so picture's Kitty probe runs in the browser.

```sh
cd ~/projects/ntcharts3d/examples/transport
go run .                                  # native
CGO_ENABLED=1 go run . -medium shm         # local macOS Kitty/Ghostty
CGO_ENABLED=1 go run . -medium shm -fps 120 # higher-refresh displays
go run . -duration=15s -report=/tmp/r.json
```

Browser:

```sh
GOOS=js GOARCH=wasm go build -o web/app.wasm .
go tool booba-assets web/
python3 -m http.server -d web 8765   # then open http://localhost:8765/
```

The browser build needs the pinned Bubble Tea fork `replace` already in
`go.mod` from go-booba v0.7.0.

Refresh the page after rebuilding, then press `m` to enable shared memory.
The header should show `shm`; `m` again returns to the selected direct format.
`-medium direct|shm` selects the initial medium, and `-format png|rgba` selects
the direct format. Native shared memory works on macOS with CGO and on Linux
with a local Kitty-capable terminal that supports `t=s`. Use direct transmission
over SSH. Unsupported builds, older browser terminals, and allocation failures
fall back to direct transmission. The header always reports the actual path. `go tool booba-assets web/` installs
the matching JavaScript and WASM files; no local overlay is needed.

### Over SSH

Run the native binary on the remote GPU host from a local Ghostty/Kitty terminal.
Allocate a PTY and explicitly enable Kitty graphics, since SSH may not forward
`TERM_PROGRAM`:

```sh
ssh -t HOST 'NTCHARTS_KITTY=supported /path/to/transport -medium direct -format png -fps 120'
```

Use `t` to compare PNG and raw RGBA, and `,`/`.` to adjust density. Keep shared
memory off (`m`): the remote process and local terminal have separate memory
namespaces. PNG generally requires much less SSH bandwidth than raw RGBA.

On dwarfspark, the isolated experiment is at
`~/projects/wgpu-shm-20260925-TYtYxO`. Its `run-ssh.sh` launcher selects the NVIDIA
Vulkan driver and sets up direct Kitty transmission:

```sh
ssh -t dwarfspark '~/projects/wgpu-shm-20260925-TYtYxO/run-ssh.sh -fps 120'
```

## Controls

| Key | Action |
| --- | --- |
| ← → ↑ ↓ | Orbit |
| + / − | Zoom |
| Space | Pause / resume auto-rotation |
| f / Esc | Fullscreen / back |
| t | Toggle direct PNG / RGBA format |
| m | Toggle direct / shared-memory transport |
| g | Toggle glyph rendering |
| , / . | Decrease / increase pixel density |
| r | Reset the camera |
| q | Quit |

## Historical measurements (before relocation)

Apple M3 Max, gogpu/wgpu v0.34.5, 4 000 points, `-fps 60`. Values are the
smoothed per-frame averages from `-report` or the header.

| Environment | Raster | Path | app fps | R total | draw | submit | map | copy | E |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Ghostty 1.3.1, Metal | 1428×752 | Kitty png | 56 | 1.20 ms | 0.10 | 0.04 | 0.80 | 0.23 | 16.34 |
| Ghostty 1.3.1, Metal | 1428×752 | Kitty shm | 61 | 1.94 ms | 0.18 | 0.08 | 1.19 | 0.39 | 1.07 |
| Chrome headless (WebGPU), local forks | 1170×624 | Kitty png | 14–16 | 24–30 ms | 0.15–0.31 | 0.05–0.06 | 21.6–27.8 | 1.7–1.8 | 43–44 |
| Chrome headless (WebGPU), local forks | 1170×624 | Kitty shm | 51–58 | 3.6–4.4 ms | 0.12–0.20 | 0.03–0.04 | 3.0–3.6 | 0.26–0.31 | 1.0–1.1 |

The paired headless rows were measured on 2026-09-25 in one Chrome session
at a 1200×700 viewport, density 16, toggling `m` without changing geometry.
The terminal APC fell from about 123 KB to 96 bytes; the 2.9 MB of RGBA
pixels travel through the registry instead. These are automated Chrome
measurements, not Safari results. The shared-memory path also skips the
explicit event-loop sleeps used around the more expensive direct encoding.

The native Ghostty 1.3.1 rows are final smoothed values from consecutive
6-second PNG and 8-second shared-memory runs on 2026-09-25, with the same
window and density 16. The target rate was 60 fps. A separate real-terminal
test confirmed that Ghostty read and unlinked a POSIX shared-memory object.

An eight-second SSH run from dwarfspark (Linux ARM64, NVIDIA GB10) to Ghostty
on this Mac rendered 655 frames at a 700×576 raster, targeting 120 fps. Final
smoothed values were 99 application fps, R=0.86 ms, and E=8.36 ms with direct
PNG. This measures application submission, not the display's refresh rate.
Linux shared-memory lifecycle tests also passed with the race detector and a
separate reader process; shared memory was not used for SSH display.

The subsequent review removed an intermediate RGBA allocation and copy.
The native `BenchmarkKittySharedMemory` at 1280×960 measured about 1.93 ms
before and 1.73 ms after, with Go heap allocation falling from 4.9 MB to
about 1.5 KB per frame. This benchmark includes object creation, pixel filling,
APC construction, and unlink; it excludes GPU work and terminal rendering.
The shared-memory object itself still holds 4.9 MB of pixels.

Native readback is cheap: `map` (GPU sync plus staging map) is about 1 ms
and dominates R. In the browser two things had to change before the same
loop ran at all:

- Chrome only settles `mapAsync` when the page ticks. Without an
  animation-frame loop `map` averaged 93 ms; `go-gpuimage` now pumps
  `requestAnimationFrame` while a map is pending, which brought it to
  about 25 ms. The remainder is main-thread contention with the PNG encode
  (`E`) and ghostty-web's decode, since everything shares one thread.
- ghostty-web's renderer has a frame-skip gate keyed on a Kitty placement
  signature (image id, geometry, wasm data address). A same-size image
  re-transmitted under the same id lands at the same address, so the gate
  dropped every frame and the sphere only repainted where a drag dirtied
  cells. The fix (renderer invalidation on any written chunk carrying
  `ESC _ G`) is in the ghostty-web fork; go-booba needs a release with it.

Shared memory removes PNG encoding and decoding from that browser frame budget.

## Remaining integration

- Replace the ntcharts pseudo-version with a release tag once available.
- Verify shared-memory display in a local Linux Kitty-capable terminal; Linux lifecycle tests now pass on dwarfspark.

Native shared memory remains opt-in. Automatically selecting it would require
a terminal capability probe; OS support alone is insufficient, especially over SSH.

## Tests

```sh
go test ./...                          # camera, points, model
NTCHARTS3D_TRANSPORT_GPU_TEST=1 go test ./...    # + real GPU scene test; never with -race
```

## Relocation

Moved from booba-shim `examples/wgpu-picture` at commit `b033365`. Source,
shader, tests, diagnostics, and historical measurements are preserved.
Generated browser assets are not copied; regenerate them with the instructions
above. Native and browser rendering behavior is unchanged by the module rename.
