# Dedicated software renderer and bounded depth reuse

Implemented 2026-10-18. This supersedes the unimplemented depth-scratch proposal
in [the original performance assessment](README.md#depth-scratch-assessment-and-upstream-handoff).

The model now owns a software renderer independently of its GPU adapter.
Snapshot and scheduled render commands share it under a lock. GPU fallback
cleanup leaves it alive. Close waits for active work, rejects queued software
jobs, releases both renderers, and is idempotent. Frame construction and other
model mutations remain on the event loop; rendering may overlap Snapshot.
Dispose and stale/pending-frame handling retain their existing contracts.

The engine camera conversion now maps orbit fields explicitly so the engine's
new glTF camera fields do not break compilation. No chart camera behavior changes.

With the coordinated nt3d change, a steady 640×320 render avoids **819,200 B and
one allocation per frame** after warmup. Images and exposed depth remain owned
by callers. Engine depth retention is capped at 4 MiB; larger snapshots allocate
temporary scratch, small frames shrink excessive capacity, and Close releases it.
Measured retained depth was 819,200 B after 640×320, still 819,200 B after a
2048×1024 snapshot, and zero after Close.

This is an allocation improvement, not a general latency claim. Three sequential
72-operation samples on Apple M5 Max / Go 1.27.2, same dependency versions and
4× anisotropy, produced these medians (range):

| Workload | Before ms/op | After ms/op | B/op before → after |
| --- | ---: | ---: | ---: |
| Snapshot camera | 6.187 (6.171–6.195) | 6.257 (6.090–6.408) | 1,638,915 → 819,715 |
| Snapshot fresh texture | 12.118 (11.923–12.249) | 12.187 (12.158–12.316) | 4,087,334 → 3,268,129 |
| Direct presentation camera | 7.599 (7.505–7.985) | 6.918 (6.899–6.992) | 3,689,456 → 2,870,043 |

Other cases varied: resize Snapshot median rose from 9.778 to 9.998 ms, and shm
bars from 2.531 to 2.871 ms. Avoid inferring an across-the-board speedup. The
allocation reduction is approximately 23.4 MiB/s at 30 frames/s, which justifies
the bounded scratch and small adapter lifetime cost for sustained animation.
`PresentedMsg` reports ordered application output completion, not terminal
display acknowledgement.

The [full paired report and raw measurements](https://github.com/NimbleTerminal/nt3d/tree/main/performance/depth-reuse)
include lifetime, camera, texture, resize, large-then-small, glyph/direct/shm and
burst cases, ns/op/B/op/allocs, source identifiers, reproduction code, and logs.
It also exists locally at `../nt3d/performance/depth-reuse/README.md` from this
repository root. All temporary workspaces used identical dependency versions;
production pins remain unchanged. Both baseline and changed chart copies used
the same explicit camera mapping to build against current nt3d.

Validation passed: both repositories' CI matrices (including race, vet and
WASM builds), native GPU suites, new Snapshot/render/Close overlap tests,
fallback/invalid/late-job checks, existing cancellation/stale/presentation tests,
and an isolated mapview3d consumer race suite. 648 map/bar/line/point images
matched the baseline byte-for-byte. nt3d additionally checks transparency,
clipping, supersampling, overlays, outlines and retained public depth.
Actual browser execution, terminal painting, and shared-memory consumption by
an emulator were not measured.

No production pin changes, pushes or releases were made. Release nt3d
first, update ntcharts3d and its example modules to that engine release, then
release ntcharts3d and update consumers. The adapter also passes tests against
current published pins, but those pins cannot reuse depth until upgraded.
