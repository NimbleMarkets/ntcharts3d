# ntcharts3d gallery

The gallery runs natively or in a go-booba browser terminal. It has three tabs:

- Scatter: a 16,000-point noisy sphere.
- Surface: a 96×96 Perlin grid with a Height (µm) axis and fixed −1…1 range.
- Bars: a 12×7 hour/day chart with fixed 0…4 limits and X:Y:Z proportions of 1:.7:.6.

Data is generated locally. Each tab has a fixed color domain and legend.
Back-plane grids and in-chart hover labels are enabled.
Axes are labeled inside the chart. The bar tab uses Hour, Day, and Value, with
hour and weekday categories shared by tick labels and pick descriptions.
The numeric range caption outside the chart is provided by this example.
See the [dependency pins](../../README.md#dependencies).

From the repository root, use `task gallery -- <flags>` to run the gallery.
`task build-web` prepares both browser examples; `task serve-gallery` serves
this one. Apply the browser asset overlay below after generating assets.

## Native

Run from this directory:

```sh
go run .
go run . -medium shm             # compatible local Kitty/Ghostty terminal
go run . -render-mode software
go run . -render-mode wireframe
go run . -points 500000 -rotate=false
```

Press 1/2/3 or Tab to change tabs. Drag to orbit, Shift-drag to pan, and scroll
to zoom. Press `b` to toggle grid lines, `v` for legend mode, `g` for Kitty/glyph output, `o` for projection,
and `q` to quit. Hover shows coordinates and labels; clicks show the series name
and datum in the header. See [all chart controls](../../API.md#interaction-and-embedding).

`-tab 2` starts on the surface. `-duration 10s` exits after ten seconds.

## Browser

```sh
GOOS=js GOARCH=wasm go build -o web/app.wasm .
go tool booba-assets web/
cp ~/projects/ghostty-web-kittyfix/dist/*.js web/ghostty-web/
cp ~/projects/ghostty-web-kittyfix/ghostty-vt.wasm web/ghostty-web/
python3 -m http.server 8766 --bind 127.0.0.1 -d web
```

Open http://127.0.0.1:8766. This build uses the pinned Bubble Tea WASM fork.
The copy commands install the local ghostty-web assets needed for shared memory;
those assets are not distributed with ntcharts3d.

## Rendering and transport

`-medium auto` uses PNG natively. In the browser, it requests shared memory and
falls back to PNG when the terminal registry is unavailable. Native shared
memory requires a shared local namespace; use `-medium direct` over SSH.
Embedded charts select transport with `ntcharts3d.WithKittyMedium`.

If WebGPU is unavailable, the chart falls back to software. Kitty/glyph output
is independent of that choice. See the [API limits](../../API.md#limits-and-fallback)
and [current validation results](../../STATUS.md#validation).
