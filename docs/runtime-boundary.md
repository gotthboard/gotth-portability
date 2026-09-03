# Runtime boundary contract

## Supported target

- Go 1.26.6, Linux amd64, standard library only.
- Wire format version 1 as specified in this repository.
- SHA-256 from Go's `crypto/sha256`; the archive stores only 32-byte digest
  outputs and does not depend on serializing runtime hash state.

## Authoritative contracts

- `io.Reader` may return data and an error together; copy loops account for
  both and never assume one read fills a buffer.
- `io.Writer` may short-write and must return a non-nil error when doing so;
  exact-write helpers nevertheless reject any unexplained short write.
- `crypto/sha256.Sum256` returns the standardized 32-byte SHA-256 digest.
- `context.Context` cancellation is checked between bounded reads/writes; it
  cannot forcibly interrupt a misbehaving consumer `Reader`, `Writer`, or sink.

## Limits and completeness

Length fields are checked before conversion to `int` or `int64`. Tests cover
configured limits at limit-1, limit, limit+1, and materially beyond where
representable. Per-record digest, rolling chain, counts, manifest footer, and
required EOF are independent completeness oracles. Successful I/O alone is
not treated as complete.

The package changes no process, filesystem, network, database, session, or
global runtime state. Restoration and pooled-state leakage tests are therefore
not applicable.
