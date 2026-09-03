# Changelog

This repository records user-visible and compatibility-relevant changes here.
Released sections use Semantic Versioning; unreleased work remains under
`Unreleased` and does not imply a tag.

## Unreleased

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
