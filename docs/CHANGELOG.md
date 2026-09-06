# Changelog

This repository records user-visible and compatibility-relevant changes here.
Released sections use Semantic Versioning; unreleased work remains under
`Unreleased` and does not imply a tag.
Unless an entry explicitly identifies different event provenance, its heading
uses the named commit's Git author time, rendered to the minute in CDT.

## Unreleased

### 2026-09-05 23:23 CDT - Admit standalone portability implementation

Commit: current commit; hash assigned by Git after commit.

Affected files:

- `README.md`
- `docs/CHANGELOG.md`
- `docs/verification.md`
- `workflow.events.jsonl`
- `workflow.toml`
- `workflow/COVERAGE.md`
- `workflow/features/portable-v1/README.md`
- `workflow/features/portable-v1/evidence/verification.md`
- `workflow/features/portable-v1/review/independent-3.md`
- `workflow/features/portable-v1/review/independent-4.md`

Explanation:

Admit the standalone technical implementation after two fresh independent
CLEAN reviews of exact evidence head `6aeecae`. Preserve real-consumer schema,
behavioral verification, exact dependency pin, release verification, and
explicit release authorization as separate hard release/compatibility gates.

Verification:

- exact-source Go 1.26.6 development-host matrix recorded in feature evidence
- independent CLEAN review and independent CLEAN double-check
- final documentation diff and repository cleanliness checks

Risks / non-goals:

- Technical admission does not create a tag, release, or compatibility promise.
- No push, merge, PR, tag, release, deployment, live request, or GOTTH Board
  mutation is performed by this admission commit.

### 2026-09-05 23:09 CDT — Preserve every combined raw cause

Implementation commit: `ac7616b18b6d31282c1c402f7f30353bc6006f9e`.
Evidence commit: this documentation-only commit. The heading uses the verified
implementation commit author time.

Affected files:

- `README.md`
- `docs/CHANGELOG.md`
- `docs/architecture.md`
- `docs/implementation-spec.md`
- `docs/performance.md`
- `docs/runtime-boundary.md`
- `docs/verification.md`
- `pkg/portability/errors.go`
- `pkg/portability/hardening_test.go`
- `pkg/portability/import.go`
- `pkg/portability/public_api_test.go`
- `pkg/portability/types.go`
- `workflow/COVERAGE.md`
- `workflow/features/portable-v1/evidence/verification.md`

Explanation:

Replace sibling `errors.Join` composition with a private redaction-safe
combined error. Stable classifications remain in outer `errors.Is` traversal,
raw callback text remains absent from `Error()`, and the first standard
`errors.As` match for public `Causer` now returns a standard multi-error that
retains every non-nil raw cause. Remove private recursive cause searching and
cover compatibility, nil/nil Begin, staged Write, Commit, and Abort combinations
through the external package API.

Verification:

- lightweight local focused test, vet, compile, format-list, and diff checks
- exact clean detached `development` clone on Go 1.26.6 passed `make verify`,
  full, race, 94.5% coverage, four fuzz, external-consumer, performance, and
  benchmark gates
- every remote gate retained identical exact HEAD and clean porcelain-v2 status
  before and after; all retained artifact hashes verify

Risks / non-goals:

- The V1 wire format and exported API shape are unchanged. Combined failure
  paths add bounded error storage only when multiple outcomes occur.
- No unrelated callback, protocol, permission, or workflow change was admitted.
  Workflow stays `in_progress`; final admission remains orchestrator-owned.

### 2026-09-05 22:39 CDT — Preserve combined importer outcomes

Implementation commit: `efa533ae2c212e5a94c983ec3ab512d267dd65a2`.
Evidence commit: this documentation-only commit. The heading uses the verified
implementation commit author time.

Affected files:

- `Makefile`
- `README.md`
- `docs/CHANGELOG.md`
- `docs/architecture.md`
- `docs/implementation-spec.md`
- `docs/performance.md`
- `docs/runtime-boundary.md`
- `docs/verification.md`
- `pkg/portability/hardening_test.go`
- `pkg/portability/import.go`
- `pkg/portability/types.go`
- `workflow/COVERAGE.md`
- `workflow/features/portable-v1/evidence/verification.md`

Explanation:

