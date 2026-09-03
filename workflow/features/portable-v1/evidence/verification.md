# Portable V1 worker evidence

## Identity and scope

- Pinned placeholder base: `64ac2f3f52f05644c1e8584b3a9044d5dda1cdd1`.
- Contract commit: `d827d1b76c7f3f6c0a3ddc965a15bfadb0fddbed`.
- Initial implementation commit: `a4b39e8d20caa140d9d5dd10fdd0e5a41ad0eff4`.
- First hardened source commit: `5a291d9752839333ba033005b21c80686836b9e3`.
- Independent-review hardening commit:
  `5ec6f2f759078fa7fe894f99ff1c245a6c67e6f4`.
- Post-repair hardening commit:
  `95edd8269173a7d580d2ab0a9ecf3560c4c2b5ce`.
- Final cancellation/provenance hardening commit:
  `d22c2de207a3a6b046b7ed96aa72d7640ca616f0`.
- Branch/worktree: `feature/v1-portability` at
  `/tmp/gotth-portability-worktrees/v1-portability`.
- No push, PR, tag, release, deployment, live database, secret, or remote state
  was touched.

## Boundary decisions

The package moves opaque records. Schema meaning, authorization, disclosure,
redaction, retention, legal hold, storage, transport, compression, encryption,
and release policy remain consumer-owned. Import sinks stage one record and
commit only after digest verification. Sink commit errors have unknown outcome;
consumers must reconcile and make commit idempotent by archive ID and sequence.
Cancellation observed after `Commit` is the same unknown outcome. Pre-commit
failure uses a fresh cleanup context bounded by `AbortTimeout`. A stage returned
with a Begin error is cleaned up, and cleanup deadline expiry is an `ErrSink`
failure even after a nil Abort result. Cancellation observed after a Begin error
is retained as `ErrIO` alongside the primary `ErrSink` classification. SHA-256
provides integrity, not producer authentication. `Causer.Cause` is raw and
potentially sensitive; only `Error()` is redaction-safe.

## Toolchain and capacity

- Go: `go1.26.6-X:nodwarf5 linux/amd64`.
- Graphify: 0.9.32, local code-only extraction.
- Root preflight: 5% bytes and 1% inodes used; no capacity threshold approached.
- gopls: N/A on the canonical host because the binary is absent. Direct source,
  compiler, vet, tests, coverage, fuzzing, and Graphify supplied the required
  evidence without installing or starting another tool.
- Context broker, Zoekt, and ast-grep: N/A; canonical docs and the new package
  were small enough for bounded direct reads and `/usr/bin/rg`.

## Correctness gates

Passed locally:

```text
git diff --check -- .
go vet -mod=readonly ./...
go build -mod=readonly ./...
go test -mod=readonly -race -coverprofile=/tmp/gotth-portability-coverage-d22c2de.out ./...
go test -mod=readonly -race -count=50 ./...
```

Hardened source coverage is 92.9% under race. Boundary, negative, staged sink,
every-byte truncation, literal empty/non-empty archive and checkpoint goldens,
resume, external-package, and bounded-write tests pass. New tests bracket every
caller-owned cancellation seam, enforce a finite no-progress bound and strict
Writer contract, verify lazy/reused payload buffers, and inject every public
sentinel through every callback class to prove causes cannot spoof
`errors.Is`. Begin error paths cover nil/non-nil stages, cancellation after the
callback, and Abort success/failure. Simultaneous begin failure and cancellation
retain `ErrSink`, `ErrIO`, and both raw causes. Both nil/error returns after a
cleanup deadline have direct tests. Exact-valid fuzz seeds require the commit
entry even for empty payload. Residual lines are defensive variants within
already-covered classifications.

Fresh five-second fuzz runs:

- archive roundtrip: 19,653 executions;
- checkpoint parser: 7,696 executions;
- bounded arbitrary archive: 15,923 executions;
- valid-archive mutation oracle: 8,283 executions.

Exact commands:

