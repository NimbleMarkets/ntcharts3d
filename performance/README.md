# Presentation measurement and renderer lifetime

**Update:** [Dedicated software renderer and measured depth reuse](depth-reuse.md) now implements the coordinated lifetime change. The original baseline below is retained for comparison.

Measured 2026-10-18, Apple M5 Max, darwin/arm64, Go 1.27.2.
Dependencies remain nt3d v0.2.0, ntpicture v0.1.0, ntgpuimage v0.3.0.
No production code, filtering defaults, dependency pins, or sibling working
copies were changed. No repository/ancestor AGENTS.md was found; Taskfile's CI
checks apply.

## Reproduce

From the repository root:

```sh
sh scripts/bench-presentation.sh > performance/presentation-baseline.txt 2>&1
go test -race -run TestPresentationPump -count=3 .
task ci
```

Native shared memory requires permission to create POSIX shared-memory objects.
The harness fails if shared memory was requested but warmup used direct output;
unsupported platforms skip it explicitly. Subsequent actual direct/shm frame
counts are always reported: never interpret a mixed run as pure shm performance.

The test file uses real Model mutations, scheduled rendering commands, chart
Update, ntpicture preparation/encoding, ordered RawMsg delivery to io.Discard,
and ntpicture PresentedMsg. Batch commands execute concurrently; Sequence
commands preserve ordering. Retirement timers remain in flight across frames,
and disposal drains outstanding commands. A serial batch executor incorrectly
adds the default 50 ms retirement delay to completion latency; it must not be
used for this benchmark.

PresentedMsg means **ordered output/grid completion**, not terminal display
acknowledgement. This sink measures application-side costs, not emulator paint
speed or interactive FPS. Full chart View construction and its bytes are
included separately, but Bubble Tea screen diffing, actual terminal transport,
OS write backpressure and emulator painting are excluded. The harness schedules
one producer operation at a time, with a separate burst/backpressure case; it
is not a fixed-rate input generator. Scheduling and timer allocations are included.

## Workloads and metrics

All cases warm one fully presented frame before timing. The fixture is a local,
deterministic opaque 512×512 image containing streets, blocks and a river,
matching mapview3d's benchmark pattern and flat two-triangle, unlit surface.
Camera starts at elevation 55°, azimuth -90°, distance 3.4, perspective.
Axes and legend are hidden. Default 4× anisotropy is preserved.

- Camera: azimuth advances by 5° per operation, wrapping after 72 operations.
- Texture: fresh immutable texture copied from the fixture every operation;
  camera stays fixed to isolate replacement/mipmap work from motion.
- Bars: 8×4 bars, same camera trajectory, a normal chart comparison.
- Burst: one camera request followed by three requests while presentation is
  pending; two completed frames and two coalesced requests per operation.
  Each raw sink write sleeps 1 ms. Glyph has no Kitty raw frame output, so its
  burst tests coalescing without that simulated output delay.

The 80×22-cell chart has 80×20 plot cells. Glyph renders 80×40 pixels; Kitty
renders 640×320 at 8×16 pixels per cell. These are different workloads, not an
apples-to-apples transport speed comparison. Software quality adaptation remains
active and is checked after every message: any shrink, renderer fallback, or
unexpected dimensions fails the sample. No reduced-resolution sample is accepted.

`ns/op`, B/op and allocs/op cover producer mutation through completion and View.
`prepare-ns/op` measures texture creation/mutation/frame preparation on the event
loop; `render-ns/frame` measures the chart rendering command (including engine
frame conversion); `output-ns/frame` measures rendered-result delivery through
matching PresentedMsg, including encoding, command scheduling and sink delay.
These stage times exclude View construction and do not sum exactly to ns/op.
Counts are totals for the measured sample. `protocol-B/frame` is emitted RawMsg
bytes (including placement cleanup processed during measurement); `view-B/frame`
is full View content, not a terminal screen diff. Shared-memory pixel payloads
are not protocol bytes: each 640×320 RGBA object also holds 819,200 bytes.