Preserve semantic failures when importer callbacks also cancel: compatibility
rejection remains primary `ErrIncompatible`, while staged-write failure and a
nil/nil Begin remain primary `ErrSink`; every combined result also retains
`ErrIO` and all available explicit causes without leaking callback text. Make
repository formatting verification fail on unformatted Go without mutating the
tree.

Verification:

- lightweight local focused test, vet, build, format-list, and diff checks
- exact clean detached `development` clone on Go 1.26.6 passed `make verify`,
  full, race, 94.9% coverage, four fuzz, external-consumer, performance, and
  benchmark gates
- every remote gate retained identical exact HEAD and clean porcelain-v2 status
  before and after

Risks / non-goals:

- The adjacent audit was limited to importer outcome-classification paths; no
  broader callback or protocol refactor was admitted.
- The V1 wire format and public API are unchanged. Workflow stays `in_progress`
  and final admission remains orchestrator-owned.

### 2026-09-05 22:07 CDT — Preserve simultaneous commit outcomes

Implementation commit: `4596963503d856438ea415dcacce3619f8085436`.
Evidence commit: this documentation-only commit. The heading uses the verified
implementation commit author time.

Affected files:

- `README.md`
- `docs/CHANGELOG.md`
- `docs/architecture.md`
- `docs/implementation-spec.md`
- `docs/verification.md`
- `pkg/portability/hardening_test.go`
- `pkg/portability/import.go`
- `workflow/COVERAGE.md`
- `workflow/features/portable-v1/evidence/verification.md`

Explanation:

Retain a `RecordSink.Commit` error when the same callback cancels its context.
The unknown outcome now reports primary `ErrSink` and additional `ErrIO`, keeps
both raw causes behind `Causer`, does not call `Abort`, and does not advance the
checkpoint.

Verification:

- clean detached `development` clone passed repository `make verify`
- focused race and race-instrumented coverage passed; changed blocks are covered
- three sequential three-second fuzz targets passed

Risks / non-goals:

- The mutation-oracle fuzz target, fresh external-consumer proof, and fresh
  performance/benchmark gates were not rerun before expedited worker handoff.
- Workflow stays `in_progress`; this is not final admission.

### 2026-09-03 15:31 CDT — Complete record-cost and Abort bracketing contracts

Implementation commit: `9a0392433796c26f12b1a83e04becbf9799a3c41`.
Evidence commit: this documentation-only commit. The heading uses the verified
implementation commit author time.

Affected files:

- `docs/CHANGELOG.md`
- `docs/verification.md`
- `pkg/portability/export.go`
- `pkg/portability/hardening_test.go`
- `pkg/portability/import.go`
- `workflow/COVERAGE.md`
- `workflow/features/portable-v1/evidence/verification.md`

Explanation:

Describe complete aggregate Reader, Writer, and sink-callback costs for record
metadata, payload, digest, EOF, and footer work, including local and delegated
auxiliary space and valid-path qualifications. Check the fresh Abort cleanup
context immediately before the consumer callback and fail closed without
calling `RecordSink.Abort` when that context is already expired. Add a direct
zero/negative-timeout regression proving zero Abort calls.

Verification:

- expected-red then green expired-cleanup test under race
- focused Abort/cancellation race tests
- full format, diff, vet, race, coverage, fuzz, and build gates

Risks / non-goals:

- The only runtime change is fail-closed suppression of an Abort callback when
  its fresh cleanup context is already expired.
- The V1 wire format and public API are unchanged. Workflow stays `in_progress`
  and unreleased.

### 2026-09-03 14:59 CDT — Close resume-contract and provenance audit

Implementation commit: `55ba1451cf80661f97c62cc3d41e38e2a0cc77e1`.
Evidence commit: this documentation-only commit. The heading uses the verified
implementation commit author time.

Affected files:

- `docs/CHANGELOG.md`
- `docs/verification.md`
- `pkg/portability/export.go`
- `pkg/portability/import.go`
- `pkg/portability/import_test.go`
- `pkg/portability/wire.go`
- `workflow/COVERAGE.md`
- `workflow/features/portable-v1/evidence/verification.md`

Explanation:

Qualify the remaining I/O and resume complexity contracts so early rejection
never claims a skipped delegated call. Add direct `ResumeImporter` tests for
argument and validation order, pre- and post-compatibility cancellation,
redacted incompatible causes, zero reader I/O, and exact state restoration.
Replace historical changelog globs and narrative file claims with the exact
file union of every commit each entry names.

