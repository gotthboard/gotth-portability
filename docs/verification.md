# Verification status

Current worker verification:

- Go 1.26.6 format, diff check, vet, build, race, and coverage pass locally.
- Hardened source `5ec6f2f759078fa7fe894f99ff1c245a6c67e6f4` reports
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
- Fresh five-second fuzz admissions passed 100,791 valid roundtrip inputs,
  90,176 arbitrary checkpoint inputs, 252,625 bounded arbitrary archive inputs,
  and 44,769 valid-archive mutation inputs. A simultaneous four-process fuzz
  attempt produced one harness EOF; the affected target was rerun sequentially
  and passed, so the retained admission artifact contains the passing run.
- Fifty consecutive race-instrumented suite runs pass.
- A separate external module imports the public package and passes race, vet,
  and build gates using a local replacement for this unreleased source.
- Performance percentiles, allocation evidence, and limitations are in
  `docs/performance.md`.
- Graphify 0.9.32 extracted 263 nodes and 636 edges from the exact hardened
  source commit. It confirms context-aware exact write reaches construction,
  and context-aware exact read reaches header/import through `readExact`; those
  edges were verified in source. The graph has no self-loop or exact duplicate
  edge.

Worker and independent cold reviews drove checkpoint, recovery, cancellation,
callback-identity, I/O-contract, wire, allocation, fuzz-oracle, and evidence
corrections. Revision-matched gates, performance, clean-clone,
external-consumer, graph, and fuzz evidence pass. A real downstream consumer
schema/pin does not yet exist; workflow therefore remains `in_progress` and
unreleased. Independent final admission remains orchestrator-owned.
