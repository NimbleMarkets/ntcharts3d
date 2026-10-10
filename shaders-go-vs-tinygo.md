# Shaders demo: Go vs TinyGo in the browser

Measured 2026-09-29.

TinyGo's frames are 13–18% faster at the median, but its page leaks about one
RGBA frame of memory per frame, which makes it unfit for a page left open.

The shaders demo lives in `~/projects/ntcharts/examples/shaders`, not in
`ntcharts3d`. The work is uncommitted on a new branch `shaders-stage-timings`
in `~/projects/ntcharts`. All tests pass, including the real-GPU test.

## Setup

| | |
| --- | --- |
| Machine | Apple M5 Max, macOS 27.0 |
| Browser | Chrome 154, headless, WebGPU on Metal (not a fallback adapter) |
| Go | 1.26.8 |
| TinyGo | 0.42.0, `-target=wasm -opt=2 -no-debug -gc=boehm` |
| Workload | plasma preset, density 16, 830×352 raster, 120 FPS cap |
| Runs | five per build per transport, builds alternating, one Chrome per run |
| Sample | the last 600 frames of each run, after 120 frames of warmup |

Times are in milliseconds. Values are the median of the per-run medians, with
the range across runs in parentheses. Ratio is TinyGo / Go; below 1 favors
TinyGo.

## Results

### Size, build and startup

| | Go | TinyGo | Ratio |
| --- | --- | --- | --- |
| `app.wasm` | 9.23 MiB | 2.81 MiB | 0.30× |
| `app.wasm`, gzip | 2.50 MiB | 0.96 MiB | 0.38× |
| Cold build | 2.6 s | 8 min 50 s | about 200× |
| Load, to program start | 182–206 | 170–175 | 0.82–0.96× |
| Program start to first frame | 69 | 65–67 | 0.95–0.97× |

### Stages with shared memory (the gallery's default)

| Stage | Go p50 | TinyGo p50 | Ratio | Go p95 | TinyGo p95 |
| --- | --- | --- | --- | --- | --- |
| frame | 9.99 (9.54–10.27) | 8.74 (8.64–8.76) | 0.87× | 12.90 | 13.57 |
| render (R) | 1.04 (1.01–1.05) | 0.94 (0.93–0.97) | 0.90× | 2.28 | 4.82 |
| · setup | 0.06 | 0.05 | 1.00× | 0.10 | 0.07 |
| · submit | 0.01 | 0.01 | – | 0.02 | 0.01 |
| · map | 0.81 (0.79–0.82) | 0.79 (0.79–0.82) | 0.98× | 2.10 | 1.67 |
| · copy | 0.06 | 0.05 | 0.83× | 0.09 | 3.86 |
| encode (E) | 0.65 (0.50–0.66) | 0.16 (0.16–0.17) | 0.25× | 0.76 | 3.94 |
| view | 0.05 | 0.04 | 0.71× | 0.13 | 0.04 |

### Stages with direct PNG (`?medium=direct`)

| Stage | Go p50 | TinyGo p50 | Ratio | Go p95 | TinyGo p95 |
| --- | --- | --- | --- | --- | --- |
| frame | 53.12 (52.74–53.38) | 43.41 (40.78–44.13) | 0.82× | 81.11 | 70.91 |
| render (R) | 27.12 (26.95–27.21) | 6.71 (5.93–7.23) | 0.25× | 53.95 | 32.86 |
| · setup | 0.08 | 0.08 | 1.00× | 0.28 | 0.10 |
| · submit | 0.01 | 0.01 | – | 0.02 | 0.02 |
| · map | 26.79 (26.57–26.92) | 2.31 (2.24–2.39) | 0.09× | 53.57 | 28.43 |
| · copy | 0.07 (0.07–0.07) | 4.34 (0.05–4.67) | – | 0.57 | 4.68 |
| encode (E) | 25.07 (25.00–25.36) | 16.61 (16.14–18.27) | 0.66× | 49.20 | 57.05 |
| view | 0.05 | 0.04 | 0.71× | 0.08 | 0.05 |

### Stage definitions

| Stage | What it covers |
| --- | --- |
| frame | Interval between presented frames |
| render (R) | The whole GPU call; contains setup, submit, map and copy |
| setup | Creating buffers, the bind group and the commands |
| submit | Submitting the commands to the queue |
| map | Waiting for the mapped readback |
| copy | Copying pixels into the image |
| encode (E) | Preparing the image and encoding the Kitty frame |
| view | Building the Bubble Tea view |

### Memory when the report was read (MiB)

| | Go | TinyGo | Ratio |
| --- | --- | --- | --- |
| Program heap (`HeapSys`), shared memory | 11.8 | 166.7 | 14× |
| Program heap (`HeapSys`), direct PNG | 17.8 | 201.1 | 11× |
| Page JS heap, shared memory | 19.6 (6.9–30.2) | 1024.0 (1020.3–1029.3) | 52× |
| Page JS heap per frame, shared memory | 0.03 | 1.20 | 48× |
| Page JS heap, direct PNG | 51.9 (15.1–72.1) | 417.5 (376.6–423.5) | 8× |

