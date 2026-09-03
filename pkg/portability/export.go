package portability

import (
	"context"
	"crypto/sha256"
	"hash"
	"io"
)

// Exporter streams one archive to a caller-owned writer. It is not safe for
// concurrent use.
type Exporter struct {
	w      io.Writer
	limits Limits
	cp     Checkpoint
	buffer []byte
	done   bool
}

// NewExporter validates the contract and writes exactly one archive header.
// Complexity: time O(a+s)+W(a+s), Omega(1), with tight
// Theta(a+s)+W(a+s) on a valid successful path; auxiliary space O(a+s),
// Omega(1), with tight Theta(a+s) on a valid successful path. Variables a/s
// are identifier lengths; W is delegated writer cost.
func NewExporter(ctx context.Context, w io.Writer, header Header, limits Limits) (*Exporter, error) {
	if ctx == nil || w == nil {
		return nil, wrap(ErrInvalid, "context or writer", nil)
	}
	if err := header.validate(); err != nil {
		return nil, err
	}
	normalized, err := limits.normalize()
	if err != nil {
		return nil, err
	}
	encoded := encodeHeader(header)
	n, err := writeExactContext(ctx, w, encoded)
	if err != nil {
		return nil, wrap(ErrIO, "write header", err)
	}
	chain := sha256.Sum256(encoded)
	return &Exporter{w: w, limits: normalized, cp: Checkpoint{header: header, offset: uint64(n), chain: chain}}, nil
}

// ResumeExporter restores a committed boundary. The caller must position and,
// after a failed prior attempt, truncate the writer to checkpoint.Offset.
// Complexity: time O(a+s), Omega(1), with tight Theta(a+s) for a non-nil
// Writer and valid checkpoint; auxiliary space O(a+s), Omega(1), with tight
// Theta(a+s) on that path because checkpoint validation reconstructs the
// canonical header. Nil-Writer and constant-time invalid-limit/checkpoint
// paths can take tight Theta(1); variables a/s are header identifier lengths.
func ResumeExporter(w io.Writer, checkpoint Checkpoint, limits Limits) (*Exporter, error) {
	if w == nil {
		return nil, wrap(ErrInvalid, "writer", nil)
	}
	normalized, err := limits.normalize()
	if err != nil {
		return nil, err
	}
	if err := checkpoint.validate(normalized); err != nil {
		return nil, err
	}
	return &Exporter{w: w, limits: normalized, cp: checkpoint}, nil
}

// Checkpoint returns the last fully written record boundary, including the
// initial header boundary and the prior safe boundary after a poisoned write.
// Complexity: time and auxiliary space O(1), Omega(1), tight Theta(1); strings
// share immutable Go string storage.
func (e *Exporter) Checkpoint() Checkpoint {
	if e == nil {
		return Checkpoint{}
	}
	return e.cp
}

// WriteRecord streams one exact-length record and returns its committed
// boundary. On a live exporter, nil context/body, entry-cancellation, metadata,
// limit, and offset preflight failures occur before record output and leave it
// reusable. Once preflight passes and record-frame I/O begins, any failure
// poisons the exporter, even if cancellation prevents the first Writer callback;
// recover from the prior checkpoint.
// Complexity: general record-path time O(k+r+n)+RB(n,E)+WA(k+r,n), Omega(1),
// with tight Theta(k+r+n)+RB(n,E)+WA(k+r,n) on success; preflight rejection
// can take tight Theta(1). RB is aggregate Body Reader cost for the declared n
// bytes plus the required EOF probe, including at most E consecutive empty
// reads. WA is aggregate archive Writer cost for the Theta(k+r) prefix, payload
// chunks totaling n bytes, and fixed 32-byte digest; thus empty payloads still
// write prefix and digest. Local auxiliary space is O(k+r+B), Omega(1), tight
// Theta(k+r+B) on successful paths, where B is the fixed 32KiB working buffer
// for a non-empty record and zero for an empty record; only the first non-empty
// record allocates and retains B. Delegated Reader/Writer auxiliary space is
// ARB(n,E)+AWA(k+r,n). Variables k/r are metadata byte lengths,
// n is payload size, and E=MaxConsecutiveEmptyReads.
func (e *Exporter) WriteRecord(ctx context.Context, record Record) (Checkpoint, error) {
	if e == nil || e.done {
		return Checkpoint{}, wrap(ErrFinalized, "write record", nil)
	}
	if ctx == nil {
		return Checkpoint{}, wrap(ErrInvalid, "context", nil)
	}
	if err := ctx.Err(); err != nil {
		return Checkpoint{}, wrap(ErrIO, "write record", err)
	}
	meta := RecordMeta{Sequence: e.cp.nextSequence, Kind: record.Kind, Key: record.Key, Size: record.Size}
	if record.Body == nil {
		return Checkpoint{}, wrap(ErrInvalid, "record body", nil)
	}
	if err := meta.validate(); err != nil {
		return Checkpoint{}, err
	}
	if e.cp.records >= e.limits.MaxRecords || record.Size > e.limits.MaxRecordBytes {
		return Checkpoint{}, wrap(ErrLimit, "record", nil)
	}
	total, overflow := checkedAdd(e.cp.payloadBytes, record.Size)
	if overflow || total > e.limits.MaxTotalBytes {
		return Checkpoint{}, wrap(ErrLimit, "total payload", nil)
	}
	prefix := encodeRecordPrefix(meta)
	frameBytes, overflow := checkedAdd(uint64(len(prefix)), record.Size)
	if !overflow {
		frameBytes, overflow = checkedAdd(frameBytes, sha256.Size)
	}
	nextOffset, offsetOverflow := checkedAdd(e.cp.offset, frameBytes)
	if overflow || offsetOverflow {
		return Checkpoint{}, wrap(ErrLimit, "stream offset", nil)
	}
	if _, err := writeExactContext(ctx, e.w, prefix); err != nil {
		e.done = true
		return Checkpoint{}, wrap(ErrIO, "write record metadata", err)
	}
	digest, err := e.copyPayload(ctx, record.Body, record.Size)
	if err != nil {
		e.done = true
		return Checkpoint{}, err
	}
	if _, err := writeExactContext(ctx, e.w, digest[:]); err != nil {
		e.done = true
		return Checkpoint{}, wrap(ErrIO, "write record digest", err)
	}
	e.cp.chain = nextChain(e.cp.chain, prefix, digest)
	e.cp.records++
	e.cp.nextSequence++
	e.cp.payloadBytes = total
	e.cp.offset = nextOffset
	return e.cp, nil
}

