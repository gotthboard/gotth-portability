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
selects documented defaults. Copying uses one fixed 32 KiB buffer.

## Public contracts

- `NewExporter` writes and hashes one header; `WriteRecord` streams exactly the
  declared payload length and returns a boundary checkpoint; `Finalize` writes
  one footer and returns the manifest. `Exporter.Checkpoint` returns the last
  safe boundary even when the first record fails after partial output.
- `ResumeExporter` restores the header, counters, chain, and offset from a
  validated checkpoint; it trusts the caller to position an append target at
  that exact offset.
- `NewImporter` reads and compatibility-checks the header before any sink is
  opened. `Importer.Checkpoint` returns the last committed boundary even after
  an unknown sink-commit outcome. `ResumeImporter` restores validated boundary
  state and trusts the caller to position input at the checkpoint offset.
- `Next` stages, streams, validates, and commits one record. It returns
  `ErrComplete` only after footer and EOF validation. `Manifest` fails until
  that explicit completion state exists.
- Checkpoints have a versioned binary encoding protected by SHA-256. Decoding
  rejects corruption, unknown versions, and structurally inconsistent state.

## Deterministic errors

Public sentinels distinguish invalid arguments, malformed/truncated stream, I/O
failure, incompatibility, limit overflow, integrity failure, sequence mismatch,
sink failure, incomplete state, finalized state, and clean completion. Wrapped
errors retain a cause but never include payload content.