Verification:

- focused resume tests under race
- exact named-commit file and author-time audit
- full format, diff, vet, race, coverage, fuzz, and build gates

Risks / non-goals:

- Runtime behavior and the V1 wire format are unchanged.
- No downstream consumer schema or pin is fabricated. Workflow stays
  `in_progress` and unreleased.

### 2026-09-03 14:22 CDT — Close constructor-cost and chronology audit

Implementation commit: `63b7c8af4450e577d8f834d7592fcea0b32d6a3d`.
Evidence commit: `b1f718af7ceba135d7deb289822bf87306ca9fbf`. The heading
uses the implementation commit's verified author time.

Affected files:

- `docs/CHANGELOG.md`
- `docs/verification.md`
- `pkg/portability/export.go`
- `workflow/features/portable-v1/evidence/verification.md`

Explanation:

Correct `NewExporter`'s general time and auxiliary-space lower bounds to
`Omega(1)` and limit its tight linear bounds to valid successful paths.
Normalize seven historical changelog headings to their named commits' Git
author times, restore descending chronology, and document the timestamp
provenance convention.

Verification:

- retained contract audit checks the constructor claim and all headings present
  at the implementation commit against exact Git author time and document order
- full format/diff/vet/race/coverage/build passes at 93.6% statement coverage
- fresh traces bind both gates to exact implementation `63b7c8a`

Risks / non-goals:

- Runtime behavior and the V1 wire format are unchanged.
- No downstream consumer schema or pin is fabricated. Workflow stays
  `in_progress` and unreleased.

### 2026-09-03 14:06 CDT — Specify retry and poisoning boundaries

Implementation commits: `8d7836c09cf5e35c5088065c0a3906ae1ddb3e2e`
and `fa12f158ed4c0cf97a95d2cc67da5a0a1a986d40`.
Evidence commit: `c510f4b1ccf7881658bcb386f9d13e8dd6abc288`. The heading
uses the later implementation commit's verified author time.

Affected files:

- `README.md`
- `docs/CHANGELOG.md`
- `docs/implementation-spec.md`
- `docs/verification.md`
- `pkg/portability/checkpoint.go`
- `pkg/portability/export.go`
- `pkg/portability/failure_paths_test.go`
- `pkg/portability/import.go`
- `workflow/features/portable-v1/evidence/verification.md`

Explanation:

State the existing mechanism precisely: argument, validation, limit/offset,
and entry-cancellation preflights perform no record/frame I/O and leave live
objects reusable; failures after I/O begins poison the object. Add direct tests
for every retryable preflight and for poisoning after partial progress. Correct
general lower bounds for checkpoint marshaling and record operations to
`Omega(1)`, retaining tight linear successful/valid-path bounds.

Verification:

- focused retry/poison boundary tests pass under race three consecutive times
- full format/diff/vet/race/coverage/build passes at 93.6% statement coverage
- fresh traces bind both gates to exact implementation `fa12f15`

Risks / non-goals:

- Runtime behavior and the V1 wire format are unchanged; this repairs the
  public contract and its proof.
- No downstream consumer schema or pin is fabricated. Workflow stays
  `in_progress` and unreleased.

### 2026-09-03 13:19 CDT — Preserve Begin cancellation and bind raw evidence

Implementation commit: `d22c2de207a3a6b046b7ed96aa72d7640ca616f0`.
Evidence commit: `d2bd5ea199964a4eea8a6462e80618e49f468f80`. The heading
uses the implementation commit's verified author time.

Affected files:

- `README.md`
- `docs/CHANGELOG.md`
- `docs/architecture.md`
- `docs/implementation-spec.md`
- `docs/performance.md`
- `docs/verification.md`
- `pkg/portability/checkpoint.go`
- `pkg/portability/export.go`
- `pkg/portability/hardening_test.go`
- `pkg/portability/import.go`
- `pkg/portability/types.go`
- `workflow/features/portable-v1/evidence/verification.md`

Explanation:

