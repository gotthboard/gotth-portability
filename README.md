# gotth-portability

> **Distribution:** GitHub is the public clone and, only if implementation is
> admitted later, the future release endpoint.
> Forgejo remains canonical development and the issue/contribution location.
> See [the distribution contract](docs/distribution.md).


Consumer-neutral import, export, and data-portability mechanics shared by GOTTH
applications.

## Intended boundary

This project owns a versioned binary interchange envelope, bounded streaming,
record-boundary resume checkpoints, per-record SHA-256 integrity, a final
rolling integrity manifest, compatibility validation, and redaction-safe error
classification. Consumers retain data ownership, authorization, retention,
product schema mapping, and the decision about what may leave or enter an
application.

## Guarantees

- The entire archive is never buffered. The first non-empty payload lazily
  allocates one fixed 32 KiB buffer per exporter/importer and later records
  reuse it; empty records allocate no payload buffer.
- Empty records still incur measured frame metadata, hash-state, and callback
  allocations; the performance report quantifies them instead of calling them
  allocation-free.
- Metadata, record count, individual payload size, and cumulative payload size
  are bounded before consumer staging.
- Imported data is staged through a consumer sink and committed only after the
  record payload digest passes.
- A checkpoint identifies an exact committed record boundary and preserves the
  rolling integrity state needed to resume.
- Only a valid footer, matching record/byte counts and rolling chain, followed
  immediately by EOF makes `Importer.Manifest` succeed.
- Stable sentinels classify failures. Library-generated error strings never
  include payload bytes or consumer error text, callback causes cannot spoof a
  sentinel through `errors.Is`, and the library emits no logs. Callers that
  need the raw underlying cause can explicitly use `Causer`. That cause is
  potentially sensitive and must be consumer-redacted before logging or display.

## Deliberate limits

- SHA-256 detects accidental or adversarial modification only when the expected
  archive/checkpoint came through a trusted channel. It does not authenticate a
  producer. Sign or authenticate archives at the consumer/transport boundary.
- Resume does not seek, truncate, or verify caller-owned storage. The caller
  must reopen the exact archive and position it at `Checkpoint.Offset()`.
- A sink commit error has an unknown outcome. Sinks must reconcile and make
  commits idempotent by archive ID and sequence before resuming.
- Cancellation is checked before and after every caller-owned I/O or callback.
  A canceled commit is also an unknown outcome. Best-effort `Abort` receives a
  fresh context bounded by `AbortTimeout`; sink implementations must honor it.
  If `Begin` returns both a stage and an error, that stage is aborted; any
  cancellation observed after the callback is also retained as `ErrIO`.
  Cleanup deadline expiry is reported as `ErrSink` even if `Abort` returns nil.
- Record commit is atomic only to the degree supplied by the consumer sink.
  There is no multi-record transaction or exactly-once external effect.
- The package does not compress, encrypt, store, transmit, authorize, redact,
  or interpret records.

## API shape

Exporters accept a context plus one `Record` at a time and return a `Checkpoint`
after each complete frame. Importers ask a consumer `Sink` to begin a staged
record, copy and verify its payload, and then commit it. `Compatibility` is
called with the operation context on the header before any sink is opened.
Constructors and `Finalize` also take contexts. `Exporter.Checkpoint` and
`Importer.Checkpoint` expose the last safe boundary even after a failed first
record, which is required for truncation/replay recovery. See the package documentation and
`docs/implementation-spec.md` for the exact V1 wire contract.

## Non-goals

- A universal application schema or unrestricted database dump.
- Bypassing consumer authorization, legal hold, or retention policy.
- Silent lossy conversion or unbounded in-memory transfer.

## Status

Implemented on an unreleased feature branch and awaiting independent admission.
There is no tag, release, stable compatibility promise, or dependency version
to pin. Wire format V1 and the Go API remain unstable until consumer pinning and
release admission finish.

## Installation, compatibility, and support

The source module targets Go 1.26.6 and currently uses only the standard
library. No release exists, so consumers must not pin an invented version or
treat the feature branch as a compatibility promise.

The repository uses the MIT license. It has no long-term support promise.
Versioning, release admission, security reporting, and contribution details
are in [the release policy](docs/RELEASING.md), [security policy](SECURITY.md),
and [contribution guide](CONTRIBUTING.md).