The script uses three 300 ms minimum-duration samples for glyph/direct and
renderer lifetime. Actual operation counts and ns/op in the raw file determine
workload duration. Shm uses three separate processes of 20 operations each
(40 completed frames for burst). This prevents unconsumed sink objects reaching
ntpicture's 64 MiB pending limit. Real terminals consume/unlink these objects;
this sink does not. Native orphan retirement can continue for five seconds
after disposal, and process exit is not an emulator consumption test.

## WASM

`presentation-wasm.txt` records a Node v26.12.0 / Go 1.27.2 WASM lifecycle and
glyph performance run. The root Bubble Tea dependency does not compile for JS;
the temporary modfile uses the exact fork pinned in examples/gallery/go.mod:

```sh
cp go.mod /tmp/presentation-wasm.mod
cat go.sum examples/gallery/go.sum > /tmp/presentation-wasm.sum
go mod edit -modfile=/tmp/presentation-wasm.mod \
  -replace=charm.land/bubbletea/v2=github.com/neomantra/bubbletea/v2@v2.0.0-20261009201241-c397c32a5074
GOOS=js GOARCH=wasm go test -mod=mod -modfile=/tmp/presentation-wasm.mod \
  -exec="$(go env GOROOT)/lib/wasm/go_js_wasm_exec" \
  -run TestPresentationPump -bench '^BenchmarkPresentation/glyph/(camera|texture)$' \
  -benchtime=300ms -count=1 .
```

The lifecycle test covers direct output too; shm skips without a browser
registry. This measures WASM application scheduling under Node, not browser
rendering or JS terminal consumption. Existing untracked gallery browser
benchmarks were inspected but left intact; their prebuilt artifacts and gallery
scenes do not establish current mapview3d presentation performance. Browser
map-texture lifecycle, real-terminal latency, GPU and TinyGo comparisons remain
gaps. Native and Node results must not be treated as equivalent runtime costs.

## Depth scratch assessment and upstream handoff

nt3d v0.2.0 `NewSoftwareRenderer` returns an empty value. Its Render validates
then calls softwareRender; Close is a no-op. `softwareDraw` allocates a fresh
RGBA image and float32 depth slice every frame. Retaining this renderer alone
cannot reuse memory. The lifetime benchmark isolates this fact with map and
bar frames at 640×320, fresh versus retained renderer, with warmed textures.
No sampling or encoded-color conversion changes are included.

Proposed request for the separately assigned **nt3d agent** (handoff only; no
message was sent or sibling tree edited): implement private depth scratch in
the renderer returned by the existing NewSoftwareRenderer API, with no broad
pool or new public buffer API. Keep every returned image independently owned.
Keep `RenderSoftware`'s exported SoftwareResult.Depth independently owned too;
callers may retain it across all later renders. Reset the entire active depth
slice to noDepth on every frame, including equal-area dimension changes.
Preserve supersampling, overlays, invalid-frame behavior and pixel equivalence.
State Render/Close concurrency semantics explicitly, and benchmark time as well
as allocations independently of the separate sampling and mipmap changes.

At 640×320 a reusable float32 depth buffer could avoid 819,200 allocated bytes
per steady-size frame (half of the two raster buffers), at the cost of retaining
that storage. This is a theoretical allocation bound, **not a measured latency
improvement**. Clearing depth and allocating a fresh output image remain necessary.

ntcharts3d should integrate only after that upstream implementation is measured:

- Retain a dedicated software renderer independently of the GPU renderer, so
  GPU failure does not dispose the software fallback resource.
- Serialize Render and Close around private scratch. Snapshot's frame building
  stays on the model's event-loop thread, per its documented contract, but it
  can overlap an already-running interactive render command. The renderer lock
  must cover both paths without sharing mutable image results.
- Grow scratch for larger renders, reset on every resize, and define retention
  bounds/shrink behavior. A snapshot can be much larger than the live chart.
- Reject late jobs after Close, make Close idempotent and release scratch.
  Dispose invalidates/cancels presentation; it does not preempt a synchronous
  software render. Wait for an active render before releasing its storage.