```text
go test -mod=readonly -run '^$' -fuzz '^FuzzArchiveRoundTrip$' -fuzztime=5s ./pkg/portability
go test -mod=readonly -run '^$' -fuzz '^FuzzCheckpointParserNeverPanics$' -fuzztime=5s ./pkg/portability
go test -mod=readonly -run '^$' -fuzz '^FuzzArbitraryArchiveTerminatesWithinRecordLimit$' -fuzztime=5s ./pkg/portability
go test -mod=readonly -run '^$' -fuzz '^FuzzValidArchiveMutationOracle$' -fuzztime=5s ./pkg/portability
```

All four final runs were sequential. `ErrComplete` is not treated as a generic
successful error: exact-valid oracles require one commit, map membership, exact
bytes, zero aborts, counts, digest-derived rolling chain, Manifest, and EOF.
Invalid mutation classes assert no forbidden commit, completion, or Manifest.

## External consumer

A separate temporary module imported the public package from a no-local clone
detached at the exact hardened source and passed race, vet, and build. It
constructs exporter/importer, persists/parses a checkpoint, implements the
staged sink interfaces, and observes explicit completion. The no-local clone
itself passed readonly race, vet, and build with a clean detached status. Both
temporary directories are disposable. Every final trace records exact detached
HEAD and clean porcelain-v2 status, source hashes, toolchain/kernel, literal
command, start/end timestamps, explicit exit status, SHA-256 of a separately
retained raw output, and that raw output inline. The clean-clone proof also
records explicit per-command exits for race/vet/build. The external trace prints
and hashes its module files, hashes the replaced source, resolves `go.mod`
replacement with `go list -m -json all`, and records explicit per-command exits
for race/vet/build. Trace helpers, traces, and raw artifacts are retained and
hashed below, so helper-backed command records remain reproducible.

## Performance evidence

The end-to-end matrix and limitations are in `docs/performance.md`. Zero-record
export uses 304 B/op and three allocations; one empty record uses 440 B/op and
nine allocations. Zero-record import uses 544 B/op and 21 allocations; one
empty record uses 776 B/op and 35 allocations. Non-empty export/import remains
about 33.2/33.5 KiB per operation from 1 KiB through 16 MiB payloads. Thus empty
records do not pay the 32 KiB buffer, and the first non-empty record's buffer is
reused. No runtime speedup is claimed; Amdahl inputs are N/A.

Exact uninstrumented commands were:

```text
env GOTTH_PORTABILITY_PERF=1 go test -mod=readonly -run '^TestPerformanceSamples$' -count=1 -v ./pkg/portability
go test -mod=readonly -run '^$' -bench '^Benchmark(Export|Import)$' -benchmem -benchtime=500ms -count=5 ./pkg/portability
```

The harness reported `GOMAXPROCS=4`; each export benchmark operation is serial.

Worker review also split `ErrIO` from `ErrMalformed` and `ErrTruncated`.
Independent review then found that callback causes could counterfeit public
sentinels. `errors.Is` now exposes only the library classification. `Causer`
provides explicit access to the unchanged raw cause, which may be sensitive and
must be consumer-redacted before logging or display; public `Error()` text
remains redaction-safe.

