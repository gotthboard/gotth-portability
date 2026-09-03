# gotth-portability

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
