# Implementation specification

## Wire version 1

All integers are unsigned big-endian. The header is the eight-byte magic
`GOTTHP1\n`, wire version (`u16`), archive-ID length/value (`u16`), schema-ID
length/value (`u16`), and schema version (`u32`). A record frame is type `0x01`,
sequence (`u64`), kind length/value (`u16`), key length/value (`u16`), payload
length (`u64`), payload bytes, and SHA-256 payload digest. The footer is type
`0xff`, record count (`u64`), payload-byte count (`u64`), and rolling chain.

Archive ID, schema ID, and record kind are non-empty; record key may be empty.
All string lengths count encoded bytes, not Unicode code points. Strings must be
valid UTF-8 and contain no NUL byte. The first record sequence is zero and each
subsequent sequence is exactly the previous value plus one; consequently the
next sequence equals the number of committed records.

The initial chain is SHA-256 of the exact encoded header. Each next chain is
SHA-256 of the previous chain plus the canonical record frame metadata and
payload digest. The footer itself is not chained. EOF immediately after a
valid footer is required.

## Limits

Archive ID and schema ID are at most 256 bytes, record kind 128 bytes, and key
1024 bytes. UTF-8 is required and NUL is rejected. Caller-supplied `Limits`
bound record count, individual payload bytes, and total payload bytes; zero
selects documented defaults. Each exporter/importer lazily allocates one fixed
32 KiB buffer for its first non-empty record and reuses it. Empty records do not
allocate the payload buffer, but still incur measured metadata, hashing, and
callback allocations.

## Public contracts

- `NewExporter` writes and hashes one header; `WriteRecord` streams exactly the
  declared payload length and returns a boundary checkpoint; `Finalize` writes
  one footer and returns the manifest. All three operations accept a context
  and check cancellation before and after caller-owned I/O. `Exporter.Checkpoint`
  returns the last safe boundary even when the first record fails after partial
  output.
- `ResumeExporter` restores the header, counters, chain, and offset from a
  validated checkpoint; it trusts the caller to position an append target at
  that exact offset.
- `NewImporter` and `ResumeImporter` accept a context and compatibility-check
  the header before any sink is opened. `Compatibility`, `Sink.Begin`, staged
  writes, `Commit`, and all reads are cancellation-bracketed. `Importer.Checkpoint`
  returns the last committed boundary even after an unknown sink-commit outcome.
  `ResumeImporter` restores validated boundary state and trusts the caller to
  position input at the checkpoint offset.
- `Next` stages, streams, validates, and commits one record. It returns
  `ErrComplete` only after footer and EOF validation. `Manifest` fails until
  that explicit completion state exists.
- On a live exporter, `WriteRecord` nil context/body, entry-cancellation,
  metadata, limit, and offset preflight failures happen before record output
  and leave it reusable. On a live importer, `Next` nil context/sink and
  entry-cancellation preflight failures consume no frame and leave it reusable.
  Once preflight passes and record/frame I/O begins, any non-completion failure
  poisons that object even if cancellation prevents the first callback;
  recovery resumes from its prior checkpoint.
- Failures before commit call `Abort` with a cancellation-independent context
  whose deadline is `AbortTimeout`. A commit error, or cancellation observed
  after a commit call, has unknown outcome and does not call `Abort` or advance
  the checkpoint.
- `Sink.Begin` may return a non-nil stage with an error. The begin error remains
  primary and the library runs bounded `Abort` on that stage. Cancellation
  observed after the callback adds `ErrIO` without discarding `ErrSink` or its
  raw cause. Cleanup context expiry adds `ErrSink`, including when `Abort`
  returns nil after the deadline.
- Checkpoints have a versioned binary encoding protected by SHA-256. Decoding
  rejects corruption, unknown versions, and structurally inconsistent state.

## Deterministic errors

Public sentinels distinguish invalid arguments, malformed/truncated stream, I/O
failure, incompatibility, limit overflow, integrity failure, sequence mismatch,
sink failure, incomplete state, finalized state, and clean completion. Wrapped
errors never include payload or callback text. `errors.Is` traverses only the
library classification; callers may explicitly retrieve the raw underlying
error through `Causer.Cause` without letting callback values spoof library
sentinels. Raw causes may contain sensitive application data and must be
consumer-redacted before logging or display.
