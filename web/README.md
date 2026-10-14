# Demo site

The site runs the gallery with stock Go or TinyGo. Each build has its own
`wasm_exec.js`; the compiler selector loads one gallery at a time.

Install [TinyGo 0.43.0](https://tinygo.org/getting-started/install/), then run
from the repository root:

```sh
task deps
task build-wasm-site
task serve-wasm-site
```

Open http://localhost:8768. To use a TinyGo executable outside PATH:

```sh
task build-wasm-site TINYGO=/path/to/tinygo
```

The build task selects Go 1.26.8 for both compilers. With Go 1.27.1, TinyGo
0.43.0 fails to compile `hash/maphash`, which ultraviolet imports: Go 1.27's
map runtime uses `internal/abi` names that TinyGo's copy lacks (tinygo#5684).
The WASM gallery
uses `tea.WithoutSignalHandler()` for TinyGo's browser signal limitation.
The TinyGo build also selects `-gc=boehm`: its default collector stalled in
`gcBlock.findHead` during gallery rendering. Disabling signals alone did not
prevent those stalls.

The builder writes `web/dist/`, including compiler versions and binary sizes.
HTML, styles, and scripts in `web/` are source files. Generated files are ignored
by Git. Relative asset URLs work at `/` and under `/ntcharts3d/`.

The site uses Kitty shared memory and the terminal assets bundled with go-booba
v0.9.2. Terminal initialization creates the image registry before Go starts.
The chart header reports `Kitty/shm` when frames use shared memory. No local
ghostty-web checkout or asset overlay is needed.

## GitHub Pages

The Pages workflow builds both variants on pull requests and pushes to `main`.
Only `main` deploys. In repository **Settings → Pages**, choose **GitHub Actions**
as the build source. After a successful deployment, the site is available at:

https://nimblemarkets.github.io/ntcharts3d/

`task ci` checks the Go builder along with the library. The separate Pages
workflow verifies the complete TinyGo build and uploads `web/dist/` as its Pages
artifact. Browser checks are still needed when compiler or terminal pins change.

## Local validation

On 2026-10-14, the nt3d/ntpicture migration passed `task ci`, `task test-gpu`
(Apple M5 Max), `task build-wasm-site`, and `actionlint`. In Chrome, Go 1.26.8
and TinyGo 0.43.0 rendered with WebGPU and Kitty shared memory. The smoke check
covered scatter/surface/bar/vector chart switching, terrain texture replacement,
glyph fallback, and returning to Kitty with the text legend's narrower viewport.
These are local results; hosted CI still requires the pinned nt3d commit to be
published upstream. Earlier release validation follows.


Go 1.26.8 and TinyGo 0.42.0 both rendered with Apple WebGPU in Chrome under
the `/ntcharts3d/` prefix. Browser checks confirmed shared-memory commands,
image registry reads, and cleanup with no browser errors.
With Boehm, TinyGo displayed 144 shared-memory frames in a 10-second browser
check and responded to chart switching, projection changes, and resizing.

TinyGo 0.43.0 with Boehm rendered every gallery chart with WebGPU and
`Kitty/shm` in headless Chrome, switching charts and projection over a
20-second check with no console errors. Chart switches log a few ghostty
`kitty_gfx` response warnings with both compilers.

`task ci`, `task test-gpu`, and `actionlint` passed. Hosted GitHub Actions and
Pages deployment passed for v0.2.2.

On 2026-10-12, fresh Go 1.26.8 and TinyGo 0.43.0 (Boehm) builds using
go-booba v0.9.2 were checked in Safari 27.0.1. Both used `WebGPU` and
`Kitty/shm`, animated without input, redrew after drag-orbit, and switched
from scatter to surface without grey placeholder cells. `GOWORK=off task ci`
and `GOWORK=off task build-wasm-site` passed; the generated assets matched
the booba v0.9.2 bundle byte for byte.
