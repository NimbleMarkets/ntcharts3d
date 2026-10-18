#!/bin/sh
# Run from the repository root. Keep shm samples in separate processes: a sink
# does not unlink objects as a real terminal would, and ntpicture caps pending
# shared memory at 64 MiB. 20 ops (40 burst frames) stays below that cap.
set -eu
go test -run '^$' -bench '^BenchmarkPresentation/(glyph|direct)/' -benchmem -benchtime=300ms -count=3 .
for workload in camera texture bars burst; do
  for sample in 1 2 3; do
    go test -run '^$' -bench "^BenchmarkPresentation/shm/$workload$" -benchmem -benchtime=20x -count=1 .
  done
done
go test -run '^$' -bench '^BenchmarkSoftwareRendererLifetime$' -benchmem -benchtime=300ms -count=3 .
