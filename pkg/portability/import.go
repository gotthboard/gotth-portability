package portability

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"time"
	"unicode/utf8"
)

// Importer validates and stages one archive. It is not safe for concurrent use.
type Importer struct {
	r          io.Reader
	limits     Limits
	cp         Checkpoint
	readOffset uint64
	buffer     []byte
	done       bool
	manifest   Manifest
	complete   bool
}

// NewImporter reads and validates the header and calls compatibility before
// any record can be staged.
// Complexity: time O(a+s)+R(a+s)+C, Omega(1), tight Theta(a+s)+R(a+s)+C on
// success; local auxiliary space O(a+s), Omega(1), tight Theta(a+s), plus
// delegated Reader and Compatibility working space; variables: a/s identifier
// lengths, R reader cost, C delegated compatibility cost.
func NewImporter(ctx context.Context, r io.Reader, limits Limits, compatibility Compatibility) (*Importer, error) {
	if ctx == nil || r == nil || compatibility == nil {
		return nil, wrap(ErrInvalid, "context, reader, or compatibility", nil)
	}
	normalized, err := limits.normalize()
	if err != nil {
		return nil, err
	}
	header, encoded, err := readHeader(ctx, r)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, wrap(ErrIO, "consumer schema", err)
	}
	compatibilityErr := compatibility(ctx, header)
	if compatibilityErr != nil {
		primary := wrap(ErrIncompatible, "consumer schema", compatibilityErr)
		if contextErr := ctx.Err(); contextErr != nil {
			return nil, errors.Join(primary, wrap(ErrIO, "consumer schema", contextErr))
		}
		return nil, primary
	}
	if err := ctx.Err(); err != nil {
		return nil, wrap(ErrIO, "consumer schema", err)
	}
	chain := sha256.Sum256(encoded)
	cp := Checkpoint{header: header, offset: uint64(len(encoded)), chain: chain}
	return &Importer{r: r, limits: normalized, cp: cp, readOffset: cp.offset}, nil
}

// ResumeImporter restores a validated committed boundary. The caller must
// position the reader at checkpoint.Offset.
// Complexity: general time O(a+s)+C, Omega(1); a non-nil-argument, valid-limit,
// valid-checkpoint path that reaches Compatibility has tight Theta(a+s)+C,
// while argument, limit, checkpoint, and entry-cancellation rejection can skip
// C. Local auxiliary space is O(a+s), Omega(1), with tight Theta(a+s) when
// checkpoint validation reconstructs a valid canonical header, plus delegated
// Compatibility working space only when called. Variables a/s are identifier
// lengths and C is the cost of zero or one Compatibility call.
func ResumeImporter(ctx context.Context, r io.Reader, checkpoint Checkpoint, limits Limits, compatibility Compatibility) (*Importer, error) {
	if ctx == nil || r == nil || compatibility == nil {
		return nil, wrap(ErrInvalid, "context, reader, or compatibility", nil)
	}
	normalized, err := limits.normalize()
	if err != nil {
		return nil, err
	}
	if err := checkpoint.validate(normalized); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, wrap(ErrIO, "consumer schema", err)
	}
	compatibilityErr := compatibility(ctx, checkpoint.header)
	if compatibilityErr != nil {
		primary := wrap(ErrIncompatible, "consumer schema", compatibilityErr)
		if contextErr := ctx.Err(); contextErr != nil {
			return nil, errors.Join(primary, wrap(ErrIO, "consumer schema", contextErr))
		}
		return nil, primary
	}
	if err := ctx.Err(); err != nil {
		return nil, wrap(ErrIO, "consumer schema", err)
	}
	return &Importer{r: r, limits: normalized, cp: checkpoint, readOffset: checkpoint.offset}, nil
}

// Header returns the validated archive header.
// Complexity: time and auxiliary space O(1), Omega(1), tight Theta(1); strings
// share immutable Go string storage.
func (i *Importer) Header() Header {
	if i == nil {
		return Header{}
	}
	return i.cp.header
}

// Checkpoint returns the last consumer-committed record boundary, including
// the initial header boundary and the prior boundary after any failed attempt.
// Complexity: time and auxiliary space O(1), Omega(1), tight Theta(1); strings
// share immutable Go string storage.
func (i *Importer) Checkpoint() Checkpoint {
	if i == nil {
		return Checkpoint{}
	}
	return i.cp
}

