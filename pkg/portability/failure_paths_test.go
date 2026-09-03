package portability

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"testing"
)

type failingWriter struct {
	remaining int
	short     bool
}

func (w *failingWriter) Write(p []byte) (int, error) {
	if w.short {
		return 0, nil
	}
	if w.remaining <= 0 {
		return 0, errors.New("writer-secret")
	}
	n := len(p)
	if n > w.remaining {
		n = w.remaining
	}
	w.remaining -= n
	if n < len(p) {
		return n, errors.New("writer-secret")
	}
	return n, nil
}

type noProgressReader struct{}

func (noProgressReader) Read([]byte) (int, error) { return 0, nil }

type oversizedCountReader struct{}

func (oversizedCountReader) Read(p []byte) (int, error) { return len(p) + 1, nil }

func TestExporterArgumentAndIOFailures(t *testing.T) {
	t.Parallel()

	if _, err := NewExporter(nil, testHeader(), Limits{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil writer = %v", err)
	}
	if _, err := NewExporter(io.Discard, Header{}, Limits{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid header = %v", err)
	}
	if _, err := NewExporter(io.Discard, testHeader(), Limits{MaxRecordBytes: 2, MaxTotalBytes: 1}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid limits = %v", err)
	}
	for _, writer := range []io.Writer{&failingWriter{}, &failingWriter{short: true}} {
		if _, err := NewExporter(writer, testHeader(), Limits{}); !errors.Is(err, ErrMalformed) || strings.Contains(err.Error(), "secret") {
			t.Fatalf("header writer = %v", err)
		}
	}

	var out bytes.Buffer
	ex, err := NewExporter(&out, testHeader(), Limits{MaxRecords: 1, MaxRecordBytes: 2, MaxTotalBytes: 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 0}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil body = %v", err)
	}
	if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 2, Body: strings.NewReader("ab")}); err != nil {
		t.Fatal(err)
	}
	if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Body: strings.NewReader("")}); !errors.Is(err, ErrLimit) {
		t.Fatalf("record count = %v", err)
	}

	for _, reader := range []io.Reader{noProgressReader{}, oversizedCountReader{}} {
		var target bytes.Buffer
		ex, err := NewExporter(&target, testHeader(), Limits{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 1, Body: reader}); err == nil {
			t.Fatal("bad reader unexpectedly succeeded")
		}
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	var canceledOut bytes.Buffer
	canceledExporter, err := NewExporter(&canceledOut, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := canceledExporter.WriteRecord(canceled, Record{Kind: "x", Size: 1, Body: strings.NewReader("a")}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled write = %v", err)
	}
}

func TestExporterRecordAndFooterWriterFailures(t *testing.T) {
	t.Parallel()

	headerLen := len(encodeHeader(testHeader()))
	for _, remaining := range []int{headerLen + 1, headerLen + 30, headerLen + 50} {
		writer := &failingWriter{remaining: remaining}
		ex, err := NewExporter(writer, testHeader(), Limits{})
		if err != nil {
			t.Fatal(err)
		}
		_, err = ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 40, Body: strings.NewReader(strings.Repeat("a", 40))})
		if !errors.Is(err, ErrMalformed) || strings.Contains(err.Error(), "secret") {
			t.Fatalf("remaining %d: %v", remaining, err)
		}
	}

	writer := &failingWriter{remaining: headerLen}
	ex, err := NewExporter(writer, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ex.Finalize(); !errors.Is(err, ErrMalformed) {
		t.Fatalf("footer writer = %v", err)
	}
}

