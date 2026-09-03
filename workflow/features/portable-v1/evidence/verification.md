# Portable V1 worker evidence

## Identity and scope

- Pinned placeholder base: `64ac2f3f52f05644c1e8584b3a9044d5dda1cdd1`.
- Contract commit: `d827d1b76c7f3f6c0a3ddc965a15bfadb0fddbed`.
- Initial implementation commit: `a4b39e8d20caa140d9d5dd10fdd0e5a41ad0eff4`.
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
SHA-256 provides integrity, not producer authentication.

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
go test -mod=readonly -race -coverprofile=/tmp/gotth-portability-coverage-final.out ./...
go test -mod=readonly -race -count=50 ./...
```

Post-correction cold-review coverage: 94.3% under race. Final revision-matched
evidence is pending. Boundary, negative, staged sink, every-byte
truncation, V1 golden, resume, external-package, and bounded-write tests pass.
No claimed public behavior or failure class lacks a direct test. Residual lines
are defensive variants within already-covered classifications.

Fresh five-second fuzz runs:

- archive roundtrip: 251,419 executions;
- checkpoint parser: 304,670 executions;
- malformed archive: 323,051 executions.

## External consumer

A separate module under `/tmp/gotth-portability-consumer.2WECWD` imports the
public package through a local replacement and passes race, vet, and build. It
constructs exporter/importer, persists/parses a checkpoint, implements the
staged sink interfaces, and observes explicit completion. The directory is
disposable task-owned scratch and is removed at handoff.

## Performance evidence

The end-to-end matrix and limitations are in `docs/performance.md`. Export uses
33,176-33,178 B/op and 10 allocs/op across 0 byte, 1 KiB, 1 MiB, and 16 MiB
payloads. No optimization or speedup is claimed; Amdahl inputs are N/A.

Exact uninstrumented commands were:

```text
GOTTH_PORTABILITY_PERF=1 go test -mod=readonly -run '^TestPerformanceSamples$' -count=1 -v ./pkg/portability
go test -mod=readonly -run '^$' -bench '^BenchmarkExport$' -benchmem -benchtime=500ms -count=5 ./pkg/portability
```

The harness reported `GOMAXPROCS=4`; each export benchmark operation is serial.

Worker review also split `ErrIO` from `ErrMalformed` and `ErrTruncated`.
Injected source/destination failures verify the distinction and that underlying
consumer error text remains absent from the public error string.

The following pre-correction artifacts are retained as audit history but are
superseded and not final evidence:

| Superseded external artifact | SHA-256 |
| --- | --- |
| `/tmp/gotth-portability-performance-final.log` | `1ed2ee04cc1ebfd8394463dbd0a27b3026d77b0e942c492309fdf2e20ca331b8` |
| `/tmp/gotth-portability-bench-final.log` | `0a32c024a3b5e6b6e541e065181f7a8d0bd5b77f7c6aa80988ad9df9ed4f10ee` |
| `/tmp/gotth-portability-race50-final.log` | `8e38ec201ca587f9939a1f915df632b078806383ad7f1f26a66c1f49ffa15379` |
| `/tmp/gotth-portability-coverage-final.out` | `7f2663a3e1b3e923e47b2875f83ad588e9d2d7a78ddc0cadd3847a9be3ae7fa7` |

## Graph review

Graphify extracted implementation commit
`a4b39e8d20caa140d9d5dd10fdd0e5a41ad0eff4` in code-only mode: 175 nodes,
401 edges, and 12 communities. It skipped 16 non-code documents and seven
unclassified non-code files. The graph is at
`~/.cache/openclaw-code-index/gotth-portability/a4b39e8d20caa140d9d5dd10fdd0e5a41ad0eff4/graphify/graphify-out/graph.json`
with SHA-256
`c54becbead65354d8124ee25217e8f4560d51bff1dc2ef294e4fd77935bd237a`.
Graph queries show `nextChain` reaches both export and import, while
`Checkpoint` reaches both resume constructors and the codec. Source and tests
confirm those consequential edges. Graph output is iteration evidence, not a
correctness or admission oracle.

## Cold review

The first fresh worker Judge pass failed on weak checkpoint invariants, missing
prior-checkpoint recovery, incomplete unknown-commit modeling, dishonest I/O
classification, incomplete wire/boundary pins, and stale evidence. Those code
and test findings are corrected. A fresh pass found no remaining code blocker;
final revision-matched evidence remains the only open worker gate.

## Remaining gate

Final revision-matched reruns, clean-clone verification, a fresh evidence-only
Judge pass, and cleanup remain. The orchestrator owns independent final review
and admission. Workflow state intentionally remains `in_progress`.
