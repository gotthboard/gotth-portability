# Architecture

## Boundary

The sole public package is `pkg/portability`. It knows record kind, key, and
opaque payload bytes. The consumer supplies schema identity/version and decides
which records may leave or enter the application. No application schema or
policy is embedded here.

## Stream

An archive is a header, zero or more record frames, and one manifest footer.
The header carries the wire version plus caller-owned archive and schema
identities. Each record carries a monotonically increasing sequence, bounded
metadata, payload length, streamed payload, and SHA-256 payload digest. The
footer carries record count, total payload bytes, and a rolling SHA-256 chain
over record metadata and payload digests.

The rolling chain is the resume mechanism: a checkpoint stores the last
committed chain, counts, next sequence, and exact stream offset. Resumption
continues at a frame boundary without rereading or buffering prior payloads.
The caller owns positioning and truncating external storage to the checkpoint
offset.

## Import commit boundary

An importer asks the consumer for a staged `RecordSink`, streams exactly one
payload into it, validates the payload digest, and only then calls `Commit`.
Checksum or stream failure calls `Abort`. Sink implementations must make
`Commit` idempotent by archive identity and sequence and must treat a commit
error as potentially having taken effect. The library cannot resolve a
consumer backend's unknown commit outcome.

## Failure model

Malformed, truncated, incompatible, over-limit, corrupt, out-of-sequence, and
trailing input fails closed under stable sentinels. The library emits no logs
and error text contains field names and positions, never payloads. A successful
record checkpoint is not proof that the archive is complete; only `Manifest`
after `ErrComplete` is the completeness oracle.
