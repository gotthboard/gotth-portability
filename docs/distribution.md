# Distribution Contract

## Endpoints

- Canonical development and change tracking:
  <https://git.dannyhunn.com/agents/gotth-portability>
- Public clone and, only after independent implementation admission, future releases:
  <https://github.com/gotthboard/gotth-portability>

Forgejo pushes one way to GitHub. GitHub does not feed commits or tags back to
Forgejo. A ref is distributed only when the exact object ID is visible at both
endpoints.

## Maturity and compatibility

Current status: unreleased feature implementation awaiting independent
admission. The API and wire format are not yet a compatibility promise.

## Installation

There is no version to install or pin. Source exists on an unreleased feature
branch; that is not a supported distribution artifact.

The repository pins Go 1.26.6 where a Go module exists. Supported protocol,
runtime, database, and tool versions remain the ones stated in the README and
project verification documents; this distribution change does not widen those
contracts.

## Licensing gate

The maintainer selected the MIT license and the repository contains `LICENSE`.
Release publication remains blocked on independent admission, consumer pinning,
and the verification defined by the release policy.

## Migration traceability

| Requirement | Repository implementation | Verification |
| --- | --- | --- |
| DIST-001 | Existing history, tags, worktrees, and mirror direction remain unchanged | pinned ref and worktree inventory |
| DIST-005 | Unreleased implementation claims no admitted version or release | tracked-tree and README audit |
| DIST-003/004 | README, contribution, security, changelog, and release contracts describe public use and support | documentation audit |
| DIST-006 | Maintainer-selected MIT license is explicit | license inventory |
| DIST-008 | Forgejo remains source and GitHub remains the one-way mirror target | push-mirror configuration and exact ref comparison |
