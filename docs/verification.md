# Verification status

Current worker verification:

- Go 1.26.6 format, diff check, vet, build, race, and coverage pass locally.
- The post-correction cold-review run reports 94.3% race-instrumented statement
  coverage; final revision-matched evidence is pending. Residual statements are
  defensive malformed-checkpoint/header, impossible count-overflow, and rare
  delegated-I/O branches; every public operation, sentinel family, record state
  boundary, checksum/finalization path, and resume path has direct tests.
- Metadata and configured payload limits cover limit-1, limit, limit+1, and
  materially-beyond cases. Every byte truncation of a representative archive
  fails without producing a manifest.
- Bounded-stream tests move a 5 MiB+17 byte payload in writes no larger than
  32 KiB. V1 has literal empty-archive, non-empty archive, and persistent
  checkpoint golden tests.
- Injected reader and writer failures prove that local I/O failure is distinct
  from malformed input and premature EOF while retaining redacted diagnostics.
- Fresh fuzz admissions passed 251,419 valid roundtrip inputs, 304,670 arbitrary
  checkpoint inputs, and 323,051 arbitrary archive inputs.
- Fifty consecutive race-instrumented suite runs pass.
- A separate external module imports the public package and passes race, vet,
  and build gates using a local replacement for this unreleased source.
- Performance percentiles, allocation evidence, and limitations are in
  `docs/performance.md`.
- Graphify 0.9.32 extracted 175 nodes and 401 edges from the implementation
  commit. It confirms the rolling chain reaches export and import and the
  checkpoint reaches both resume constructors; those edges were verified in
  source and tests.

Worker cold review found and drove concrete checkpoint, recovery, error, wire,
and evidence corrections. Final revision-matched reruns remain pending.
Independent final admission remains orchestrator-owned; this status creates no
release claim.