| Revision-matched retained artifact | SHA-256 |
| --- | --- |
| `/tmp/gotth-portability-verify-d22c2de.log` | `3d424f8690a0d1a835cf5de1620e58904c42ae8a94642d8ab2051b56b47b7507` |
| `/tmp/gotth-portability-verify-d22c2de.raw.log` | `5d4014153793a187de23270b75de8a100fbb4170d4951d5b717abee699a89af7` |
| `/tmp/gotth-portability-coverage-d22c2de.log` | `63668c9601c257d818d38811fc0510c102cd7e86c09c36d72fd468ba336c7674` |
| `/tmp/gotth-portability-coverage-d22c2de.raw.log` | `7267971e0853a381a1e1e1a64fca7d2413dac7910c55a516d6c5b7c98f240a97` |
| `/tmp/gotth-portability-coverage-d22c2de.out` | `162437dd10f8e8e50da8c3154c3b86d996ae2dccae23e41dbdc4f80ba43a2f49` |
| `/tmp/gotth-portability-race50-d22c2de.log` | `a4dc7d8c3ad85246b68be01d11a2a84d7efbe37f2ec2b442efe7ff59e31a4391` |
| `/tmp/gotth-portability-race50-d22c2de.raw.log` | `fab3a1c9685109b80455e816238da73706c380824898f1e3ac62db1949ad6db7` |
| `/tmp/gotth-portability-fuzz-roundtrip-d22c2de.log` | `45b4a6e6a0a3b7bd2a146391ebb7b984c9331c8feae1dfe24f97766330a13f32` |
| `/tmp/gotth-portability-fuzz-roundtrip-d22c2de.raw.log` | `47e86cd1fde75a48a1d5e7cfff9d292cfcf3683a9c937f7975c845223861df23` |
| `/tmp/gotth-portability-fuzz-checkpoint-d22c2de.log` | `2ef4bc921341014da286cc78f134c250cf28ad18de4ac905333424aea79ebca8` |
| `/tmp/gotth-portability-fuzz-checkpoint-d22c2de.raw.log` | `49929053895723b42a71c0911e03f7e0eefa5e0c3607b592e7865c2ea6c13dfb` |
| `/tmp/gotth-portability-fuzz-arbitrary-d22c2de.log` | `3ef65f4045f13706edab166bb71da961e3402c209454ba34b87ae20644b0c448` |
| `/tmp/gotth-portability-fuzz-arbitrary-d22c2de.raw.log` | `43425cbcd02802589d102d3dfe500f5a743d13f6a253eefb2038854be5b08a55` |
| `/tmp/gotth-portability-fuzz-mutation-d22c2de.log` | `abee5b0d8369fefb85c92ba97cfe498082070a5ad2c7dde0c192618f13f70d3c` |
| `/tmp/gotth-portability-fuzz-mutation-d22c2de.raw.log` | `c771cdb1f21923eed3cbc1b4f69368b7ae4d8687c21b78082754f1e20c300fb2` |
| `/tmp/gotth-portability-performance-d22c2de.log` | `43cdf0a96f9c4bc0800ef906309bcd997ea0dee53d9a8af9f91c0fcd39a7b904` |
| `/tmp/gotth-portability-performance-d22c2de.raw.log` | `0a33cad1e03f7867d58e5cc51f3d123e350ac9c73a00dd84ca36ba73135776e5` |
| `/tmp/gotth-portability-bench-d22c2de.log` | `b970b5f0e07d0fb67e5165e699c05babccaddd1b272a07ed85de2713790e4eac` |
| `/tmp/gotth-portability-bench-d22c2de.raw.log` | `adbce8469f5f916a834d706bd233de48427c24b97e7aacd9c67f098bb2bece38` |
| `/tmp/gotth-portability-clean-clone-d22c2de.log` | `1c0c04cd6664714b6e8a694ea140228baf16e5d5cd08fab2864efc0b4fdf36b0` |
| `/tmp/gotth-portability-clean-clone-d22c2de.raw.log` | `5c7e0b44950f4b2cad4ac261d962368092554dd29f62879f57f81db73924255c` |
| `/tmp/gotth-portability-external-proof-d22c2de.log` | `491fd334c59ba2fa4d57dd94bb8713127aecaf0947b446328259733d83142151` |
| `/tmp/gotth-portability-external-proof-d22c2de.raw.log` | `90c2e6130546e464e06b0c367b0154359415673aefd8bc90f676a369b077bf8a` |
| `/tmp/gotth-portability-graphify-d22c2de.log` | `825139aa916058b7eed425c227bdc3a4a28cd90cb98bb6dbf13734857ebe498f` |
| `/tmp/gotth-portability-graphify-d22c2de.raw.log` | `5f92c7ac39d3719e08cc5f1d42375003c86c05b2900f55ce51eac7ba66745419` |
| `/tmp/gotth-portability-graphify-cluster-d22c2de.log` | `718ee77405e17ac23408f69a14e9256d4160ac01c8906a12a63a09628887a212` |
| `/tmp/gotth-portability-graphify-cluster-d22c2de.raw.log` | `dc360fc38197f6072e16f61e675c0a487129153561b17818528d3f291cefbfdc` |
| `/tmp/gotth-portability-graph-audit-d22c2de.log` | `2d190674adf087d6ee8f5ba374858a8622a0b7573e1e5b105865a70d8b2909c9` |
| `/tmp/gotth-portability-graph-audit-d22c2de.raw.log` | `bcde7a87b8201fb6e213b3296222788b5ada910e74efeacb407890512a1f6bdb` |
| `/tmp/gotth-portability-trace-d22c2de.sh` | `cbd5361acdad7e1ce8f742f22f5622426e9d9955bdadf477f3b9e65a528d5ef9` |
| `/tmp/gotth-portability-clean-run-d22c2de.sh` | `99edd5829dda4f8d4ed1b121861c6faabb4f7ea28cc02e88e5a984d6e177e21f` |
| `/tmp/gotth-portability-external-run-d22c2de.sh` | `38c7372099dd304441d374e7aff1bba18706651034d714ec0c70a2f9c6f75720` |
| `/tmp/gotth-portability-graph-audit-d22c2de.sh` | `da70f829c463a40a752aaeaab3d4fe5411b7199410721cd2b81cd598bfc28134` |
| `/home/linus/.cache/openclaw-graphify/gotth-portability/d22c2de207a3a6b046b7ed96aa72d7640ca616f0/graphify-out/graph.json` | `f5898d27c61228193a62247126aefeba46fe250e756f49c19dee951fc45b40bc` |
| `/home/linus/.cache/openclaw-graphify/gotth-portability/d22c2de207a3a6b046b7ed96aa72d7640ca616f0/graphify-out/GRAPH_REPORT.md` | `cd2126d3280a0e7f40a1823b84ff3f3139b3f83b6f09fa65bd6502018a94a34c` |

