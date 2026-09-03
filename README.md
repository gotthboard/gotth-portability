# gotth-portability

> **Distribution:** GitHub is the public clone, and future release endpoint.
> Forgejo remains canonical development and the issue/contribution location.
> See [the distribution contract](docs/distribution.md).


Reserved for reusable import, export, and data-portability mechanics shared by
GOTTH applications.

## Intended boundary

This project may eventually own versioned interchange envelopes, streaming and
resumable transfer, integrity manifests, compatibility validation, importer
checkpoints, and redaction-safe tooling. Consumers retain data ownership,
authorization, retention, product schema mapping, and the decision about what
may leave or enter an application.

## Non-goals

- A universal application schema or unrestricted database dump.
- Bypassing consumer authorization, legal hold, or retention policy.
- Silent lossy conversion or unbounded in-memory transfer.

## Status

Placeholder only. There is no implementation, API, release, tag, compatibility
promise, or dependency to pin.

## Installation, compatibility, and support

Planned placeholder only. There is no implementation, API, support promise, or release.

There is nothing to install or import. Do not add this repository as a
dependency.

The repository has no selected license and no long-term support promise.
Versioning, release admission, security reporting, and contribution details are
in [the release policy](docs/RELEASING.md), [security policy](SECURITY.md), and
[contribution guide](CONTRIBUTING.md).
