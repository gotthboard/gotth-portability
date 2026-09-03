# Changelog

This repository records user-visible and compatibility-relevant changes here.
Released sections use Semantic Versioning; unreleased work remains under
`Unreleased` and does not imply a tag.

## Unreleased

### 2026-09-03 — Harden cancellation, callback identity, I/O, and allocation

Implementation commit: `5ec6f2f759078fa7fe894f99ff1c245a6c67e6f4`.
Evidence commit: current commit; hash assigned by Git after commit.

Affected files:

- `pkg/portability/**`
- `README.md`, `docs/architecture.md`, `docs/implementation-spec.md`
- `docs/runtime-boundary.md`, `docs/performance.md`, `docs/verification.md`

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
- full format, vet, build, race, coverage, fuzz, and benchmark gates pending on
  the resulting exact source commit

Risks / non-goals:

- This deliberately changes the unreleased Go API; there is no released caller
  compatibility contract to preserve.
- No consumer schema or downstream integration was invented. Workflow remains
  `in_progress`, and no release or admission is claimed.

### 2026-09-03 14:20 CDT — Bind the external-consumer proof

Commit: current commit; hash assigned by Git after commit

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

### 2026-09-03 14:05 CDT — Close evidence reproducibility gaps

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

### 2026-09-03 13:25 CDT — Finalize revision-matched worker evidence

Commit: `72a5c318cb58c4d2b6d787cc0c66eb753e51d31e`

Affected files:

- `docs/performance.md`, `docs/verification.md`, `docs/CHANGELOG.md`
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

### 2026-09-03 12:40 CDT — Record worker verification evidence

Commit: `5a291d9752839333ba033005b21c80686836b9e3`

Affected files:

- `pkg/portability/**`
- `docs/performance.md`, `docs/verification.md`
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

### 2026-09-03 12:10 CDT — Implement bounded resumable portability streams

Commit: `a4b39e8d20caa140d9d5dd10fdd0e5a41ad0eff4`

Affected files:

- `pkg/portability/**`
- `README.md`, `CONTRIBUTING.md`, `SECURITY.md`
- `docs/distribution.md`, `docs/RELEASING.md`

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

### 2026-09-03 10:45 CDT — Define the portable streaming contract

Commit: `d827d1b76c7f3f6c0a3ddc965a15bfadb0fddbed`

Affected files:

- `LICENSE`, `.go-version`, `go.mod`, `Makefile`
- `docs/prd.md`, `docs/architecture.md`, `docs/implementation-spec.md`
- `docs/feature-plan.md`, `docs/runtime-boundary.md`
- `docs/performance.md`, `docs/verification.md`
- `workflow.toml`, `workflow/**`, `workflow.events.jsonl`

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

### 2026-09-03 00:42 CDT — Establish GitHub public distribution

Commit: `64ac2f3f52f05644c1e8584b3a9044d5dda1cdd1`

Affected files:

- `README.md`
- `CONTRIBUTING.md`
- `SECURITY.md`
- `docs/distribution.md`
- `docs/RELEASING.md`

Explanation:

Declare GitHub as the public distribution endpoint while retaining Forgejo as
canonical development, define maturity and support honestly, and document the
independent release process. The reserved namespace remains a documentation-only placeholder and makes no API or release claim.

Verification:

- exact old-import search
- documentation contract audit

Risks / non-goals:

- No license is selected.
- No existing tag is changed and no new release is created.
- Mirror direction, repository ownership, and account type are unchanged.
