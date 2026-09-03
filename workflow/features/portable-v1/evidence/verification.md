# Portable V1 worker evidence

## Identity and scope

- Pinned placeholder base: `64ac2f3f52f05644c1e8584b3a9044d5dda1cdd1`.
- Contract commit: `d827d1b76c7f3f6c0a3ddc965a15bfadb0fddbed`.
- Initial implementation commit: `a4b39e8d20caa140d9d5dd10fdd0e5a41ad0eff4`.
- First hardened source commit: `5a291d9752839333ba033005b21c80686836b9e3`.
- Independent-review hardening commit:
  `5ec6f2f759078fa7fe894f99ff1c245a6c67e6f4`.
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
failure uses a fresh cleanup context bounded by `AbortTimeout`. SHA-256 provides
integrity, not producer authentication.

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
go test -mod=readonly -race -coverprofile=/tmp/gotth-portability-coverage-5ec6f2f.out ./...
go test -mod=readonly -race -count=50 ./...
```

Hardened source coverage is 92.9% under race. Boundary, negative, staged sink,
every-byte truncation, literal empty/non-empty archive and checkpoint goldens,
resume, external-package, and bounded-write tests pass. New tests bracket every
caller-owned cancellation seam, enforce a finite no-progress bound and strict
Writer contract, verify lazy/reused payload buffers, and inject every public
sentinel through every callback class to prove causes cannot spoof
`errors.Is`. Residual lines are defensive variants within already-covered
classifications.

Fresh five-second fuzz runs:

- archive roundtrip: 100,791 executions;
- checkpoint parser: 90,176 executions;
- bounded arbitrary archive: 252,625 executions;
- valid-archive mutation oracle: 44,769 executions.

Exact commands:

```text
go test -mod=readonly -run '^$' -fuzz '^FuzzArchiveRoundTrip$' -fuzztime=5s ./pkg/portability
go test -mod=readonly -run '^$' -fuzz '^FuzzCheckpointParserNeverPanics$' -fuzztime=5s ./pkg/portability
go test -mod=readonly -run '^$' -fuzz '^FuzzArbitraryArchiveTerminatesWithinRecordLimit$' -fuzztime=5s ./pkg/portability
go test -mod=readonly -run '^$' -fuzz '^FuzzValidArchiveMutationOracle$' -fuzztime=5s ./pkg/portability
```

Four simultaneous fuzz processes produced one harness EOF while gathering the
arbitrary-archive baseline. That target was rerun alone and passed; the retained
artifact is the passing sequential run. `ErrComplete` is no longer treated as a
generic successful error: both fuzz oracles require an available, consistent
manifest and expected committed output.

## External consumer

A separate temporary module imported the public package from a no-local clone
detached at the exact hardened source and passed race, vet, and build. It
constructs exporter/importer, persists/parses a checkpoint, implements the
staged sink interfaces, and observes explicit completion. The no-local clone
itself passed readonly race, vet, and build with a clean detached status. Both
temporary directories are disposable. The final proof log prints and hashes
the external module files, resolves the replacement path, records the replaced
repository's exact detached HEAD and clean status, prints `go list -m -json
all`, and traces race, vet, and build. Result logs are retained and hashed
below.

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
GOTTH_PORTABILITY_PERF=1 go test -mod=readonly -run '^TestPerformanceSamples$' -count=1 -v ./pkg/portability
go test -mod=readonly -run '^$' -bench '^Benchmark(Export|Import)$' -benchmem -benchtime=500ms -count=5 ./pkg/portability
```

The harness reported `GOMAXPROCS=4`; each export benchmark operation is serial.

Worker review also split `ErrIO` from `ErrMalformed` and `ErrTruncated`.
Independent review then found that callback causes could counterfeit public
sentinels. `errors.Is` now exposes only the library classification; `Causer`
provides explicit access while consumer error text remains absent.

| Revision-matched external artifact | SHA-256 |
| --- | --- |
| `/tmp/gotth-portability-verify-5ec6f2f.log` | `23e80afd43805dc883a355e83e58ea78556356707e1b8422225994f9d74c5064` |
| `/tmp/gotth-portability-coverage-5ec6f2f.out` | `6d133be7ecf6d6ddaf320cb2c5b0347f2f8e02e98c9270655b473fe5f2336c93` |
| `/tmp/gotth-portability-race50-5ec6f2f.log` | `56cdbf494ccafd8535ca1febc8628719039a5341849d606bd5bb464a13d75792` |
| `/tmp/gotth-portability-fuzz-roundtrip-5ec6f2f.log` | `ec240d37578d3031a22485d423670b46374984c3194f200b2cc8d1fbc344e77f` |
| `/tmp/gotth-portability-fuzz-checkpoint-5ec6f2f.log` | `0e90723140a541fb19f7828fe946414321a76feb94067818e16a5f5e577b0c0a` |
| `/tmp/gotth-portability-fuzz-arbitrary-5ec6f2f.log` | `596206e2c046674eb0742bae96f4c1dec78eb1890ac1bb056a08fd0fdfa02b01` |
| `/tmp/gotth-portability-fuzz-mutation-5ec6f2f.log` | `91616219d2b83dd53f6702b7fe9ac631903bdf37ca59195cdafee33797233439` |
| `/tmp/gotth-portability-performance-5ec6f2f.log` | `5bf1acd2e66a3e75b8031050660c6f88392df0132802bd6ece697b91bc1b3f53` |
| `/tmp/gotth-portability-bench-5ec6f2f.log` | `de7a4579e68bad8f117fe52f233f2435104e7d535ccf725d51755b6a5755c84f` |
| `/tmp/gotth-portability-clean-clone-5ec6f2f.log` | `4a8b57cfd27c45a479764b6047ae1fb0d1ac6aa9169aeaf9636e8a154988b533` |
| `/tmp/gotth-portability-external-proof-5ec6f2f.log` | `76e5ddafa021c7904275812751d2ebf546a27a9c69c0361812def81f9168f0c6` |

## Graph review

Graphify extracted hardened source commit
`5ec6f2f759078fa7fe894f99ff1c245a6c67e6f4` in code-only mode: 263 nodes,
636 edges, and 13 communities. It skipped 17 non-code documents and six
unclassified non-code files. The graph is at
`~/.cache/openclaw-graphify/gotth-portability/5ec6f2f759078fa7fe894f99ff1c245a6c67e6f4/graphify-out/graph.json`
with SHA-256
`a3835b40f1a78ff299a01b66b5dda001d1bd9e9eb47057fcbca3a81df68eb101`.
Graph queries show `NewExporter` reaches `writeExactContext`, while
`NewImporter` reaches `readHeader`, `readExact`, and `readExactContext`.
Source confirms those consequential edges. The graph contains no self-loop or
exact duplicate edge. Graph output is iteration evidence, not correctness or
admission.

## Cold review

Earlier worker reviews corrected checkpoint invariants, first-record recovery,
unknown-commit modeling, I/O classification, wire/boundary pins, and evidence.
Independent review of exact prior head `66feed0` then rejected cancellation
boundaries, callback error identity, Reader/Writer contract handling,
per-record 32 KiB allocation, and a weak arbitrary-archive fuzz oracle. Those
implementation and evidence findings are corrected in exact source `5ec6f2f`.

## Remaining gate

No actual downstream consumer schema or dependency pin exists. Inventing one
would defeat portability's consumer-owned boundary. That product integration
is the remaining blocker; workflow stays `in_progress`, and the package remains
unreleased. The orchestrator owns any further independent review and admission.