// Next validates and commits one record or returns ErrComplete after validating
// the manifest and required EOF. On a live importer, nil context/sink and
// entry-cancellation preflight failures occur before frame input and leave it
// reusable. Once preflight passes and frame I/O begins, any non-completion
// failure poisons the importer, even if cancellation prevents the first Reader
// callback; recover from the prior checkpoint.
// Complexity: general record-frame time O(k+r+n)+RA(k+r,n,E)+WS(n)+S,
// Omega(1), with tight Theta(k+r+n)+RA(k+r,n,E)+WS(n)+S on a successful
// record; preflight rejection can take tight Theta(1). RA is aggregate archive
// Reader cost for the frame byte, fixed metadata fields, k/r metadata bytes,
// n payload bytes, and fixed 32-byte digest. WS is aggregate staged Writer cost
// for n payload bytes. S is aggregate Sink.Begin and RecordSink.Commit cost on
// success; failure paths may instead include RecordSink.Abort. Footer completion
// has O(1)+RF(E) time as defined by readFooter. Local auxiliary space is
// O(k+r+B), Omega(1), tight Theta(k+r+B) on a successful record, where B is the
// fixed 32KiB working buffer for a non-empty record and zero for an empty one;
// only the first non-empty record allocates and retains B. Delegated auxiliary
// space is ARA(k+r,n,E)+AWS(n)+AS. Variables k/r are metadata byte lengths,
// n is payload size, and E=MaxConsecutiveEmptyReads.
func (i *Importer) Next(ctx context.Context, sink Sink) (Checkpoint, error) {
	if i == nil || sink == nil {
		return Checkpoint{}, wrap(ErrInvalid, "importer or sink", nil)
	}
	if i.complete {
		return i.cp, ErrComplete
	}
	if i.done {
		return Checkpoint{}, wrap(ErrFinalized, "read record", nil)
	}
	if ctx == nil {
		return Checkpoint{}, wrap(ErrInvalid, "context", nil)
	}
	if err := ctx.Err(); err != nil {
		return Checkpoint{}, wrap(ErrIO, "read frame", err)
	}
	var frame [1]byte
	if err := i.readExact(ctx, frame[:], "read frame"); err != nil {
		i.done = true
		return Checkpoint{}, err
	}
	switch frame[0] {
	case recordFrame:
		checkpoint, err := i.readRecord(ctx, sink)
		if err != nil {
			i.done = true
		}
		return checkpoint, err
	case footerFrame:
		if err := i.readFooter(ctx); err != nil {
			i.done = true
			return Checkpoint{}, err
		}
		return i.cp, ErrComplete
	default:
		i.done = true
		return Checkpoint{}, wrap(ErrMalformed, "frame type", nil)
	}
}

// Manifest returns the final oracle only after Next returns ErrComplete.
// Complexity: time and auxiliary space O(1), Omega(1), tight Theta(1).
func (i *Importer) Manifest() (Manifest, error) {
	if i == nil || !i.complete {
		return Manifest{}, wrap(ErrIncomplete, "manifest", nil)
	}
	return i.manifest, nil
}

