# Runtime boundary contract

## Supported target

- Go 1.26.6, Linux amd64, standard library only.
- Wire format version 1 as specified in this repository.
- SHA-256 from Go's `crypto/sha256`; the archive stores only 32-byte digest
  outputs and does not depend on serializing runtime hash state.

## Authoritative contracts

- `io.Reader` may return data and an error together or transiently return
  `(0, nil)`; copy loops account for both, validate counts before slicing, and
  fail with `io.ErrNoProgress` after `MaxConsecutiveEmptyReads` empty results.
- `io.Writer` may short-write and must return a non-nil error when doing so;
  exact-write helpers make one call and reject any unexplained short write
  without retrying a potentially side-effecting callback.
- `crypto/sha256.Sum256` returns the standardized 32-byte SHA-256 digest.
- `context.Context` cancellation is checked immediately before and after every
  consumer `Reader`, `Writer`, compatibility, or sink callback. It cannot
  forcibly interrupt a misbehaving callback. `Abort` receives a fresh
  cancellation-independent context with an `AbortTimeout` deadline, which a
  conforming sink must honor; the library cannot stop a sink that ignores it.
  Once `Abort` returns, deadline expiry is reported as `ErrSink` even if its
  direct return is nil.
- When a compatibility, staged-write, begin, or commit callback both produces
  a semantic failure and leaves the context canceled, the semantic class stays
  primary and cancellation adds `ErrIO`. The library retains each raw cause
  behind `Causer` while keeping callback text out of `Error()`.

## Limits and completeness

Length fields are checked before conversion to `int` or `int64`. Tests cover
configured limits at limit-1, limit, limit+1, and materially beyond where
representable. Per-record digest, rolling chain, counts, manifest footer, and
required EOF are independent completeness oracles. Successful I/O alone is
not treated as complete.

The package changes no process, filesystem, network, database, session, or
global runtime state. Restoration and pooled-state leakage tests are therefore
not applicable.
