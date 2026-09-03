package portability

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"unicode/utf8"
)

// Importer validates and stages one archive. It is not safe for concurrent use.
type Importer struct {
	r          io.Reader
	limits     Limits
	cp         Checkpoint
	readOffset uint64
	done       bool
	manifest   Manifest
	complete   bool
}

// NewImporter reads and validates the header and calls compatibility before
// any record can be staged.
// Complexity: time O(a+s)+R(a+s)+C, Omega(1), tight Theta(a+s)+R(a+s)+C on
// success; auxiliary space O(a+s), Omega(1), tight Theta(a+s); variables: a/s
// identifier lengths, R reader cost, C delegated compatibility cost.
func NewImporter(r io.Reader, limits Limits, compatibility Compatibility) (*Importer, error) {
	if r == nil || compatibility == nil {
		return nil, wrap(ErrInvalid, "reader or compatibility", nil)
	}
	normalized, err := limits.normalize()
	if err != nil {
		return nil, err
	}
	header, encoded, err := readHeader(r)
	if err != nil {
		return nil, err
	}
	if err := compatibility(header); err != nil {
		return nil, wrap(ErrIncompatible, "consumer schema", err)
	}
	chain := sha256.Sum256(encoded)
	cp := Checkpoint{header: header, offset: uint64(len(encoded)), chain: chain}
	return &Importer{r: r, limits: normalized, cp: cp, readOffset: cp.offset}, nil
}