// readRecord stages, hashes, verifies, and commits one record.
// Complexity: general time O(k+r+n)+RR(k+r,n,E)+WS(n)+S, Omega(1), with
// tight Theta(k+r+n)+RR(k+r,n,E)+WS(n)+S on success. RR is aggregate archive
// Reader cost for fixed metadata fields, k/r metadata bytes, n payload bytes,
// and the fixed 32-byte digest; WS covers staged writes totaling n bytes; S
// covers Sink.Begin and RecordSink.Commit on success, while failure paths may
// include RecordSink.Abort. Local auxiliary space is O(k+r+B), Omega(1), tight
// Theta(k+r+B) on success; delegated auxiliary space is
// ARR(k+r,n,E)+AWS(n)+AS. B, k/r, n, E, allocation, and retention semantics
// match Next.
func (i *Importer) readRecord(ctx context.Context, sink Sink) (Checkpoint, error) {
	meta, prefix, err := i.readRecordMeta(ctx)
	if err != nil {
		return Checkpoint{}, err
	}
	if meta.Sequence != i.cp.nextSequence {
		return Checkpoint{}, wrap(ErrSequence, "record sequence", nil)
	}
	if i.cp.records >= i.limits.MaxRecords || meta.Size > i.limits.MaxRecordBytes {
		return Checkpoint{}, wrap(ErrLimit, "record", nil)
	}
	total, overflow := checkedAdd(i.cp.payloadBytes, meta.Size)
	if overflow || total > i.limits.MaxTotalBytes {
		return Checkpoint{}, wrap(ErrLimit, "total payload", nil)
	}
	if err := ctx.Err(); err != nil {
		return Checkpoint{}, wrap(ErrIO, "begin record", err)
	}
	stage, beginErr := sink.Begin(ctx, i.cp.header, meta)
	beginContextErr := ctx.Err()
	if beginErr != nil {
		var primary error = wrap(ErrSink, "begin record", beginErr)
		if beginContextErr != nil {
			primary = errors.Join(primary, wrap(ErrIO, "begin record", beginContextErr))
		}
		if stage != nil {
			return Checkpoint{}, i.abort(ctx, stage, primary)
		}
		return Checkpoint{}, primary
	}
	if stage == nil {
		primary := wrap(ErrSink, "begin record", nil)
		if beginContextErr != nil {
			return Checkpoint{}, errors.Join(primary, wrap(ErrIO, "begin record", beginContextErr))
		}
		return Checkpoint{}, primary
	}
	if beginContextErr != nil {
		primary := wrap(ErrIO, "begin record", beginContextErr)
		return Checkpoint{}, i.abort(ctx, stage, primary)
	}
	digest, err := i.copyPayload(ctx, stage, meta.Size)
	if err != nil {
		return Checkpoint{}, i.abort(ctx, stage, err)
	}
	var expected [sha256.Size]byte
	if err := i.readExact(ctx, expected[:], "read record digest"); err != nil {
		return Checkpoint{}, i.abort(ctx, stage, err)
	}
	if !bytes.Equal(digest[:], expected[:]) {
		return Checkpoint{}, i.abort(ctx, stage, wrap(ErrIntegrity, "record digest", nil))
	}
	if err := ctx.Err(); err != nil {
		return Checkpoint{}, i.abort(ctx, stage, wrap(ErrIO, "commit record", err))
	}
	commitErr := stage.Commit(ctx)
	commitContextErr := ctx.Err()
	if commitErr != nil {
		primary := wrap(ErrSink, "commit record; outcome may be unknown", commitErr)
		if commitContextErr != nil {
			canceled := wrap(ErrIO, "commit record; outcome may be unknown", commitContextErr)
			return Checkpoint{}, errors.Join(primary, canceled)
		}
		return Checkpoint{}, primary
	}
	if commitContextErr != nil {
		return Checkpoint{}, wrap(ErrIO, "commit record; outcome may be unknown", commitContextErr)
	}
	i.cp.chain = nextChain(i.cp.chain, prefix, digest)
	i.cp.records++
	i.cp.nextSequence++
	i.cp.payloadBytes = total
	i.cp.offset = i.readOffset
	return i.cp, nil
}

// copyPayload transfers exactly size bytes to staged storage while hashing.
// Complexity: general time O(n)+RP(n,E)+WS(n), Omega(1), with tight
// Theta(n)+RP(n,E)+WS(n) on success. RP is aggregate archive Reader cost for
// exactly n payload bytes, including legal short/empty reads; WS is aggregate
// staged Writer cost for chunks totaling n bytes and is zero when n=0. Peak
// local auxiliary space is O(B), Omega(1), tight Theta(B) for a non-empty
// payload and Theta(1) for an empty payload, where B is the fixed 32KiB working
// buffer; only the first non-empty record allocates and retains B. Delegated
// auxiliary space is ARP(n,E)+AWS(n). Variable n is size and
// E=MaxConsecutiveEmptyReads.
func (i *Importer) copyPayload(ctx context.Context, dst io.Writer, size uint64) ([32]byte, error) {
	h := sha256.New()
	var buf []byte
	if size > 0 {
		if i.buffer == nil {
			i.buffer = make([]byte, copyBufferBytes)
		}
		buf = i.buffer
	}
	remaining := size
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return [32]byte{}, wrap(ErrIO, "read payload", err)
		}
		want := uint64(len(buf))
		if remaining < want {
			want = remaining
		}
		if err := i.readExact(ctx, buf[:int(want)], "read payload"); err != nil {
			return [32]byte{}, err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return [32]byte{}, wrap(ErrIO, "write staged record", ctxErr)
		}
		_, writeErr := writeExact(dst, buf[:int(want)])
		writeContextErr := ctx.Err()
		if writeErr != nil {
			primary := wrap(ErrSink, "write staged record", writeErr)
			if writeContextErr != nil {
				return [32]byte{}, errors.Join(primary, wrap(ErrIO, "write staged record", writeContextErr))
			}
			return [32]byte{}, primary
		}
		if writeContextErr != nil {
			return [32]byte{}, wrap(ErrIO, "write staged record", writeContextErr)
		}
		writeHash(h, buf[:int(want)])
		remaining -= want
	}
	var digest [32]byte
	copy(digest[:], h.Sum(nil))
	return digest, nil
}