// copyPayload writes and hashes exactly size bytes, then requires source EOF.
// Complexity: general time O(n)+RB(n,E)+WP(n), Omega(1), with tight
// Theta(n)+RB(n,E)+WP(n) on success. RB is aggregate Body Reader cost for n
// declared bytes plus the required EOF probe; WP is aggregate archive Writer
// cost for payload chunks totaling n bytes and is zero when n=0. Peak local
// auxiliary space is O(B), Omega(1), tight Theta(B) for a non-empty payload and
// Theta(1) for an empty payload, where B is the fixed 32KiB working buffer; only
// the first non-empty record allocates and retains B. Delegated auxiliary space
// is ARB(n,E)+AWP(n). Variable n is size and E=MaxConsecutiveEmptyReads.
func (e *Exporter) copyPayload(ctx context.Context, src io.Reader, size uint64) ([32]byte, error) {
	h := sha256.New()
	var buf []byte
	if size > 0 {
		if e.buffer == nil {
			e.buffer = make([]byte, copyBufferBytes)
		}
		buf = e.buffer
	}
	remaining := size
	emptyReads := 0
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return [32]byte{}, wrap(ErrIO, "read payload", err)
		}
		want := uint64(len(buf))
		if remaining < want {
			want = remaining
		}
		n, err := src.Read(buf[:int(want)])
		if n < 0 || n > int(want) {
			return [32]byte{}, wrap(ErrIO, "read payload", io.ErrShortBuffer)
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return [32]byte{}, wrap(ErrIO, "read payload", ctxErr)
		}
		if n > 0 {
			emptyReads = 0
			if _, writeErr := writeExactContext(ctx, e.w, buf[:n]); writeErr != nil {
				return [32]byte{}, wrap(ErrIO, "write payload", writeErr)
			}
			_, _ = h.Write(buf[:n])
			remaining -= uint64(n)
		}
		if err != nil {
			if err == io.EOF && remaining == 0 {
				break
			}
			if err == io.EOF {
				return [32]byte{}, wrap(ErrTruncated, "read payload", err)
			}
			return [32]byte{}, wrap(ErrIO, "read payload", err)
		}
		if n == 0 && err == nil {
			emptyReads++
			if emptyReads > MaxConsecutiveEmptyReads {
				return [32]byte{}, wrap(ErrIO, "read payload", io.ErrNoProgress)
			}
		}
	}
	n, err := probeEOFContext(ctx, src)
	if n != 0 {
		return [32]byte{}, wrap(ErrInvalid, "payload length", nil)
	}
	if err != io.EOF {
		return [32]byte{}, wrap(ErrIO, "check payload boundary", err)
	}
	var digest [32]byte
	copy(digest[:], h.Sum(nil))
	return digest, nil
}

// Finalize writes the final manifest exactly once.
// Complexity: general time O(1)+W(1), Omega(1); a live, non-nil-context,
// non-canceled path reaches one fixed-size Writer call and has tight
// Theta(1)+W(1), while rejection paths skip W and take tight Theta(1).
// Auxiliary space is O(1), Omega(1), tight Theta(1); W is the cost of zero or
// one delegated Writer call.
func (e *Exporter) Finalize(ctx context.Context) (Manifest, error) {
	if e == nil || e.done {
		return Manifest{}, wrap(ErrFinalized, "finalize", nil)
	}
	if ctx == nil {
		return Manifest{}, wrap(ErrInvalid, "context", nil)
	}
	if err := ctx.Err(); err != nil {
		return Manifest{}, wrap(ErrIO, "finalize", err)
	}
	e.done = true
	manifest := Manifest{Records: e.cp.records, PayloadBytes: e.cp.payloadBytes, Chain: e.cp.chain}
	if _, err := writeExactContext(ctx, e.w, encodeFooter(manifest)); err != nil {
		return Manifest{}, wrap(ErrIO, "write manifest", err)
	}
	return manifest, nil
}

// nextChain hashes the previous chain, canonical metadata, and payload digest.
// Complexity: time O(m), Omega(m), tight Theta(m); auxiliary space O(1),
// Omega(1), tight Theta(1) excluding fixed hash state; variable m is
// len(prefix); delegated SHA-256 cost is linear in its input.
func nextChain(previous [32]byte, prefix []byte, digest [32]byte) [32]byte {
	h := sha256.New()
	writeHash(h, previous[:])
	writeHash(h, prefix)
	writeHash(h, digest[:])
	var next [32]byte
	copy(next[:], h.Sum(nil))
	return next
}

// writeHash centralizes writes whose hash.Hash contract cannot fail.
// Complexity: delegated time H(n), Omega(1), and tight Theta(n) for SHA-256;
// auxiliary space O(1), Omega(1), tight Theta(1); variable n is len(p).
func writeHash(h hash.Hash, p []byte) { _, _ = h.Write(p) }
