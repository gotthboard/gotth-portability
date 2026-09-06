# Verification status

Current worker verification binds exact source
`ac7616b18b6d31282c1c402f7f30353bc6006f9e`.

- Lightweight local checks used `GOMAXPROCS=2` and `-p=1`: focused importer
  combined-error tests, package vet, package compile, non-mutating `gofmt -l`,
  and `git diff --check` all passed.
- Full verification ran only on `development`, in the isolated clean detached
  clone `/tmp/gotth-portability-ac7616b.41ue1v/repo`, with Go 1.26.6. Repository
  `make verify`, an independent full test, race test, coverage test, all four
  fuzz targets, an external-consumer module, performance samples, and five
  benchmark samples all passed.
- Every development trace records the literal command, exact source/tree IDs,
  toolchain, timestamps, exit status, raw-output hash, and exact `HEAD` plus
  porcelain-v2 status before and after. Every command began and ended clean at
  `ac7616b`. In particular, `make verify` used the failing, non-mutating
  `gofmt -l` check and left exact `HEAD` and status unchanged.
- Statement coverage is 94.5%. External-package tests use one standard
  `errors.As` to public `Causer` for compatibility in both constructors,
  nil/nil `Begin` plus cancellation, staged `Write` failure plus cancellation,
  `Commit` failure plus cancellation, and staged failure plus `Abort` failure.
  Each result retains every expected library classification and non-nil raw
  cause while excluding raw causes from outer `errors.Is` traversal and
  `Error()` text. The returned cause implements standard `Unwrap() []error`.
- The adjacent importer outcome audit rechecked compatibility-only,
  cancellation-only, begin-error, begin-stage cleanup, staged-write-only,
  commit-only, commit-cancellation, and abort-failure paths. Existing
  classifications remain unchanged; private tests now use the same single
  `errors.As` idiom instead of recursively searching sibling errors.
- Five-second fuzzing passed 2,460,552 valid roundtrip executions, 1,906,286
  arbitrary checkpoint executions, 3,063,225 bounded arbitrary archive
  executions, and 836,486 valid-archive mutation executions. Exact-valid
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
`workflow/features/portable-v1/evidence/verification.md`. Two fresh independent
reviews of exact evidence head `6aeecae201b839f27f4a7597536e8216373cc626`
returned CLEAN. The standalone technical implementation is admitted and remains
unreleased. A real downstream consumer schema and dependency pin are separate
hard release gates and still do not exist.