// abort preserves the primary classification and adds a redacted sink failure
// if cleanup itself fails.
// Complexity: delegated time/space equal Abort; local time and auxiliary space
// O(1), Omega(1), tight Theta(1), excluding error allocation.
func (i *Importer) abort(ctx context.Context, stage RecordSink, primary error) error {
	return abortWithTimeout(ctx, stage, primary, AbortTimeout)
}

// abortWithTimeout is the bounded cleanup mechanism split out so deadline
// behavior can be tested without making every test wait for AbortTimeout.
// Complexity: general time O(1)+A, Omega(1); an already-expired cleanup context
// takes tight Theta(1) and skips A, while the callback path has tight Theta(1)+A.
// Local auxiliary space is O(1), Omega(1), tight Theta(1), plus delegated Abort
// auxiliary space AA only when called. A/AA are the time/space cost of zero or
// one Abort call. The deadline bounds conforming Abort implementations but
// cannot forcibly interrupt a callback that ignores ctx.
func abortWithTimeout(ctx context.Context, stage RecordSink, primary error, timeout time.Duration) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer cancel()
	if cleanupErr := cleanupCtx.Err(); cleanupErr != nil {
		return errors.Join(primary, wrap(ErrSink, "abort record deadline", cleanupErr))
	}
	abortErr := stage.Abort(cleanupCtx)
	if cleanupErr := cleanupCtx.Err(); cleanupErr != nil {
		if abortErr != nil {
			return errors.Join(primary, wrap(ErrSink, "abort record deadline", errors.Join(cleanupErr, abortErr)))
		}
		return errors.Join(primary, wrap(ErrSink, "abort record deadline", cleanupErr))
	}
	if abortErr != nil {
		return errors.Join(primary, wrap(ErrSink, "abort record", abortErr))
	}
	return primary
}

// readRecordMeta decodes bounded metadata and reconstructs its canonical bytes.
// Complexity: general time O(k+r)+RM(k+r,E), Omega(1), with tight
// Theta(k+r)+RM(k+r,E) on success; local auxiliary space O(k+r), Omega(1),
// tight Theta(k+r) on success, plus delegated Reader space ARM(k+r,E). RM is
// aggregate archive Reader cost for fixed sequence/length/size fields and k/r
// metadata bytes. Variables k/r are metadata byte lengths and
// E=MaxConsecutiveEmptyReads.
func (i *Importer) readRecordMeta(ctx context.Context) (RecordMeta, []byte, error) {
	sequence, err := i.readUint64(ctx, "read record sequence")
	if err != nil {
		return RecordMeta{}, nil, err
	}
	kind, err := i.readText(ctx, maxKindBytes, false, "record kind")
	if err != nil {
		return RecordMeta{}, nil, err
	}
	key, err := i.readText(ctx, maxKeyBytes, true, "record key")
	if err != nil {
		return RecordMeta{}, nil, err
	}
	size, err := i.readUint64(ctx, "read payload size")
	if err != nil {
		return RecordMeta{}, nil, err
	}
	meta := RecordMeta{Sequence: sequence, Kind: kind, Key: key, Size: size}
	return meta, encodeRecordPrefix(meta), nil
}

// readFooter verifies counts, chain, and immediate EOF before marking complete.
// Complexity: general time O(1)+RF(E), Omega(1), with tight Theta(1)+RF(E) on
// successful completion; local auxiliary space O(1), Omega(1), tight Theta(1),
// plus delegated Reader space ARF(E). RF is aggregate archive Reader cost for
// two fixed uint64 values, the fixed 32-byte chain, and the required EOF probe,
// including at most E consecutive empty reads; E=MaxConsecutiveEmptyReads.
func (i *Importer) readFooter(ctx context.Context) error {
	records, err := i.readUint64(ctx, "read manifest records")
	if err != nil {
		return err
	}
	payloadBytes, err := i.readUint64(ctx, "read manifest payload bytes")
	if err != nil {
		return err
	}
	var chain [sha256.Size]byte
	if err := i.readExact(ctx, chain[:], "read manifest chain"); err != nil {
		return err
	}
	if records != i.cp.records || payloadBytes != i.cp.payloadBytes {
		return wrap(ErrIntegrity, "manifest counters", nil)
	}
	if !bytes.Equal(chain[:], i.cp.chain[:]) {
		return wrap(ErrIntegrity, "manifest chain", nil)
	}
	n, readErr := probeEOFContext(ctx, i.r)
	nextOffset, overflow := checkedAdd(i.readOffset, uint64(n))
	if overflow {
		return wrap(ErrLimit, "stream offset", nil)
	}
	i.readOffset = nextOffset
	if n != 0 {
		return wrap(ErrMalformed, "trailing archive data", nil)
	}
	if readErr != io.EOF {
		return wrap(ErrIO, "read trailing archive data", readErr)
	}
	i.manifest = Manifest{Records: records, PayloadBytes: payloadBytes, Chain: chain}
	i.complete = true
	i.done = true
	return nil
}