- Test retained images/depth, resize including equal area, Snapshot overlap,
  fallback, stale/cancelled results, invalid frames, repeated Close and race
  behavior. Compare pixels for map, bars, lines, points, axes and transparency;
  compare end-to-end camera and texture workloads, not allocation count alone.

No production persistence change is justified by the current API. There is no
before/after optimization claim here: the delivered baseline and lifetime
comparison establish what to measure after upstream scratch support arrives.
Release/pin changes are unnecessary for this harness; an eventual integration
needs an nt3d release, ntcharts3d release, then consumer pin updates. Nothing was
pushed or published.

## Results

Medians of three local samples; see `presentation-baseline.txt` for raw counts,
bytes, dimensions, selected transport and individual timings. All native
samples retained software rendering, 4× anisotropy and zero resolution shrink;
all bounded shm samples completed entirely over shm (zero direct frames).

| Workload | Raster | Total ms/op | Render ms/frame | Output ms/frame | MiB/op | Allocs/op |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| Glyph camera | 80×40 | 0.244 | 0.157 | 0.026 | 0.216 | 55 |
| Glyph fresh texture | 80×40 | 5.844 | 5.563 | 0.029 | 2.552 | 87 |
| Direct camera | 640×320 | 9.765 | 8.808 | 0.933 | 3.515 | 208 |
| Direct fresh texture | 640×320 | 15.219 | 14.262 | 0.726 | 5.703 | 211 |
| Shm camera | 640×320 | 9.200 | 8.858 | 0.305 | 1.655 | 169 |
| Shm fresh texture | 640×320 | 14.759 | 14.221 | 0.308 | 3.991 | 211 |
| Direct bars | 640×320 | 2.760 | 2.026 | 0.695 | 3.294 | 182 |
| Shm bars | 640×320 | 2.631 | 2.280 | 0.343 | 1.686 | 188 |

Direct camera emitted about 47,909 protocol bytes/frame; direct replacement
31,590; shm 103 plus the separate 819,200-byte pixel object. Camera trajectories
have different sample lengths under duration-based calibration, so small
cross-transport render-time differences are not transport improvements.

Burst samples delivered two frames for four requests, coalescing two. Direct
cost 21.665 ms/op (2.073 ms output/frame), shm 20.239 ms/op (1.367 ms output/frame).
These totals include two renders and the 1 ms per-raw-write simulated sink delay.

Renderer lifetime alone: map fresh 8.621 ms versus retained 8.622 ms, both
1,638,500 B/op and 5 allocations. Bars fresh 1.964 ms versus retained 1.969 ms,
both approximately 1,671,275 B/op and 37 allocations. Retention alone provides
no measured benefit. This control excludes chart frame conversion/Update/View,
so its allocation count should not be compared directly with Snapshot or
presentation totals.

The corrected measurements put rendering ahead of ordered-output preparation
for the map case. They support investigating private depth scratch upstream,
but do not demonstrate a responsiveness gain from that unimplemented change.

Node WASM glyph camera and replacement are recorded separately in the raw WASM
file; they are one short sample each, useful as lifecycle/performance smoke
measurements rather than stable cross-runtime comparisons.

## Validation

The required `task ci` covers formatting, tidy-diff, checksum verification,
all-module race tests, vet, native builds and browser example WASM builds.
The new lifecycle test exercises real completion routing, pending output,
coalescing and disposal; existing suites cover retained snapshots, resizing,
stale frames, cancellation, fallback and shutdown. No production lifetime
change was made, so new scratch-ownership tests are deferred with that change.

The actual mapview3d consumer tests passed from an isolated temporary module:
copy `examples/mapview3d`, `examples/internal`, `examples/go.mod` and go.sum
into a temporary directory, point its ntcharts-osm replacement at the original
read-only sibling, and add an ntcharts3d replacement pointing at this checkout.
Run `go test -mod=readonly ./mapview3d` there. The sibling's working tree and
module files remain untouched.