func TestResumeArgumentFailuresAndCheckpointAccessors(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ex, err := NewExporter(&out, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	cp := ex.cp
	if cp.Header() != testHeader() || cp.NextSequence() != 0 || cp.PayloadBytes() != 0 {
		t.Fatalf("accessors: %#v", cp)
	}
	if _, err := ResumeExporter(nil, cp, Limits{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil resume writer = %v", err)
	}
	if _, err := ResumeExporter(io.Discard, cp, Limits{MaxRecordBytes: 2, MaxTotalBytes: 1}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad resume limits = %v", err)
	}
	bad := cp
	bad.offset = 0
	if _, err := ResumeExporter(io.Discard, bad, Limits{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad checkpoint = %v", err)
	}
	var nilExporter *Exporter
	if _, err := nilExporter.WriteRecord(context.Background(), Record{}); !errors.Is(err, ErrFinalized) {
		t.Fatalf("nil exporter write = %v", err)
	}
	if _, err := nilExporter.Finalize(); !errors.Is(err, ErrFinalized) {
		t.Fatalf("nil exporter finalize = %v", err)
	}
}

func TestEveryArchiveTruncationFailsClosed(t *testing.T) {
	t.Parallel()

	data := archiveBytes(t, Record{Kind: "kind", Key: "key", Size: 5, Body: strings.NewReader("value")})
	for cut := 0; cut < len(data); cut++ {
		im, err := NewImporter(bytes.NewReader(data[:cut]), Limits{}, acceptExact)
		if err != nil {
			if !errors.Is(err, ErrTruncated) {
				t.Fatalf("cut %d constructor = %v", cut, err)
			}
			continue
		}
		sink := &memorySink{}
		for {
			_, err = im.Next(context.Background(), sink)
			if err != nil {
				break
			}
		}
		if !errors.Is(err, ErrTruncated) {
			t.Fatalf("cut %d next = %v, want ErrTruncated", cut, err)
		}
		if _, manifestErr := im.Manifest(); !errors.Is(manifestErr, ErrIncomplete) {
			t.Fatalf("cut %d manifest = %v", cut, manifestErr)
		}
	}
}

func TestImporterMalformedSequenceLimitsAndSinkFailures(t *testing.T) {
	t.Parallel()

	data := archiveBytes(t,
		Record{Kind: "x", Size: 1, Body: strings.NewReader("a")},
		Record{Kind: "x", Size: 1, Body: strings.NewReader("b")},
	)
	headerLen := len(encodeHeader(testHeader()))

	unknownFrame := append([]byte(nil), data...)
	unknownFrame[headerLen] = 0x44
	im, err := NewImporter(bytes.NewReader(unknownFrame), Limits{}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := im.Next(context.Background(), &memorySink{}); !errors.Is(err, ErrMalformed) {
		t.Fatalf("unknown frame = %v", err)
	}

	badSequence := append([]byte(nil), data...)
	binary.BigEndian.PutUint64(badSequence[headerLen+1:headerLen+9], 1)
	im, err = NewImporter(bytes.NewReader(badSequence), Limits{}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := im.Next(context.Background(), &memorySink{}); !errors.Is(err, ErrSequence) {
		t.Fatalf("sequence = %v", err)
	}

	im, err = NewImporter(bytes.NewReader(data), Limits{MaxRecords: 1, MaxRecordBytes: 1, MaxTotalBytes: 2}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := im.Next(context.Background(), &memorySink{}); err != nil {
		t.Fatal(err)
	}
	if _, err := im.Next(context.Background(), &memorySink{}); !errors.Is(err, ErrLimit) {
		t.Fatalf("record count = %v", err)
	}

	for _, sink := range []*memorySink{{failBegin: true}, {failCommit: true}, {failWrite: true, failAbort: true}} {
		im, err := NewImporter(bytes.NewReader(data), Limits{}, acceptExact)
		if err != nil {
			t.Fatal(err)
		}
		_, firstErr := im.Next(context.Background(), sink)
		if !errors.Is(firstErr, ErrSink) || strings.Contains(firstErr.Error(), "secret") {
			t.Fatalf("sink failure = %v", firstErr)
		}
		if _, err := im.Next(context.Background(), sink); !errors.Is(err, ErrFinalized) {
			t.Fatalf("poisoned importer = %v", err)
		}
	}
}

func TestImporterArgumentAndHeaderFailures(t *testing.T) {
	t.Parallel()

	if _, err := NewImporter(nil, Limits{}, acceptExact); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil reader = %v", err)
	}
	if _, err := NewImporter(strings.NewReader(""), Limits{}, nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil compatibility = %v", err)
	}
	data := archiveBytes(t)
	wrongMagic := append([]byte(nil), data...)
	wrongMagic[0] ^= 1
	if _, err := NewImporter(bytes.NewReader(wrongMagic), Limits{}, acceptExact); !errors.Is(err, ErrMalformed) {
		t.Fatalf("magic = %v", err)
	}
	wrongVersion := append([]byte(nil), data...)
	binary.BigEndian.PutUint16(wrongVersion[8:10], WireVersion+1)
	if _, err := NewImporter(bytes.NewReader(wrongVersion), Limits{}, acceptExact); !errors.Is(err, ErrIncompatible) {
		t.Fatalf("wire version = %v", err)
	}
	im, err := NewImporter(bytes.NewReader(data), Limits{}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	if im.Header() != testHeader() {
		t.Fatalf("header = %#v", im.Header())
	}
	var nilImporter *Importer
	if nilImporter.Header() != (Header{}) {
		t.Fatal("nil importer header")
	}
	if _, err := nilImporter.Next(context.Background(), &memorySink{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil importer next = %v", err)
	}
	if _, err := im.Next(context.Background(), nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil sink = %v", err)
	}
	if _, err := nilImporter.Manifest(); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("nil manifest = %v", err)
	}
}

func TestSinkFuncAndCanceledImport(t *testing.T) {
	t.Parallel()

	called := false
	f := SinkFunc(func(context.Context, Header, RecordMeta) (RecordSink, error) {
		called = true
		return nil, errors.New("stop")
	})
	_, _ = f.Begin(context.Background(), Header{}, RecordMeta{})
	if !called {
		t.Fatal("sink function not called")
	}
	im, err := NewImporter(bytes.NewReader(archiveBytes(t)), Limits{}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := im.Next(ctx, &memorySink{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled next = %v", err)
	}
}

func resignCheckpoint(data []byte) {
	digest := sha256.Sum256(data[:len(data)-sha256.Size])
	copy(data[len(data)-sha256.Size:], digest[:])
}

func TestCheckpointParserClassifications(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ex, err := NewExporter(&out, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := ex.cp.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	wrongMagic := append([]byte(nil), encoded...)
	wrongMagic[0] ^= 1
	if _, err := ParseCheckpoint(wrongMagic, Limits{}); !errors.Is(err, ErrMalformed) {
		t.Fatalf("magic = %v", err)
	}
	wrongVersion := append([]byte(nil), encoded...)
	binary.BigEndian.PutUint16(wrongVersion[8:10], checkpointVersion+1)
	resignCheckpoint(wrongVersion)
	if _, err := ParseCheckpoint(wrongVersion, Limits{}); !errors.Is(err, ErrIncompatible) {
		t.Fatalf("version = %v", err)
	}
	badHeaderLength := append([]byte(nil), encoded...)
	binary.BigEndian.PutUint16(badHeaderLength[10:12], 0xffff)
	if _, err := ParseCheckpoint(badHeaderLength, Limits{}); !errors.Is(err, ErrMalformed) {
		t.Fatalf("header length = %v", err)
	}
	if _, err := ParseCheckpoint(encoded, Limits{MaxRecordBytes: 2, MaxTotalBytes: 1}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("limits = %v", err)
	}
}
