# Demo site

The site runs the gallery with stock Go or TinyGo. Each build has its own
`wasm_exec.js`; the compiler selector loads one gallery at a time.

Install [TinyGo 0.42.0](https://tinygo.org/getting-started/install/), then run
from the repository root:

```sh
task build-wasm-site
task serve-wasm-site
```

Open http://localhost:8768. To use a TinyGo executable outside PATH:

```sh
task build-wasm-site TINYGO=/path/to/tinygo
```

The build task selects Go 1.26.8 for both compilers. TinyGo 0.42.0 hit
standard-library map errors with Go 1.27.1 in this project. The WASM gallery
uses `tea.WithoutSignalHandler()` for TinyGo's browser signal limitation.
The TinyGo build also selects `-gc=boehm`: its default collector stalled in
`gcBlock.findHead` during gallery rendering. Disabling signals alone did not
prevent those stalls.

The builder writes `web/dist/`, including compiler versions and binary sizes.
HTML, styles, and scripts in `web/` are source files. Generated files are ignored
by Git. Relative asset URLs work at `/` and under `/ntcharts3d/`.

The site uses Kitty shared memory and the terminal assets bundled with go-booba
v0.7.0. Terminal initialization creates the image registry before Go starts.
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

Go 1.26.8 and TinyGo 0.42.0 both rendered with Apple WebGPU in Chrome under
the `/ntcharts3d/` prefix. Browser checks confirmed shared-memory commands,
image registry reads, and cleanup with no browser errors.
With Boehm, TinyGo displayed 144 shared-memory frames in a 10-second browser
check and responded to chart switching, projection changes, and resizing.

`task ci`, `task test-gpu`, and `actionlint` passed. Hosted GitHub Actions and
Pages deployment have not run yet.
