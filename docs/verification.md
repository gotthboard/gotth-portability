# Verification status

Current worker verification:

- Go 1.26.6 format, diff check, vet, build, race, and coverage pass locally.
- Hardened source `95edd8269173a7d580d2ab0a9ecf3560c4c2b5ce` reports
  92.9% race-instrumented statement coverage. Residual statements are
  defensive malformed-checkpoint/header, impossible count-overflow, and rare
  delegated-I/O branches; every public operation, sentinel family, record state
  boundary, checksum/finalization path, and resume path has direct tests.
- Metadata and configured payload limits cover limit-1, limit, limit+1, and
  materially-beyond cases. Every byte truncation of a representative archive
  fails without producing a manifest.
- Bounded-stream tests move a 5 MiB+17 byte payload in writes no larger than
  32 KiB. V1 has literal empty-archive, non-empty archive, and persistent
  checkpoint golden tests.
- Cancellation seam tests cover header/metadata/payload/digest/footer I/O,
  compatibility, begin, staged write, commit unknown-outcome, and bounded abort.
  Pre-canceled construction performs no I/O. Begin-with-stage-and-error cleanup,
  abort failure, caller cancellation, and nil/error returns after cleanup
  deadline are covered without losing the primary classification.
- Strict I/O tests cover positive-short nil writes without retry, legal
  data-plus-error reads, negative/oversized Reader counts, transient empty reads,
  and the documented no-progress bound.
- Every public sentinel is injected through export Reader/Writer,
  compatibility, begin, staged write, commit, and abort callbacks. Only the
  library classification participates in `errors.Is`; `Causer` retains explicit
  access to the raw potentially sensitive cause while the library error string
  remains redaction-safe.
- Fresh sequential five-second fuzz admissions passed 182,220 valid roundtrip
  inputs, 241,957 arbitrary checkpoint inputs, 314,127 bounded arbitrary archive
  inputs, and 83,297 valid-archive mutation inputs. Exact-valid fuzzing requires
  one committed map entry even for an empty payload, exact bytes/counts/chain,
  zero aborts, Manifest, and EOF. Invalid mutations assert no forbidden commit,
  completion, or Manifest.
- Fifty consecutive race-instrumented suite runs pass.
- A separate external module imports the public package and passes race, vet,
  and build gates using a local replacement for this unreleased source. Both
  it and the detached clean-clone proof record each command and exit status,
  exact HEAD/clean status, module/source hashes, and replacement resolution.
- Performance percentiles, allocation evidence, and limitations are in
  `docs/performance.md`.
- Graphify 0.9.32 extracted 274 nodes and 677 edges in 14 communities from the
  exact hardened source commit. The graph binds to that commit and has no
  self-loop or exact duplicate edge; consequential begin/abort and fuzz-oracle
  edges were verified directly in source when ambiguous graph names existed.

Worker and independent cold reviews drove checkpoint, recovery, cancellation,
callback-identity, I/O-contract, wire, allocation, fuzz-oracle, and evidence
corrections. Revision-matched gates, performance, clean-clone,
external-consumer, graph, and fuzz evidence pass. A real downstream consumer
schema/pin does not yet exist; workflow therefore remains `in_progress` and
unreleased. Independent final admission remains orchestrator-owned.