## Findings

- **Memory leak in the TinyGo build.** The page's JS heap grows about 1.2 MiB
  per frame and reached 8.4 GiB in a 70-second run; Go's stays in the tens of
  MiB. `syscall/js` releases JS values through `runtime.SetFinalizer`, which
  TinyGo 0.42.0 implements for its default collector but not for the Boehm
  collector (`-gc=boehm`) that the gallery builds with, so every frame's
  buffers stay referenced. The build from before these changes grows the same way.
- **One TinyGo hang.** One run in 27 hung the page, with the renderer at 240%
  CPU and 10 GiB. It didn't recur and the cause is not established; the memory
  growth is a candidate, not a finding. No Go run hung.
- **Median frame time favors TinyGo.** 13% faster with shared memory, 18% with
  PNG. With shared memory its median is near the 8.33 ms cap, so the
  difference there may be understated.
- **Tail latency does not.** TinyGo's 95th percentiles are no better than
  Go's, and its copy and encode stages show pauses of about 4 ms that Go does
  not have. With PNG its copy median fell on either side of such a pause from
  run to run, so that row carries no ratio.
- **Startup is close.** Pages were served from localhost, so download time is
  absent; over a network the smaller module should matter more. One Go load
  took 2.7 s.

## Limits of these numbers

- **Stage boundaries are approximate.** The page has one thread, so a stage
  that yields to the event loop also absorbs whatever else was queued: the
  terminal decoding the previous PNG, painting, the runtime's scheduler. The
  tenfold gap in `map` under PNG is where the waiting landed, not GPU speed.
  Why it lands differently in the two runtimes is not established. Frame time
  is the number to trust.
- **`shader compile` is not compilation.** WebGPU compiles asynchronously in
  the GPU process; about 1 ms for six pipelines (Go 1.1, TinyGo 0.6) is the
  cost of the calls.
- **Medians under 0.05 ms carry no ratio.** They are a few steps of the page's
  5 µs clock.
- **Allocation counters are not compared.** TinyGo's Boehm collector reports
  `HeapAlloc` as zero, and its `TotalAlloc` has not been checked against Go's.
- **Narrow coverage.** One preset, one raster size, headless Chrome only (not
  Safari, not a visible window), and the machine was in use during the runs
  (load average 4–5).

## What changed in `ntcharts`

| Area | Change |
| --- | --- |
| Stage timers | The header gains a line splitting R into `setup / submit / map / copy` |
| Report | `-report` adds per-stage summaries (mean, p50, p95 over the last 600 frames), startup times, heap size and the compiler; existing fields are unchanged |
| Browser report | The browser build publishes the same report in `globalThis.ntchartsShadersReport` every two seconds |
| New flag | `-medium shm\|direct` |
| Browser flags | The browser build reads flags from the page's query string, e.g. `?preset=julia&medium=direct` |
| Harness | `task bench-shaders-wasm` runs `examples/shaders/bench/compare.mjs` |
| Saved runs | `examples/shaders/bench/results-2026-09-29.json` |
| Docs | A results section in `examples/shaders/README.md` and a CHANGELOG entry |

Files touched:

```
CHANGELOG.md
Taskfile.yml
examples/shaders/README.md
examples/shaders/dispatch.go
examples/shaders/gpu.go
examples/shaders/gpu_test.go
examples/shaders/main.go
examples/shaders/model.go
examples/shaders/model_test.go
examples/shaders/options_js.go
examples/shaders/options_native.go
examples/shaders/stats.go            (new)
examples/shaders/stats_test.go       (new)
examples/shaders/bench/compare.mjs   (new)
examples/shaders/bench/results-2026-09-29.json (new)
```

## Reproducing

From `~/projects/ntcharts`, with Node 22+, Chrome and TinyGo 0.42.0:

```sh
task build-wasm-site
task bench-shaders-wasm -- -runs 5 -out /tmp/shaders-bench.json

# Print the comparison of saved results without measuring again:
node examples/shaders/bench/compare.mjs -from examples/shaders/bench/results-2026-09-29.json
```

## Things to know

- **TinyGo toolchain.** The dev checkout at `~/projects/tinygo/build/tinygo`
  couldn't compile its own runtime for wasm (the source tree is ahead of the
  binary), so the 0.42.0 release was downloaded into the session scratchpad.
  The checkout was not touched.
- **Rebuilt artifacts.** `web/demos/shaders/app.wasm` and
  `web/demos/shaders-tinygo/app.wasm` (gitignored) now contain the
  instrumented builds.
- **ntcharts3d.** Its TinyGo gallery uses the same WebGPU readback path, so
  the same leak is likely there. It has not been measured.
- **Verification.** `task test` passes in `ntcharts`; the real-GPU test
  passes; a native run in a pseudo-terminal produced the new header line and
  a report with stage samples. Native display in a real Kitty terminal was
  not checked.
