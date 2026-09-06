# Verification status

Current worker verification binds exact source
`efa533ae2c212e5a94c983ec3ab512d267dd65a2`.

- Lightweight local checks used `GOMAXPROCS=2` and `-p=1`: focused importer
  outcome tests, package vet, package build, non-mutating `gofmt -l`, and
  `git diff --check` all passed.
- Full verification ran only on `development`, in the isolated clean detached
  clone `/tmp/gotth-portability-efa533a.bmHfmt/repo`, with Go 1.26.6. Repository
  `make verify`, an independent full test, race test, coverage test, all four
  fuzz targets, an external-consumer module, performance samples, and five
  benchmark samples all passed.
- Every development trace records the literal command, exact source/tree IDs,
  toolchain, timestamps, exit status, raw-output hash, and exact `HEAD` plus
  porcelain-v2 status before and after. Every command began and ended clean at
  `efa533a`. In particular, `make verify` used the failing, non-mutating
  `gofmt -l` check and left exact `HEAD` and status unchanged.
- Statement coverage is 94.9%. `ResumeImporter` remains 100% covered. The four
  repaired combined outcomes have direct tests: `NewImporter` and
  `ResumeImporter` compatibility rejection plus cancellation, staged `Write`
  failure plus cancellation, and `Begin` returning nil/nil plus cancellation.
  Tests assert the exact class set, every explicit raw cause, redacted public
  text, checkpoint preservation, and required Abort/Commit behavior.
- The adjacent importer outcome audit rechecked compatibility-only,
  cancellation-only, begin-error, begin-stage cleanup, staged-write-only,
  commit-only, commit-cancellation, and abort-failure paths. Existing
  classifications remain unchanged outside the four verified repairs.
- Five-second fuzzing passed 2,344,018 valid roundtrip executions, 1,614,245
  arbitrary checkpoint executions, 3,094,791 bounded arbitrary archive
  executions, and 837,707 valid-archive mutation executions. Exact-valid
  fuzzing requires committed payload/count/chain state and a valid Manifest;
  invalid mutations cannot produce forbidden commit or completion state.
- A fresh external module resolved the unreleased package through a local
  replacement to the exact clean clone, then passed readonly race, vet, and
  build gates. Its module and consumer test hashes are retained in the trace.
- Metadata and payload boundary tests cover limit-1, limit, limit+1, and
  materially-beyond cases. Every byte truncation of a representative archive
  fails without producing a manifest. Bounded streaming uses writes no larger
  than 32 KiB, and literal V1 archive/checkpoint goldens remain covered.
- Strict I/O tests cover positive-short nil writes without retry, legal
  data-plus-error reads, invalid Reader counts, transient empty reads, and the
  no-progress bound. State-boundary tests prove preflight failures are
  retryable without record/frame I/O while failures after progress poison the
  object.
- Performance and allocation evidence for exact source is in
  `docs/performance.md`. No speedup or cross-host timing claim is made.
- The historical Graphify result still binds runtime baseline `d22c2de`; no
  exact-source graph claim is made for this localized repair.

All retained exact-source artifacts are indexed and hashed in
`workflow/features/portable-v1/evidence/verification.md`. A real downstream
consumer schema and dependency pin still do not exist. Workflow remains
`in_progress` and unreleased; final admission is orchestrator-owned.
