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
| empty archive | 200 | 25.442 us | 106.732 us | 151.115 us | 28,444.18 archives/s |
| 1 x 1 KiB | 200 | 252.210 us | 609.331 us | 1.723953 ms | 3,169.92 archives/s |
| 16 x 64 KiB | 80 | 68.748623 ms | 102.088278 ms | 141.738960 ms | 13.49 archives/s |
| 4 x 1 MiB | 20 | 245.800180 ms | 367.929337 ms | 420.502129 ms | 3.78 archives/s |
| 1,000 empty records | 10 | 100.553123 ms | 127.474194 ms | 127.474194 ms | 9.59 archives/s |

The 1,000-empty-record workload isolates per-frame overhead and is the
pathological metadata regime. The large workloads show the expected linear
hash/copy cost. This is a local admission fixture, not a consumer service-level
objective.

## Allocation and scaling

Export reported 33,176-33,178 B/op and 10 allocs/op at 0 bytes, 1 KiB, 1 MiB,
and 16 MiB. That constant allocation is the fixed 32 KiB payload buffer plus
small hash/frame state; it does not grow with archive size. The five observed
samples ranged from roughly 4.0-5.1 ms at 1 MiB and 53-76 ms at 16 MiB,
consistent with linear payload work.

The timing spread, especially the first empty sample, reflects an uncontrolled
shared host. No CPU profile, scheduler isolation, or hardware counter evidence
was collected because there is no optimization proposal. Re-run against real
consumer record distributions before setting an SLO or changing the mechanism.