Sample cancellation after every `Sink.Begin` result shape. A simultaneous begin
error and cancellation now retains primary `ErrSink`, additional `ErrIO`, and
both raw causes while bounded-aborting any returned stage. Correct checkpoint,
resume, and parser complexity bounds to include temporary canonical-header
allocation and constant-time early rejects. Replace unbound raw admission logs
with exact-source traces plus separately hashed raw outputs.

Verification:

- focused Begin/Abort/cancellation and checkpoint/resume tests pass under race
- full clean-clone format/diff/vet/build/race, 92.9% coverage, race x50, four
  sequential five-second fuzz targets, performance/allocation matrix, external
  consumer proof, and revision-bound Graphify pass against exact implementation
  `d22c2de`; every trace records source, command, time, exit, and raw-output hash

Risks / non-goals:

- The API remains unreleased; no compatibility promise or speedup claim is made.
- No downstream consumer schema or pin is fabricated. Workflow stays
  `in_progress` and unreleased.

### 2026-09-03 12:56 CDT — Complete independent post-repair findings

Implementation commit: `95edd8269173a7d580d2ab0a9ecf3560c4c2b5ce`.
Evidence commit: `e4fe93fb1132e62035080c59c342d756fbbec624`. The heading
uses that evidence commit's verified author time.

Affected files:

- `README.md`
- `docs/CHANGELOG.md`
- `docs/architecture.md`
- `docs/implementation-spec.md`
- `docs/performance.md`
- `docs/runtime-boundary.md`
- `docs/verification.md`
- `pkg/portability/errors.go`
- `pkg/portability/export.go`
- `pkg/portability/fuzz_test.go`
- `pkg/portability/hardening_test.go`
- `pkg/portability/import.go`
- `pkg/portability/types.go`
- `pkg/portability/values_test.go`
- `pkg/portability/wire.go`
- `workflow/COVERAGE.md`
- `workflow/features/portable-v1/evidence/verification.md`

Explanation:

Document `Causer.Cause` honestly as raw and potentially sensitive, abort a
non-nil stage returned with a `Sink.Begin` error, report cleanup deadline expiry
as `ErrSink` even after a nil Abort result, add required exact-I/O cost
contracts, and strengthen empty/non-empty valid plus invalid mutation fuzz
oracles. Replace silent proof logs with explicit command/exit traces after this
implementation commit is fixed.

Verification:

- focused raw-cause, begin-cleanup, cleanup-deadline, and fuzz-oracle tests pass
- full format/diff/vet/build/race, 92.9% coverage, race x50, four sequential
  five-second fuzz targets, performance/allocation matrix, traced clean-clone
  and external-consumer proofs, and revision-bound Graphify pass against exact
  implementation `95edd82`; hashes and commands are recorded in feature evidence

Risks / non-goals:

- `Cause` remains an explicit raw diagnostic and must not be logged or displayed
  without consumer-controlled redaction; `Error()` remains redaction-safe.
- No downstream consumer schema or pin is fabricated. Workflow stays
  `in_progress` and unreleased.

### 2026-09-03 12:15 CDT — Harden cancellation, callback identity, I/O, and allocation

Implementation commit: `5ec6f2f759078fa7fe894f99ff1c245a6c67e6f4`.
Evidence commit: `ca9de4dfa87334fc4dd9929649e48c783bd466c6`.
The heading uses that evidence commit's verified author time.

Affected files:

- `README.md`
- `docs/CHANGELOG.md`
- `docs/architecture.md`
- `docs/implementation-spec.md`
- `docs/performance.md`
- `docs/runtime-boundary.md`
- `docs/verification.md`
- `pkg/portability/boundary_test.go`
- `pkg/portability/checkpoint.go`
- `pkg/portability/checkpoint_test.go`
- `pkg/portability/errors.go`
- `pkg/portability/export.go`
- `pkg/portability/export_test.go`
- `pkg/portability/failure_paths_test.go`
- `pkg/portability/fuzz_test.go`
- `pkg/portability/hardening_test.go`
- `pkg/portability/import.go`
- `pkg/portability/import_test.go`
- `pkg/portability/performance_test.go`
- `pkg/portability/public_api_test.go`
- `pkg/portability/streaming_test.go`
- `pkg/portability/types.go`
- `pkg/portability/values_test.go`
- `pkg/portability/wire.go`
- `workflow/COVERAGE.md`
- `workflow/features/portable-v1/evidence/verification.md`