// readExact fills p, tracks physical progress, and classifies premature EOF.
// Complexity: time O(n)+R(n), Omega(1), tight Theta(n)+R(n) on success;
// auxiliary space O(1), Omega(1), tight Theta(1); variable n is len(p), R is
// delegated reader cost.
func (i *Importer) readExact(ctx context.Context, p []byte, op string) error {
	n, err := readExactContext(ctx, i.r, p)
	nextOffset, overflow := checkedAdd(i.readOffset, uint64(n))
	if overflow {
		return wrap(ErrLimit, "stream offset", nil)
	}
	i.readOffset = nextOffset
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return wrap(ErrTruncated, op, err)
		}
		return wrap(ErrIO, op, err)
	}
	return nil
}

// readUint64 reads one canonical unsigned integer.
// Complexity: time O(1)+R(8,E), Omega(1), with tight Theta(1)+R(8,E) on
// success; auxiliary space O(1), Omega(1), tight Theta(1). R(8,E) is the
// aggregate delegated Reader cost of the bounded exact-read loop for eight
// bytes, including legal short and at most E consecutive empty results; it is
// not necessarily one Reader call.
func (i *Importer) readUint64(ctx context.Context, op string) (uint64, error) {
	var b [8]byte
	if err := i.readExact(ctx, b[:], op); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(b[:]), nil
}

// readText reads one u16-length bounded UTF-8 string.
// Complexity: general time O(n)+RT(n,E), Omega(1), with tight
// Theta(n)+RT(n,E) on success; local auxiliary space O(n), Omega(1), tight
// Theta(n) on non-empty success and Theta(1) for an empty allowed value, plus
// delegated Reader space ART(n,E). RT is aggregate archive Reader cost for the
// fixed two-byte length and n text bytes; E=MaxConsecutiveEmptyReads.
func (i *Importer) readText(ctx context.Context, max int, allowEmpty bool, field string) (string, error) {
	var length [2]byte
	if err := i.readExact(ctx, length[:], "read "+field+" length"); err != nil {
		return "", err
	}
	n := int(binary.BigEndian.Uint16(length[:]))
	if n > max || (!allowEmpty && n == 0) {
		return "", wrap(ErrMalformed, field, nil)
	}
	b := make([]byte, n)
	if err := i.readExact(ctx, b, "read "+field); err != nil {
		return "", err
	}
	if !utf8.Valid(b) || bytes.IndexByte(b, 0) >= 0 {
		return "", wrap(ErrMalformed, field, nil)
	}
	return string(b), nil
}

// readHeader decodes the bounded fixed-version header and returns its canonical
// bytes for the initial chain.
// Complexity: time O(a+s)+R(a+s), Omega(1), tight Theta(a+s)+R(a+s) on
// success; auxiliary space O(a+s), Omega(1), tight Theta(a+s); variables a/s
// are identifier lengths and R is delegated reader cost.
func readHeader(ctx context.Context, r io.Reader) (Header, []byte, error) {
	reader := &Importer{r: r}
	var magic [8]byte
	if err := reader.readExact(ctx, magic[:], "read header magic"); err != nil {
		return Header{}, nil, err
	}
	if magic != streamMagic {
		return Header{}, nil, wrap(ErrMalformed, "header magic", nil)
	}
	var versionBytes [2]byte
	if err := reader.readExact(ctx, versionBytes[:], "read wire version"); err != nil {
		return Header{}, nil, err
	}
	version := binary.BigEndian.Uint16(versionBytes[:])
	if version != WireVersion {
		return Header{}, nil, wrap(ErrIncompatible, "wire version", nil)
	}
	archiveID, err := reader.readText(ctx, maxArchiveIDBytes, false, "archive id")
	if err != nil {
		return Header{}, nil, err
	}
	schema, err := reader.readText(ctx, maxSchemaBytes, false, "schema")
	if err != nil {
		return Header{}, nil, err
	}
	var schemaVersion [4]byte
	if err := reader.readExact(ctx, schemaVersion[:], "read schema version"); err != nil {
		return Header{}, nil, err
	}
	header := Header{WireVersion: version, ArchiveID: archiveID, Schema: schema, SchemaVersion: binary.BigEndian.Uint32(schemaVersion[:])}
	return header, encodeHeader(header), nil
}