## Graph review

Graphify extracted hardened source commit
`d22c2de207a3a6b046b7ed96aa72d7640ca616f0` in code-only mode: 274 nodes,
677 edges, and 14 communities. It skipped 17 non-code documents and six
unclassified non-code files. The graph is at
`~/.cache/openclaw-graphify/gotth-portability/d22c2de207a3a6b046b7ed96aa72d7640ca616f0/graphify-out/graph.json`
with SHA-256
`f5898d27c61228193a62247126aefeba46fe250e756f49c19dee951fc45b40bc`.
The report records built-from commit `d22c2de2`; extraction, clustering, and
audit traces independently bind exact detached source and record raw outputs.
Graph and source checks cover the Begin/Abort path and exact-valid fuzz helper;
ambiguous method names were resolved directly in source. The graph contains no
self-loop or exact duplicate edge. Graph output is iteration evidence, not
correctness or admission.

## Cold review

Earlier worker reviews corrected checkpoint invariants, first-record recovery,
unknown-commit modeling, I/O classification, wire/boundary pins, and evidence.
Independent review of exact prior head `66feed0` then rejected cancellation
boundaries, callback error identity, Reader/Writer contract handling,
per-record 32 KiB allocation, and a weak arbitrary-archive fuzz oracle. Those
implementation and evidence findings are corrected in exact source `5ec6f2f`.
Independent review of documentation/evidence head `ca9de4d` then found raw
`Cause` mislabeled as redacted, abandoned Begin stage-plus-error state, missing
post-Abort deadline reporting, missing cost comments, empty-payload fuzz oracle
holes, silent proof logs, and stale changelog chronology. All technical
findings are corrected in exact source `95edd82`; the final evidence commit is
documentation-only.
Independent review of evidence head `e4fe93f` then found Begin callback
cancellation suppression, understated checkpoint/resume temporary space, a
false unconditional ParseCheckpoint lower bound, unbound raw admission logs,
and missing changelog times. Implementation `d22c2de` corrects the callback and
complexity findings. The provenance-bound traces and this documentation-only
evidence commit correct the evidence and chronology findings.

## Remaining gate

No actual downstream consumer schema or dependency pin exists. Inventing one
would defeat portability's consumer-owned boundary. That product integration
is the remaining blocker; workflow stays `in_progress`, and the package remains
unreleased. The orchestrator owns any further independent review and admission.
