package portability

import (
	"context"
	"io"
	"math"
	"time"
	"unicode/utf8"
)

const (
	WireVersion           uint16 = 1
	DefaultMaxRecords     uint64 = 1_000_000
	DefaultMaxRecordBytes uint64 = 1 << 30
	DefaultMaxTotalBytes  uint64 = 1 << 40
	maxArchiveIDBytes            = 256
	maxSchemaBytes               = 256
	maxKindBytes                 = 128
	maxKeyBytes                  = 1024
	copyBufferBytes              = 32 << 10
	// MaxConsecutiveEmptyReads bounds legal Reader (0, nil) responses before
	// the operation fails with io.ErrNoProgress.
	MaxConsecutiveEmptyReads = 100
	// AbortTimeout bounds best-effort staged-record cleanup after an import
	// failure. RecordSink.Abort implementations must honor the context deadline.
	AbortTimeout = 5 * time.Second
)

// Header identifies one archive and its consumer-owned schema contract.
type Header struct {
	WireVersion   uint16
	ArchiveID     string
	Schema        string
	SchemaVersion uint32
}

// Limits bounds untrusted archive work. Zero fields select documented defaults.
type Limits struct {
	MaxRecords     uint64
	MaxRecordBytes uint64
	MaxTotalBytes  uint64
}

// RecordMeta describes an opaque record without interpreting its payload.
type RecordMeta struct {
	Sequence uint64
	Kind     string
	Key      string
	Size     uint64
}

// Record is one export record. Body must end exactly after Size bytes.
type Record struct {
	Kind string
	Key  string
	Size uint64
	Body io.Reader
}

// Manifest is the validated final completeness oracle.
type Manifest struct {
	Records      uint64
	PayloadBytes uint64
	Chain        [32]byte
}

// Compatibility validates the consumer-owned schema contract. Returning an
// error rejects the archive before any record sink is opened.
type Compatibility func(context.Context, Header) error

// Causer exposes the raw callback or I/O cause without adding that cause to
// errors.Is traversal. The returned error may contain payload fragments,
// storage identifiers, secrets, or other sensitive consumer data. Callers must
// apply their own redaction before logging or displaying it. This prevents raw
// callback values from spoofing a library sentinel while preserving explicit,
// opt-in diagnostics for trusted code.
type Causer interface {
	Cause() error
}

// RecordSink stages one imported record. Commit must be idempotent for archive
// identity and sequence; Abort must discard uncommitted bytes.
type RecordSink interface {
	io.Writer
	Commit(context.Context) error
	Abort(context.Context) error
}

// Sink begins consumer-owned staging for one record. If Begin returns both a
// non-nil RecordSink and a non-nil error, the library calls Abort on that stage
// before returning the begin failure. Cancellation observed after Begin is
// additionally classified as ErrIO rather than suppressing either outcome.
type Sink interface {
	Begin(context.Context, Header, RecordMeta) (RecordSink, error)
}

// SinkFunc adapts a function to Sink.
type SinkFunc func(context.Context, Header, RecordMeta) (RecordSink, error)

// Begin calls f.
// Complexity: delegated time and space equal f; local time O(1), Omega(1),
// tight Theta(1), and local auxiliary space O(1), Omega(1), tight Theta(1).
func (f SinkFunc) Begin(ctx context.Context, h Header, m RecordMeta) (RecordSink, error) {
	return f(ctx, h, m)
}

// normalize supplies defaults and rejects internally inconsistent bounds.
// Complexity: time O(1), Omega(1), tight Theta(1); auxiliary space O(1),
// Omega(1), tight Theta(1).
func (l Limits) normalize() (Limits, error) {
	if l.MaxRecords == 0 {
		l.MaxRecords = DefaultMaxRecords
	}
	if l.MaxRecordBytes == 0 {
		l.MaxRecordBytes = DefaultMaxRecordBytes
	}
	if l.MaxTotalBytes == 0 {
		l.MaxTotalBytes = DefaultMaxTotalBytes
	}
	if l.MaxRecordBytes > math.MaxInt64 || l.MaxRecordBytes > l.MaxTotalBytes {
		return Limits{}, wrap(ErrInvalid, "limits", nil)
	}
	return l, nil
}

// validate checks the fixed wire-header contract.
// Complexity: time O(a+s), Omega(1), tight Theta(a+s) when both identifiers
// are inspected; auxiliary space O(1), Omega(1), tight Theta(1); variables: a
// and s are archive-ID and schema byte lengths.
func (h Header) validate() error {
	if h.WireVersion != WireVersion {
		return wrap(ErrInvalid, "wire version", nil)
	}
	if err := validateText("archive id", h.ArchiveID, maxArchiveIDBytes, false); err != nil {
		return err
	}
	return validateText("schema", h.Schema, maxSchemaBytes, false)
}

// validate checks bounded record metadata.
// Complexity: time O(k+r), Omega(1), tight Theta(k+r) when both fields are
// inspected; auxiliary space O(1), Omega(1), tight Theta(1); variables: k and
// r are kind and key byte lengths.
func (m RecordMeta) validate() error {
	if err := validateText("record kind", m.Kind, maxKindBytes, false); err != nil {
		return err
	}
	return validateText("record key", m.Key, maxKeyBytes, true)
}

// validateText checks length, UTF-8, and NUL without copying text.
// Complexity: time O(n), Omega(1), tight Theta(n) for valid input; auxiliary
// space O(1), Omega(1), tight Theta(1); variable n is len(value).
func validateText(field, value string, max int, allowEmpty bool) error {
	if (!allowEmpty && value == "") || len(value) > max || !utf8.ValidString(value) {
		return wrap(ErrInvalid, field, nil)
	}
	for i := 0; i < len(value); i++ {
		if value[i] == 0 {
			return wrap(ErrInvalid, field, nil)
		}
	}
	return nil
}
