# Product requirements

## Problem

GOTTH applications need a common way to move application-owned records without
turning a database dump or one product's schema into a reusable-library
contract. A transfer must be streamable, integrity checked, compatibility
checked before records are applied, and resumable at committed record
boundaries.

## Requirements

- `PORT-001`: Encode opaque consumer records in a versioned streaming format.
- `PORT-002`: Validate format and consumer schema compatibility before applying
  any record.
- `PORT-003`: Bound metadata, record count, individual payload size, cumulative
  payload size, and internal copy buffers.
- `PORT-004`: Verify each payload and a final archive manifest without buffering
  the whole archive.
- `PORT-005`: Export and import checkpoints at record boundaries, with enough
  state to resume integrity calculation from an exact byte offset.
- `PORT-006`: Apply imported payloads through consumer-owned staged sinks; admit
  a record only after its payload checksum passes.
- `PORT-007`: Expose an explicit finalization/completeness oracle and reject
  truncation, inconsistent manifests, and trailing bytes.
- `PORT-008`: Return stable error identities without logging or embedding record
  payloads in diagnostics.
- `PORT-009`: Keep application schema, authorization, disclosure, redaction,
  retention, legal hold, persistence, and release policy consumer-owned.
- `PORT-010`: Publish the source under the maintainer-selected MIT license.

## Non-goals

- A GOTTH Board or forum record schema.
- Database access, archive compression, encryption, transport, or storage.
- Authorization, retention, legal-hold, disclosure, or redaction decisions.
- Atomicity across multiple records or exactly-once consumer effects.
- Seeking, truncating, or validating the caller's underlying storage during
  resume.

## Acceptance

Requirements trace to architecture, specification, code, focused negative and
boundary tests, fuzz tests, coverage evidence, external-consumer compilation,
performance admission, and cold review. Final admission remains an independent
orchestrator decision.