Explanation:

Close independent-review findings by adding contexts to I/O-performing
constructors and finalization, checking cancellation around all caller-owned
I/O and callbacks, bounding best-effort abort cleanup, and treating canceled
commits as unknown outcomes. Callback causes are now available only through
`Causer` and cannot forge library sentinels through `errors.Is`. Exact-read
logic validates Reader counts and tolerates a documented finite number of
transient empty reads; exact writes no longer retry positive short nil results.
Payload buffers are allocated only for the first non-empty record and reused.
Fuzz oracles now validate completion state and valid-archive mutations.

Verification:

- focused cancellation, callback-spoofing, I/O-contract, and allocation tests
- full format, vet, build, race x50, 92.9% coverage, four fuzz targets, and the
  export/import allocation matrix passed against exact source `5ec6f2f`; hashes
  and commands are recorded in the feature evidence

Risks / non-goals:

- This deliberately changes the unreleased Go API; there is no released caller
  compatibility contract to preserve.
- No consumer schema or downstream integration was invented. Workflow remains
  `in_progress`, and no release or admission is claimed.

### 2026-09-03 11:38 CDT — Bind the external-consumer proof

Commit: `66feed0484b811ca57be090459ccfef7a392439f`

Affected files:

- `docs/CHANGELOG.md`
- `workflow/features/portable-v1/evidence/verification.md`

Explanation:

Replace the insufficient external-consumer log with a command-traced proof that
prints and hashes the consumer module, resolves its local replacement, binds the
replacement repository to clean detached source `5a291d9`, and runs race, vet,
and build.

Verification:

- retained proof log SHA-256 and direct evidence audit

Risks / non-goals:

- Evidence-only; implementation and workflow state remain unchanged.

### 2026-09-03 11:36 CDT — Close evidence reproducibility gaps

Commit: `a6d120f95f9cd8e359731f79584773575c1a2da5`

Affected files:

- `docs/CHANGELOG.md`
- `workflow/features/portable-v1/evidence/verification.md`

Explanation:

Record exact fuzz commands and retain hashed, command-traced clean-clone and
external-consumer logs after the first evidence audit rejected unverifiable
prose. This is bookkeeping over the unchanged hardened source.

Verification:

- direct SHA-256 comparison of retained logs
- fresh evidence-only Judge pass required after this commit

Risks / non-goals:

- No implementation, API, wire, workflow state, or release claim changes.

### 2026-09-03 11:31 CDT — Finalize revision-matched worker evidence

Commit: `72a5c318cb58c4d2b6d787cc0c66eb753e51d31e`

Affected files:

- `docs/CHANGELOG.md`
- `docs/performance.md`
- `docs/verification.md`
- `workflow/COVERAGE.md`
- `workflow/features/portable-v1/evidence/verification.md`

Explanation:

Replace superseded pre-correction measurements with exact hardened-source
coverage, race, fuzz, performance, clean-clone, external-consumer, and Graphify
evidence and cryptographic hashes. Preserve `in_progress` workflow state because
independent orchestrator admission remains deliberately separate.

Verification:

- revision-matched commands and hashes in feature evidence

Risks / non-goals:

- This is evidence-only and changes no implementation or compatibility surface.
- No push, PR, tag, release, or independent admission is claimed.

### 2026-09-03 11:25 CDT — Record worker verification evidence

Commit: `5a291d9752839333ba033005b21c80686836b9e3`

Affected files:

- `README.md`
- `docs/CHANGELOG.md`
- `docs/implementation-spec.md`
- `docs/performance.md`
- `docs/verification.md`
- `pkg/portability/boundary_test.go`
- `pkg/portability/checkpoint.go`
- `pkg/portability/errors.go`
- `pkg/portability/export.go`
- `pkg/portability/export_test.go`
- `pkg/portability/failure_paths_test.go`
- `pkg/portability/import.go`
- `pkg/portability/import_test.go`
- `pkg/portability/wire.go`
- `workflow/COVERAGE.md`
- `workflow/features/portable-v1/evidence/verification.md`

Explanation:

Pin empty and non-empty V1 archive bytes plus persistent checkpoint bytes with
literal golden tests. Harden resume checkpoints against impossible counters,
chains, offsets, and arithmetic overflow; expose the prior safe checkpoint even
after first-record failure; prove idempotent replay after an unknown sink commit
outcome; and distinguish redacted I/O failures from malformed or truncated
input. Record coverage, fuzz, repeated-race, external-consumer,
bounded-allocation, percentile, and revision-matched graph evidence. Keep
workflow state active because independent orchestrator admission remains
outstanding.

Verification:

- gates and artifact hashes recorded in feature evidence

Risks / non-goals:

- Performance data comes from an uncontrolled local host and establishes no
  consumer SLO.
- This evidence does not tag, release, push, or independently admit the API.

### 2026-09-03 11:04 CDT — Implement bounded resumable portability streams

Commit: `a4b39e8d20caa140d9d5dd10fdd0e5a41ad0eff4`

Affected files:

- `CONTRIBUTING.md`
- `README.md`
- `SECURITY.md`
- `docs/CHANGELOG.md`
- `docs/RELEASING.md`
- `docs/distribution.md`
- `pkg/portability/boundary_test.go`
- `pkg/portability/checkpoint.go`
- `pkg/portability/checkpoint_test.go`
- `pkg/portability/doc.go`
- `pkg/portability/errors.go`
- `pkg/portability/export.go`
- `pkg/portability/export_test.go`
- `pkg/portability/failure_paths_test.go`
- `pkg/portability/fuzz_test.go`
- `pkg/portability/import.go`
- `pkg/portability/import_test.go`
- `pkg/portability/performance_test.go`
- `pkg/portability/public_api_test.go`
- `pkg/portability/streaming_test.go`
- `pkg/portability/types.go`
- `pkg/portability/values_test.go`
- `pkg/portability/wire.go`

Explanation:

Implement wire-format V1 for consumer-owned opaque records. Export and import
stream through fixed buffers, validate bounded metadata and payload sizes,
carry a rolling integrity chain across persistent record-boundary checkpoints,
and require a valid footer followed by EOF for completeness. Import uses
consumer-owned staged sinks and commits only after record integrity succeeds.
Stable errors omit payload and consumer error text.

Verification:

- focused unit, negative, exhaustive truncation, boundary, external-package,
  bounded-write, and fuzz seed tests
- `go vet -mod=readonly ./...`
- `go test -mod=readonly -race ./...`
- `go build -mod=readonly ./...`

Risks / non-goals:

- This remains unreleased and independently unadmitted.
- SHA-256 integrity is not producer authentication.
- Consumer policy, persistence, idempotency, storage positioning, encryption,
  compression, and transport remain outside the library.

### 2026-09-03 10:49 CDT — Define the portable streaming contract

Commit: `d827d1b76c7f3f6c0a3ddc965a15bfadb0fddbed`

Affected files:

- `.gitignore`
- `.go-version`
- `LICENSE`
- `Makefile`
- `docs/CHANGELOG.md`
- `docs/architecture.md`
- `docs/feature-plan.md`
- `docs/implementation-spec.md`
- `docs/performance.md`
- `docs/prd.md`
- `docs/runtime-boundary.md`
- `docs/verification.md`
- `go.mod`
- `workflow.events.jsonl`
- `workflow.toml`
- `workflow/COVERAGE.md`
- `workflow/README.md`
- `workflow/features/portable-v1/README.md`

Explanation:

Define the consumer-neutral V1 boundary before implementation: opaque records,
bounded streaming, staged import, per-record and final integrity, exact
record-boundary checkpoints, and explicit completeness. Select the
maintainer-approved MIT license.

Verification:

- documentation and manifest inspection
- `git diff --check -- .`

Risks / non-goals:

- No implementation or release is claimed by this contract commit.
- Application schemas and product policy remain consumer-owned.

### 2026-09-03 01:07 CDT — Normalize release-policy formatting

Commit: `64ac2f3f52f05644c1e8584b3a9044d5dda1cdd1`

Affected files:

- `docs/RELEASING.md`

Explanation:

Normalize Markdown formatting in the existing release policy. This commit does
not introduce or change distribution behavior, repository ownership, release
authority, or API maturity.

Verification:

- exact named-commit file audit
- documentation diff inspection

Risks / non-goals:

- No license is selected.
- No existing tag is changed and no new release is created.
- Mirror direction, repository ownership, and account type are unchanged.
