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
Checksum, stream, or pre-commit cancellation calls `Abort` with a fresh cleanup
context bounded by `AbortTimeout`. Sink implementations must honor that
deadline, make `Commit` idempotent by archive identity and sequence, and treat
a commit error or cancellation observed after `Commit` as potentially having
taken effect. The library cannot resolve a consumer backend's unknown commit
outcome and therefore neither aborts nor advances its checkpoint then.
If `Commit` both returns an error and leaves the operation context canceled,
the sink error remains the primary `ErrSink` classification and cancellation is
retained as an additional `ErrIO` classification.
If `Compatibility` returns an error while canceling, incompatibility remains
primary and cancellation is additional. A staged `Write` failure follows the
same rule with primary `ErrSink`. Both cases retain both explicit raw causes.
If `Begin` returns both a stage and an error, the begin error remains primary
and the stage receives bounded `Abort`. Cancellation observed after that
callback is retained as an additional `ErrIO` classification rather than
discarded. A nil stage returned without an error is still `ErrSink` when
cancellation adds `ErrIO`. An expired cleanup context is reported as an
additional `ErrSink` failure even when `Abort` returns nil.

## Failure model

Malformed, truncated, incompatible, over-limit, corrupt, out-of-sequence, and
trailing input fails closed under stable sentinels. The library emits no logs
and error text contains operation names, never payloads or callback text.
Raw underlying causes are available only through `Causer`, not sentinel
traversal, so a callback returning (for example) `ErrComplete` cannot forge
completion. Those causes may contain sensitive consumer data and require
consumer-controlled redaction before logging or display. A successful record
checkpoint is not proof that the archive is complete; only `Manifest` after a
library-produced `ErrComplete` is the completeness oracle.
