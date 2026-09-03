# Verification status

Current worker verification:

- Go 1.26.6 format, diff check, vet, build, race, and coverage pass locally.
- Hardened source `8d7836c09cf5e35c5088065c0a3906ae1ddb3e2e` reports
  93.6% race-instrumented statement coverage. Residual statements are
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
  Pre-canceled construction performs no I/O. Begin-with-error tests cover nil
  and non-nil stages, abort success/failure, and cancellation after the callback;
  simultaneous begin failure and cancellation retain both `ErrSink` and `ErrIO`
  without losing raw causes. Nil/error returns after cleanup deadline are also
  covered.
- Strict I/O tests cover positive-short nil writes without retry, legal
  data-plus-error reads, negative/oversized Reader counts, transient empty reads,
  and the documented no-progress bound.
- Direct state-boundary tests prove that every documented `WriteRecord` and
  `Next` preflight failure consumes no record/frame I/O and permits retry, while
  failure after partial output/input poisons the object.
- Complexity contracts account for checkpoint validation's temporary canonical
  header allocation. ParseCheckpoint distinguishes constant-time prefix/length
  rejection from the linear checksum and valid-input path.
- Every public sentinel is injected through export Reader/Writer,
  compatibility, begin, staged write, commit, and abort callbacks. Only the
  library classification participates in `errors.Is`; `Causer` retains explicit
  access to the raw potentially sensitive cause while the library error string
  remains redaction-safe.
- Against unchanged runtime baseline `d22c2de`, sequential five-second fuzz
  admissions passed 19,653 valid roundtrip
  inputs, 7,696 arbitrary checkpoint inputs, 15,923 bounded arbitrary archive
  inputs, and 8,283 valid-archive mutation inputs. Exact-valid fuzzing requires
  one committed map entry even for an empty payload, exact bytes/counts/chain,
  zero aborts, Manifest, and EOF. Invalid mutations assert no forbidden commit,
  completion, or Manifest.
- Against unchanged runtime baseline `d22c2de`, fifty consecutive
  race-instrumented suite runs pass.
- Against unchanged runtime baseline `d22c2de`, a separate external module
  imports the public package and passes race, vet, and build gates using a local
  replacement for this unreleased source. Every
  retained trace records exact detached HEAD and clean status, source hashes,
  toolchain, literal command, start/end times, exit status, raw-output hash, and
  inline raw output. Separate raw files are retained and hashed. The external
  proof additionally records module hashes and replacement resolution. Helper
  scripts named by trace commands are also retained and hashed.
- Unchanged-runtime performance percentiles, allocation evidence, and
  limitations are in `docs/performance.md`.
- Graphify 0.9.32 extracted 274 nodes and 677 edges in 14 communities from
  baseline `d22c2de`. Its report and provenance trace bind to that
  commit, and it has no self-loop or exact duplicate edge; consequential
  begin/abort and fuzz-oracle edges were verified directly in source when
  ambiguous graph names existed.

Worker and independent cold reviews drove checkpoint, recovery, cancellation,
callback-identity, I/O-contract, retry/poisoning contract, cost-bound, wire,
allocation, fuzz-oracle, and evidence corrections. Focused and full gates are
revision-matched to the current source. Performance, clean-clone,
external-consumer, graph, and fuzz evidence remain the `d22c2de` runtime
baseline; `8d7836c` changes comments and tests, not runtime behavior. A real
downstream consumer schema/pin does not yet exist; workflow therefore remains
`in_progress` and unreleased. Independent final admission remains
orchestrator-owned.
