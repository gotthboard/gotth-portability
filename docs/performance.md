# Performance admission

## Decision

This revision corrects allocation shape; it makes no runtime speedup claim. The
direct streaming mechanism performs one bounded-buffer pass over each payload
on export and import and hashes each payload byte once per pass. Empty records
do not allocate a payload buffer. The first non-empty record allocates one
32 KiB buffer, which the exporter/importer retains and reuses. Empty records
are not allocation-free: frame metadata, hash state, and callback machinery
produce the measured per-record allocations below.

There is no baseline/candidate runtime claim, so hotspot share `P`, hotspot
speedup `S_hotspot`, and an Amdahl prediction are not applicable. The admission
question here is visible allocation shape, not a fabricated speedup.

## Environment and method

- Linux 7.1.8-arch1-3 x86-64 on an Intel Core i7-7660U.
- Go 1.26.6-X:nodwarf5; the benchmark harness reported `GOMAXPROCS=4`, while
  each `BenchmarkExport` operation itself is serial.
- Performance samples measure an in-memory export plus validating import,
  including archive allocation, payload hashing, frame processing, and staged
  sink writes. Each workload has fresh archive state; setup is inside the
  measurement because that is the supported operation.
- The benchmark separately exports to `io.Discard` and imports a prebuilt
  archive. Both report Go allocations. Five 500 ms samples were collected for
  zero records, one empty record, 1,000 empty records, and one payload of 1 KiB,
  1 MiB, or 16 MiB.

Exact uninstrumented commands:

```text
env GOTTH_PORTABILITY_PERF=1 go test -mod=readonly -run '^TestPerformanceSamples$' -count=1 -v ./pkg/portability
go test -mod=readonly -run '^$' -bench '^Benchmark(Export|Import)$' -benchmem -benchtime=500ms -count=5 ./pkg/portability
```

Neither timing command used the race detector. Race-instrumented results are
correctness evidence only and are not mixed into the timing table.

## End-to-end results

| Workload | Samples | p50 | p95 | p99 | Throughput |
| --- | ---: | ---: | ---: | ---: | ---: |
| empty archive | 200 | 2.328 us | 6.955 us | 15.702 us | 335,385.90 archives/s |
| 1 x 1 KiB | 200 | 15.147 us | 44.610 us | 163.725 us | 45,251.11 archives/s |
| 16 x 64 KiB | 80 | 6.255921 ms | 10.243377 ms | 11.176727 ms | 142.56 archives/s |
| 4 x 1 MiB | 20 | 25.235500 ms | 36.678080 ms | 37.869382 ms | 36.63 archives/s |
| 1,000 empty records | 10 | 2.352264 ms | 3.205633 ms | 3.205633 ms | 406.99 archives/s |

The 1,000-empty-record workload isolates per-frame overhead and is the
pathological metadata regime. The large workloads show the expected linear
hash/copy cost. This is a local admission fixture, not a consumer service-level
objective.

## Allocation and scaling

| Workload | Export B/op, allocs/op | Import B/op, allocs/op |
| --- | ---: | ---: |
| zero records | 304, 3 | 544, 21 |
| one empty record | 440, 9 | 776, 35 |
| 1,000 empty records | 136,304, 6,003 | 232,544, 14,021 |
| 1 x 1 KiB | 33,208, 10 | 33,544, 36 |
| 1 x 1 MiB | 33,208, 10 | 33,544, 36 |
| 1 x 16 MiB | 33,208-33,209, 10 | 33,544, 36 |

The zero- and empty-record rows prove that no unconditional 32 KiB payload
buffer remains. The 1,000-empty-record case shows linear metadata/hash
allocation rather than hidden 32 KiB-per-record churn. Non-empty allocation is
flat across payload size because one buffer is retained and reused. This does
not claim that per-record metadata allocations are free or optimal.

Timing differs drastically from the immediately superseded review run despite
unchanged success-path mechanics, demonstrating that this shared host is
uncontrolled. No CPU profile, scheduler isolation, or hardware counter evidence
was collected because there is no runtime speedup claim. Allocation counts and
asymptotic byte work are the stable evidence. Re-run against real consumer
distributions before setting an SLO or changing the mechanism.
