# Performance admission

## Decision

No optimization was introduced and no speedup is claimed. The direct streaming
mechanism is admitted for worker handoff: it performs one bounded-buffer pass
over each payload on export and import, hashes each payload byte once per pass,
and keeps library allocation constant as payload size grows.

With no baseline/candidate optimization pair, hotspot share `P`, hotspot
speedup `S_hotspot`, and an Amdahl prediction are not applicable. Inventing
them would be nonsense.

## Environment and method

- Linux 7.1.8 x86-64 on an Intel Core i7-7660U.
- Go 1.26.6-X:nodwarf5; the benchmark harness reported `GOMAXPROCS=4`, while
  each `BenchmarkExport` operation itself is serial.
- Performance samples measure an in-memory export plus validating import,
  including archive allocation, payload hashing, frame processing, and staged
  sink writes. Each workload has fresh archive state; setup is inside the
  measurement because that is the supported operation.
- The benchmark separately exports to `io.Discard` and reports Go allocations.
  Five 500 ms samples were collected per payload size.

Exact uninstrumented commands:

```text
GOTTH_PORTABILITY_PERF=1 go test -mod=readonly -run '^TestPerformanceSamples$' -count=1 -v ./pkg/portability
go test -mod=readonly -run '^$' -bench '^BenchmarkExport$' -benchmem -benchtime=500ms -count=5 ./pkg/portability
```

Neither timing command used the race detector. Race-instrumented results are
correctness evidence only and are not mixed into the timing table.

## End-to-end results

| Workload | Samples | p50 | p95 | p99 | Throughput |
| --- | ---: | ---: | ---: | ---: | ---: |
| empty archive | 200 | 2.095 us | 6.446 us | 16.564 us | 369,208.73 archives/s |
| 1 x 1 KiB | 200 | 17.826 us | 55.780 us | 145.519 us | 36,274.15 archives/s |
| 16 x 64 KiB | 80 | 6.257875 ms | 6.835371 ms | 10.094234 ms | 158.62 archives/s |
| 4 x 1 MiB | 20 | 22.887666 ms | 25.904109 ms | 38.462257 ms | 41.79 archives/s |
| 1,000 empty records | 10 | 11.824175 ms | 18.878047 ms | 18.878047 ms | 78.37 archives/s |

The 1,000-empty-record workload isolates per-frame overhead and is the
pathological metadata regime. The large workloads show the expected linear
hash/copy cost. This is a local admission fixture, not a consumer service-level
objective.

## Allocation and scaling

Export reported 33,176-33,178 B/op and 10 allocs/op at 0 bytes, 1 KiB, 1 MiB,
and 16 MiB. That constant allocation is the fixed 32 KiB payload buffer plus
small hash/frame state; it does not grow with archive size. The five observed
samples ranged from roughly 7.46-8.17 ms at 1 MiB and 44.6-141.1 ms at
16 MiB. The high 16 MiB outlier reinforces that this shared host is unsuitable
for an SLO; allocation counts and asymptotic byte work are the stable evidence.

The timing spread and the materially different superseded pre-correction run
reflect an uncontrolled shared host. No CPU profile, scheduler isolation, or
hardware counter evidence was collected because there is no optimization
proposal. Re-run against real consumer record distributions before setting an
SLO or changing the mechanism.
