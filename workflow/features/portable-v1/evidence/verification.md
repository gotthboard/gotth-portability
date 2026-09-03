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
failure even after a nil Abort result. SHA-256 provides integrity, not producer
authentication. `Causer.Cause` is raw and potentially sensitive; only `Error()`
is redaction-safe.

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
go test -mod=readonly -race -coverprofile=/tmp/gotth-portability-coverage-95edd82.out ./...
go test -mod=readonly -race -count=50 ./...
```

Hardened source coverage is 92.9% under race. Boundary, negative, staged sink,
every-byte truncation, literal empty/non-empty archive and checkpoint goldens,
resume, external-package, and bounded-write tests pass. New tests bracket every
caller-owned cancellation seam, enforce a finite no-progress bound and strict
Writer contract, verify lazy/reused payload buffers, and inject every public
sentinel through every callback class to prove causes cannot spoof
`errors.Is`. Begin stage-plus-error cleanup and both nil/error returns after a
cleanup deadline have direct tests. Exact-valid fuzz seeds require the commit
entry even for empty payload. Residual lines are defensive variants within
already-covered classifications.

Fresh five-second fuzz runs:

- archive roundtrip: 182,220 executions;
- checkpoint parser: 241,957 executions;
- bounded arbitrary archive: 314,127 executions;
- valid-archive mutation oracle: 83,297 executions.

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
temporary directories are disposable. Both final proof logs print each command
before execution and an explicit exit status afterward. The clean-clone log
records detached exact HEAD, clean porcelain-v2 status, source/module hashes,
and race/vet/build. The external log prints and hashes its module files, hashes
the replaced source, resolves `go.mod` replacement with `go list -m -json all`,
records exact detached/clean source, and traces race/vet/build. Result logs are
retained and hashed below.

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
sentinels. `errors.Is` now exposes only the library classification. `Causer`
provides explicit access to the unchanged raw cause, which may be sensitive and
must be consumer-redacted before logging or display; public `Error()` text
remains redaction-safe.

| Revision-matched external artifact | SHA-256 |
| --- | --- |
| `/tmp/gotth-portability-verify-95edd82.log` | `bea481c7178b05d751373ee0afedbdbcdc601c0f98ba28fc347066e23756edb8` |
| `/tmp/gotth-portability-coverage-95edd82.out` | `7a04d9d1e4b801f7f4a82ba208ffda9499bb2c8b3f0e06ddefda44596dfb6cff` |
| `/tmp/gotth-portability-race50-95edd82.log` | `e5f44023a3bbfc02e16d38545ea94aa1368f3d0e187db469f9c3a53a8a748cd4` |
| `/tmp/gotth-portability-fuzz-roundtrip-95edd82.log` | `265e24f048249c6f01cfc4356601462b089f289cd806a48bf424c0e2e351c643` |
| `/tmp/gotth-portability-fuzz-checkpoint-95edd82.log` | `1be4c5ecb258327db0ff915f6b7068633c61378750c4113d46f112f11847c7d5` |
| `/tmp/gotth-portability-fuzz-arbitrary-95edd82.log` | `eec3eab2e1d3766c74983e2d4c29d215b4aed2767c1d3d6006ca44308a7cbfdd` |
| `/tmp/gotth-portability-fuzz-mutation-95edd82.log` | `f019c198143ae29e725c4d920e8ce41d5d48c639ae8b41bea43f80d4a3de5be3` |
| `/tmp/gotth-portability-performance-95edd82.log` | `fa6a3911cd550d7c93d89a1f82834ca1a8e5e687ac82f6d9301a9936597e9e1c` |
| `/tmp/gotth-portability-bench-95edd82.log` | `87ad1a173e5987617c19276f207b1eba47da6edcee86af94c5197ec59b3f8283` |
| `/tmp/gotth-portability-clean-clone-95edd82.log` | `25513e847c2edba6c043248ca11e7ab3815cdd11b94c8a2203d6533a8a05fd53` |
| `/tmp/gotth-portability-external-proof-95edd82.log` | `d82f4f5f18d790f819a6f572dc01aefd0ad2f309e984a783826056c0c8ee918b` |

## Graph review

Graphify extracted hardened source commit
`95edd8269173a7d580d2ab0a9ecf3560c4c2b5ce` in code-only mode: 274 nodes,
677 edges, and 14 communities. It skipped 17 non-code documents and six
unclassified non-code files. The graph is at
`~/.cache/openclaw-graphify/gotth-portability/95edd8269173a7d580d2ab0a9ecf3560c4c2b5ce/graphify-out/graph.json`
with SHA-256
`1a4f3a55caa75b20b30ba63f40f7a566f2d09d2d240af1d9171dfdeafee0b29e`.
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

## Remaining gate

No actual downstream consumer schema or dependency pin exists. Inventing one
would defeat portability's consumer-owned boundary. That product integration
is the remaining blocker; workflow stays `in_progress`, and the package remains
unreleased. The orchestrator owns any further independent review and admission.