// ResumeImporter restores a validated committed boundary. The caller must
// position the reader at checkpoint.Offset.
// Complexity: time O(a+s)+C, Omega(1), tight Theta(a+s)+C for valid state;
// auxiliary space O(1), Omega(1), tight Theta(1); variables: a/s identifier
// lengths and C delegated compatibility cost.
func ResumeImporter(r io.Reader, checkpoint Checkpoint, limits Limits, compatibility Compatibility) (*Importer, error) {
	if r == nil || compatibility == nil {
		return nil, wrap(ErrInvalid, "reader or compatibility", nil)
	}
	normalized, err := limits.normalize()
	if err != nil {
		return nil, err
	}
	if err := checkpoint.validate(normalized); err != nil {
		return nil, err
	}
	if err := compatibility(checkpoint.header); err != nil {
		return nil, wrap(ErrIncompatible, "consumer schema", err)
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
// the manifest and required EOF. A non-completion failure poisons this importer.
// Complexity: record time O(k+r+n)+R(n)+W(n)+S, Omega(k+r), tight
// Theta(k+r+n)+R(n)+W(n)+S on success; auxiliary space O(k+r+B), Omega(k+r),
// tight Theta(k+r+B) for n>0; variables: metadata k/r, payload n,
// B=min(n,32KiB), reader R, sink writer W, and sink callbacks S.
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
	if err := ctx.Err(); err != nil {
		return Checkpoint{}, wrap(ErrIO, "read frame", err)
	}
	var frame [1]byte
	if err := i.readExact(frame[:], "read frame"); err != nil {
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
		if err := i.readFooter(); err != nil {
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
// Complexity: time O(k+r+n)+R(n)+W(n)+S, Omega(k+r), tight
// Theta(k+r+n)+R(n)+W(n)+S on success; auxiliary space O(k+r+B), Omega(k+r),
// tight Theta(k+r+B) for n>0; variables and delegated costs match Next.
func (i *Importer) readRecord(ctx context.Context, sink Sink) (Checkpoint, error) {
	meta, prefix, err := i.readRecordMeta()
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
	stage, err := sink.Begin(ctx, i.cp.header, meta)
	if err != nil || stage == nil {
		return Checkpoint{}, wrap(ErrSink, "begin record", err)
	}
	digest, err := i.copyPayload(ctx, stage, meta.Size)
	if err != nil {
		return Checkpoint{}, i.abort(ctx, stage, err)
	}
	var expected [sha256.Size]byte
	if err := i.readExact(expected[:], "read record digest"); err != nil {
		return Checkpoint{}, i.abort(ctx, stage, err)
	}
	if !bytes.Equal(digest[:], expected[:]) {
		return Checkpoint{}, i.abort(ctx, stage, wrap(ErrIntegrity, "record digest", nil))
	}
	if err := stage.Commit(ctx); err != nil {
		return Checkpoint{}, wrap(ErrSink, "commit record; outcome may be unknown", err)
	}
	i.cp.chain = nextChain(i.cp.chain, prefix, digest)
	i.cp.records++
	i.cp.nextSequence++
	i.cp.payloadBytes = total
	i.cp.offset = i.readOffset
	return i.cp, nil
}

// copyPayload transfers exactly size bytes to staged storage while hashing.
// Complexity: time O(n)+R(n)+W(n), Omega(1), tight Theta(n)+R(n)+W(n) on
// success; auxiliary space O(min(n,32KiB)), Omega(1), tight Theta(min(n,32KiB));
// variable n is size and R/W are delegated I/O costs.
func (i *Importer) copyPayload(ctx context.Context, dst io.Writer, size uint64) ([32]byte, error) {
	h := sha256.New()
	buf := make([]byte, copyBufferBytes)
	remaining := size
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return [32]byte{}, wrap(ErrIO, "read payload", err)
		}
		want := uint64(len(buf))
		if remaining < want {
			want = remaining
		}
		if err := i.readExact(buf[:int(want)], "read payload"); err != nil {
			return [32]byte{}, err
		}
		if _, err := writeExact(dst, buf[:int(want)]); err != nil {
			return [32]byte{}, wrap(ErrSink, "write staged record", err)
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
	if err := stage.Abort(context.WithoutCancel(ctx)); err != nil {
		return errors.Join(primary, wrap(ErrSink, "abort record", err))
	}
	return primary
}

// readRecordMeta decodes bounded metadata and reconstructs its canonical bytes.
// Complexity: time O(k+r)+R(k+r), Omega(1), tight Theta(k+r)+R(k+r) on
// success; auxiliary space O(k+r), Omega(1), tight Theta(k+r); variables k/r
// are metadata lengths and R is delegated reader cost.
func (i *Importer) readRecordMeta() (RecordMeta, []byte, error) {
	sequence, err := i.readUint64("read record sequence")
	if err != nil {
		return RecordMeta{}, nil, err
	}
	kind, err := i.readText(maxKindBytes, false, "record kind")
	if err != nil {
		return RecordMeta{}, nil, err
	}
	key, err := i.readText(maxKeyBytes, true, "record key")
	if err != nil {
		return RecordMeta{}, nil, err
	}
	size, err := i.readUint64("read payload size")
	if err != nil {
		return RecordMeta{}, nil, err
	}
	meta := RecordMeta{Sequence: sequence, Kind: kind, Key: key, Size: size}
	return meta, encodeRecordPrefix(meta), nil
}

// readFooter verifies counts, chain, and immediate EOF before marking complete.
// Complexity: time O(1)+R(1), Omega(1), tight Theta(1)+R(1); auxiliary space
// O(1), Omega(1), tight Theta(1); R is fixed-size delegated reader cost.
func (i *Importer) readFooter() error {
	records, err := i.readUint64("read manifest records")
	if err != nil {
		return err
	}
	payloadBytes, err := i.readUint64("read manifest payload bytes")
	if err != nil {
		return err
	}
	var chain [sha256.Size]byte
	if err := i.readExact(chain[:], "read manifest chain"); err != nil {
		return err
	}
	if records != i.cp.records || payloadBytes != i.cp.payloadBytes {
		return wrap(ErrIntegrity, "manifest counters", nil)
	}
	if !bytes.Equal(chain[:], i.cp.chain[:]) {
		return wrap(ErrIntegrity, "manifest chain", nil)
	}
	var trailing [1]byte
	n, readErr := i.r.Read(trailing[:])
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
func (i *Importer) readExact(p []byte, op string) error {
	n, err := io.ReadFull(i.r, p)
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
// Complexity: time O(1)+R(1), Omega(1), tight Theta(1)+R(1); auxiliary space
// O(1), Omega(1), tight Theta(1); R is one fixed-size delegated read.
func (i *Importer) readUint64(op string) (uint64, error) {
	var b [8]byte
	if err := i.readExact(b[:], op); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(b[:]), nil
}

// readText reads one u16-length bounded UTF-8 string.
// Complexity: time O(n)+R(n), Omega(1), tight Theta(n)+R(n) on success;
// auxiliary space O(n), Omega(1), tight Theta(n); variable n is encoded text
// length and R is delegated reader cost.
func (i *Importer) readText(max int, allowEmpty bool, field string) (string, error) {
	var length [2]byte
	if err := i.readExact(length[:], "read "+field+" length"); err != nil {
		return "", err
	}
	n := int(binary.BigEndian.Uint16(length[:]))
	if n > max || (!allowEmpty && n == 0) {
		return "", wrap(ErrMalformed, field, nil)
	}
	b := make([]byte, n)
	if err := i.readExact(b, "read "+field); err != nil {
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
func readHeader(r io.Reader) (Header, []byte, error) {
	reader := &Importer{r: r}
	var magic [8]byte
	if err := reader.readExact(magic[:], "read header magic"); err != nil {
		return Header{}, nil, err
	}
	if magic != streamMagic {
		return Header{}, nil, wrap(ErrMalformed, "header magic", nil)
	}
	var versionBytes [2]byte
	if err := reader.readExact(versionBytes[:], "read wire version"); err != nil {
		return Header{}, nil, err
	}
	version := binary.BigEndian.Uint16(versionBytes[:])
	if version != WireVersion {
		return Header{}, nil, wrap(ErrIncompatible, "wire version", nil)
	}
	archiveID, err := reader.readText(maxArchiveIDBytes, false, "archive id")
	if err != nil {
		return Header{}, nil, err
	}
	schema, err := reader.readText(maxSchemaBytes, false, "schema")
	if err != nil {
		return Header{}, nil, err
	}
	var schemaVersion [4]byte
	if err := reader.readExact(schemaVersion[:], "read schema version"); err != nil {
		return Header{}, nil, err
	}
	header := Header{WireVersion: version, ArchiveID: archiveID, Schema: schema, SchemaVersion: binary.BigEndian.Uint32(schemaVersion[:])}
	return header, encodeHeader(header), nil
}
