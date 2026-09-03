# Portable V1 worker evidence

## Identity and scope

- Pinned placeholder base: `64ac2f3f52f05644c1e8584b3a9044d5dda1cdd1`.
- Contract commit: `d827d1b76c7f3f6c0a3ddc965a15bfadb0fddbed`.
- Initial implementation commit: `a4b39e8d20caa140d9d5dd10fdd0e5a41ad0eff4`.
- Hardened source commit: `5a291d9752839333ba033005b21c80686836b9e3`.
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
go test -mod=readonly -race -coverprofile=/tmp/gotth-portability-coverage-5a291d9.out ./...
go test -mod=readonly -race -count=50 ./...
```

Hardened source coverage is 94.3% under race. Boundary, negative, staged sink,
every-byte truncation, literal empty/non-empty archive and checkpoint goldens,
resume, external-package, and bounded-write tests pass. No claimed public
behavior or failure class lacks a direct test. Residual lines are defensive
variants within already-covered classifications.

Fresh five-second fuzz runs:

- archive roundtrip: 242,866 executions;
- checkpoint parser: 180,671 executions;
- malformed archive: 294,402 executions.

Exact commands:

```text
go test -mod=readonly -run '^$' -fuzz '^FuzzArchiveRoundTrip$' -fuzztime=5s ./pkg/portability
go test -mod=readonly -run '^$' -fuzz '^FuzzCheckpointParserNeverPanics$' -fuzztime=5s ./pkg/portability
go test -mod=readonly -run '^$' -fuzz '^FuzzMalformedArchiveNeverCompletesSilently$' -fuzztime=5s ./pkg/portability
```

## External consumer

A separate temporary module imported the public package from a no-local clone
detached at the exact hardened source and passed race, vet, and build. It
constructs exporter/importer, persists/parses a checkpoint, implements the
staged sink interfaces, and observes explicit completion. The no-local clone
itself passed readonly race, vet, and build with a clean detached status. Both
temporary directories are disposable; command-traced result logs are retained
and hashed below.

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

| Revision-matched external artifact | SHA-256 |
| --- | --- |
| `/tmp/gotth-portability-verify-5a291d9.log` | `85d7fa21699bb6420a968fcd7c53ac1e25b6b34fd7164e9e9bf8fb3752939857` |
| `/tmp/gotth-portability-coverage-5a291d9.out` | `440ecc2a936cb48620d27ba478e067fdad57fea2bc39447e1b3c81d35a5ff2e0` |
| `/tmp/gotth-portability-race50-5a291d9.log` | `de02a62dfffa627bf374b992453541536c2e067db60c7eedbfce9caa781d5bfa` |
| `/tmp/gotth-portability-fuzz-roundtrip-5a291d9.log` | `7cd633a136e3acfeb350f948d9b76509cabe79428121bbb61383d2fab77f22ee` |
| `/tmp/gotth-portability-fuzz-checkpoint-5a291d9.log` | `ab4e27ea2147126a5f1920d3fa761217c59d5801fd1880a2ea46248b83b19f16` |
| `/tmp/gotth-portability-fuzz-archive-5a291d9.log` | `b167ef4352e3e31860c7ca7dd83d531ab333c9952378bc7b9526cf916e3efe66` |
| `/tmp/gotth-portability-performance-5a291d9.log` | `0eb05713918d47aa1fdabe2792e55ca5bf4e08b30a19e338fa8553d606b1af4f` |
| `/tmp/gotth-portability-bench-5a291d9.log` | `bcf3a1077d8a0f81c6dfe685653aad8cd21bf61e677f6c7a8c8f278d8f9dee93` |
| `/tmp/gotth-portability-clean-clone-5a291d9.log` | `45917f5efaaa532268840445242710d831b9fe55d2196ba703f789fb49b61a81` |
| `/tmp/gotth-portability-external-consumer-5a291d9.log` | `4d541d15bcd9ade61b10f18aa76d34e0cde57524088979b25d233e03f48f9b73` |

## Graph review

Graphify extracted hardened source commit
`5a291d9752839333ba033005b21c80686836b9e3` in code-only mode: 206 nodes,
490 edges, and 16 communities. It skipped 17 non-code documents and seven
unclassified non-code files. The graph is at
`~/.cache/openclaw-code-index/gotth-portability/5a291d9752839333ba033005b21c80686836b9e3/graphify/graphify-out/graph.json`
with SHA-256
`6ec6c413e774cc8419ec132a13799b635c3fb89205f84f7e4c85125c08747c25`.
Graph queries show `nextChain` reaches both export and import, while
`Checkpoint` reaches both resume constructors and the codec. Source and tests
confirm those consequential edges. The graph contains no self-loop or exact
duplicate edge. Graph output is iteration evidence, not a correctness or
admission oracle.

## Cold review

The first fresh worker Judge pass failed on weak checkpoint invariants, missing
prior-checkpoint recovery, incomplete unknown-commit modeling, dishonest I/O
classification, incomplete wire/boundary pins, and stale evidence. Those code
and test findings are corrected. A fresh pass found no remaining code blocker;
its evidence audit then failed on missing fuzz commands and missing retained
clean-clone/external-consumer logs. Those reproducibility gaps are corrected in
the current bookkeeping commit.

## Remaining gate

A fresh evidence-only Judge pass must verify this final bookkeeping commit; its
verdict is handoff evidence and does not mutate canonical source. The
orchestrator owns independent final review and admission. Workflow state
intentionally remains `in_progress`.
