# Changelog

This repository records user-visible and compatibility-relevant changes here.
Released sections use Semantic Versioning; unreleased work remains under
`Unreleased` and does not imply a tag.

## Unreleased

### 2026-09-03 12:10 CDT — Implement bounded resumable portability streams

Commit: current commit; hash assigned by Git after commit

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

Commit: current commit; hash assigned by Git after commit

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

Commit: current commit; hash assigned by Git after commit

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
