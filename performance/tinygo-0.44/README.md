# TinyGo 0.44.0 gallery comparison

Measured on 2026-10-28 at commit `9ba5f98`, with nt3d v0.7.0 and
go-booba v0.10.5. Both compilers used the same sources and dependencies.

On an Apple M5 Max running macOS 27.0.1 and Chrome 156, both official
TinyGo releases used Go 1.26.8, LLVM 22.1.4, and identical flags:
`-target=wasm -gc=boehm -opt=z -no-debug`. The browser viewport was
1280 × 800, with Apple WebGPU and Kitty shared memory.

Each scene ran three times per compiler in a fresh browser, with five
seconds of warmup and 20 seconds of measurement. Compiler order alternated
between rounds. FPS counts emitted Kitty frame commands, not display refreshes.

| Scene | 0.43.0 median FPS | 0.44.0 median FPS | Change |
| --- | ---: | ---: | ---: |
| 16,000-point scatter | 23.10 | 24.00 | +3.9% |
| 96 × 96 surface | 22.85 | 23.90 | +4.6% |
| 12 × 7 bars | 23.10 | 23.90 | +3.5% |
| Vector field | 23.20 | 24.05 | +3.7% |

All 24 samples completed without browser errors. The p95 frame intervals
remained about 64 ms for both versions; the throughput gain was modest.
Raw WASM size fell from 3,485,222 to 3,355,124 bytes (3.73% smaller),
and gzip size fell from 1,272,371 to 1,231,154 bytes (3.24% smaller).

In a separate 45-second scatter run, buffers directly referenced by
TinyGo's JS value table totaled 3,773 MiB with 0.43.0 versus 3.65 MiB
with 0.44.0, even after forced JS garbage collection. This counts directly
held buffers, not total browser or GPU memory, and is one diagnostic run
per compiler. The difference is consistent with the Boehm finalizer support
added in [TinyGo 0.44.0](https://github.com/tinygo-org/tinygo/releases/tag/v0.44.0).
These short local measurements do not predict results on other hardware.

The Pages workflow now pins 0.44.0. The complete demo-site build, visual
checks of all four scenes, and workflow lint passed. Benchmark scripts
and raw measurements are retained locally rather than checked into Git.
