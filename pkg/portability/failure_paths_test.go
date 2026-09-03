package portability

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"math"
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

type failAfterReader struct {
	r         io.Reader
	remaining int
}

type cancelOnWriteSink struct {
	cancel context.CancelFunc
	aborts int
}

func (s *cancelOnWriteSink) Begin(context.Context, Header, RecordMeta) (RecordSink, error) {
	return cancelOnWriteRecord{s}, nil
}

type cancelOnWriteRecord struct{ owner *cancelOnWriteSink }

func (r cancelOnWriteRecord) Write(p []byte) (int, error) {
	r.owner.cancel()
	return len(p), nil
}
func (cancelOnWriteRecord) Commit(context.Context) error { return nil }
func (r cancelOnWriteRecord) Abort(context.Context) error {
	r.owner.aborts++
	return nil
}

func (r *failAfterReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, errors.New("reader-secret")
	}
	if len(p) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.r.Read(p)
	r.remaining -= n
	return n, err
}

func TestExporterArgumentAndIOFailures(t *testing.T) {
	t.Parallel()

	if _, err := NewExporter(context.Background(), nil, testHeader(), Limits{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil writer = %v", err)
	}
	if _, err := NewExporter(context.Background(), io.Discard, Header{}, Limits{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid header = %v", err)
	}
	if _, err := NewExporter(context.Background(), io.Discard, testHeader(), Limits{MaxRecordBytes: 2, MaxTotalBytes: 1}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid limits = %v", err)
	}
	for _, writer := range []io.Writer{&failingWriter{}, &failingWriter{short: true}} {
		if _, err := NewExporter(context.Background(), writer, testHeader(), Limits{}); !errors.Is(err, ErrIO) || strings.Contains(err.Error(), "secret") {
			t.Fatalf("header writer = %v", err)
		}
	}

	var out bytes.Buffer
	ex, err := NewExporter(context.Background(), &out, testHeader(), Limits{MaxRecords: 1, MaxRecordBytes: 2, MaxTotalBytes: 2})
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
		ex, err := NewExporter(context.Background(), &target, testHeader(), Limits{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 1, Body: reader}); err == nil {
			t.Fatal("bad reader unexpectedly succeeded")
		}
	}
	var boundaryOut bytes.Buffer
	boundaryExporter, err := NewExporter(context.Background(), &boundaryOut, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = boundaryExporter.WriteRecord(context.Background(), Record{Kind: "x", Size: 1, Body: &failAfterReader{r: strings.NewReader("a"), remaining: 1}})
	if !errors.Is(err, ErrIO) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("EOF probe I/O = %v", err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	var canceledOut bytes.Buffer
	canceledExporter, err := NewExporter(context.Background(), &canceledOut, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := canceledExporter.WriteRecord(canceled, Record{Kind: "x", Size: 1, Body: strings.NewReader("a")}); !errors.Is(err, ErrIO) || errors.Is(err, context.Canceled) || !causeIs(err, context.Canceled) {
		t.Fatalf("canceled write = %v", err)
	}
}

func TestExporterRecordAndFooterWriterFailures(t *testing.T) {
	t.Parallel()

	headerLen := len(encodeHeader(testHeader()))
	for _, remaining := range []int{headerLen + 1, headerLen + 30, headerLen + 50} {
		writer := &failingWriter{remaining: remaining}
		ex, err := NewExporter(context.Background(), writer, testHeader(), Limits{})
		if err != nil {
			t.Fatal(err)
		}
		_, err = ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 40, Body: strings.NewReader(strings.Repeat("a", 40))})
		if !errors.Is(err, ErrIO) || strings.Contains(err.Error(), "secret") {
			t.Fatalf("remaining %d: %v", remaining, err)
		}
	}

	writer := &failingWriter{remaining: headerLen}
	ex, err := NewExporter(context.Background(), writer, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ex.Finalize(context.Background()); !errors.Is(err, ErrIO) {
		t.Fatalf("footer writer = %v", err)
	}
}

func TestResumeArgumentFailuresAndCheckpointAccessors(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ex, err := NewExporter(context.Background(), &out, testHeader(), Limits{})
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
	if _, err := nilExporter.Finalize(context.Background()); !errors.Is(err, ErrFinalized) {
		t.Fatalf("nil exporter finalize = %v", err)
	}
}

func TestEveryArchiveTruncationFailsClosed(t *testing.T) {
	t.Parallel()

	data := archiveBytes(t, Record{Kind: "kind", Key: "key", Size: 5, Body: strings.NewReader("value")})
	for cut := 0; cut < len(data); cut++ {
		im, err := NewImporter(context.Background(), bytes.NewReader(data[:cut]), Limits{}, acceptExact)
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
	im, err := NewImporter(context.Background(), bytes.NewReader(unknownFrame), Limits{}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := im.Next(context.Background(), &memorySink{}); !errors.Is(err, ErrMalformed) {
		t.Fatalf("unknown frame = %v", err)
	}

	badSequence := append([]byte(nil), data...)
	binary.BigEndian.PutUint64(badSequence[headerLen+1:headerLen+9], 1)
	im, err = NewImporter(context.Background(), bytes.NewReader(badSequence), Limits{}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := im.Next(context.Background(), &memorySink{}); !errors.Is(err, ErrSequence) {
		t.Fatalf("sequence = %v", err)
	}

	im, err = NewImporter(context.Background(), bytes.NewReader(data), Limits{MaxRecords: 1, MaxRecordBytes: 1, MaxTotalBytes: 2}, acceptExact)
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
		im, err := NewImporter(context.Background(), bytes.NewReader(data), Limits{}, acceptExact)
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

	if _, err := NewImporter(context.Background(), nil, Limits{}, acceptExact); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil reader = %v", err)
	}
	if _, err := NewImporter(context.Background(), strings.NewReader(""), Limits{}, nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil compatibility = %v", err)
	}
	data := archiveBytes(t)
	wrongMagic := append([]byte(nil), data...)
	wrongMagic[0] ^= 1
	if _, err := NewImporter(context.Background(), bytes.NewReader(wrongMagic), Limits{}, acceptExact); !errors.Is(err, ErrMalformed) {
		t.Fatalf("magic = %v", err)
	}
	wrongVersion := append([]byte(nil), data...)
	binary.BigEndian.PutUint16(wrongVersion[8:10], WireVersion+1)
	if _, err := NewImporter(context.Background(), bytes.NewReader(wrongVersion), Limits{}, acceptExact); !errors.Is(err, ErrIncompatible) {
		t.Fatalf("wire version = %v", err)
	}
	im, err := NewImporter(context.Background(), bytes.NewReader(data), Limits{}, acceptExact)
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

func TestImporterDistinguishesIOFailureFromTruncation(t *testing.T) {
	t.Parallel()

	data := archiveBytes(t, Record{Kind: "x", Size: 1, Body: strings.NewReader("a")})
	if _, err := NewImporter(context.Background(), &failAfterReader{r: bytes.NewReader(data), remaining: 5}, Limits{}, acceptExact); !errors.Is(err, ErrIO) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("header I/O = %v", err)
	}
	headerLen := len(encodeHeader(testHeader()))
	prefixLen := len(encodeRecordPrefix(RecordMeta{Kind: "x", Size: 1}))
	im, err := NewImporter(context.Background(), &failAfterReader{r: bytes.NewReader(data), remaining: headerLen + prefixLen}, Limits{}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := im.Next(context.Background(), &memorySink{}); !errors.Is(err, ErrIO) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("payload I/O = %v", err)
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
	im, err := NewImporter(context.Background(), bytes.NewReader(archiveBytes(t)), Limits{}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := im.Next(ctx, &memorySink{}); !errors.Is(err, ErrIO) || errors.Is(err, context.Canceled) || !causeIs(err, context.Canceled) {
		t.Fatalf("canceled next = %v", err)
	}
}

func TestImporterCancellationBetweenPayloadChunksIsIOFailure(t *testing.T) {
	t.Parallel()

	const size = copyBufferBytes + 1
	data := archiveBytes(t, Record{Kind: "x", Size: size, Body: strings.NewReader(strings.Repeat("x", size))})
	im, err := NewImporter(context.Background(), bytes.NewReader(data), Limits{}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	sink := &cancelOnWriteSink{cancel: cancel}
	_, err = im.Next(ctx, sink)
	if !errors.Is(err, ErrIO) || errors.Is(err, context.Canceled) || !causeIs(err, context.Canceled) || sink.aborts != 1 {
		t.Fatalf("mid-payload cancellation = %v, aborts=%d", err, sink.aborts)
	}
}

func resignCheckpoint(data []byte) {
	digest := sha256.Sum256(data[:len(data)-sha256.Size])
	copy(data[len(data)-sha256.Size:], digest[:])
}

func TestCheckpointParserClassifications(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ex, err := NewExporter(context.Background(), &out, testHeader(), Limits{})
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

func TestCheckpointRejectsResignedImpossibleState(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ex, err := NewExporter(context.Background(), &out, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	empty, err := ex.cp.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	headerLen := int(binary.BigEndian.Uint16(empty[10:12]))
	position := 12 + headerLen
	payloadPosition := position + 16
	offsetPosition := position + 24
	chainPosition := position + 32
	for _, mutate := range []func([]byte){
		func(b []byte) { binary.BigEndian.PutUint64(b[payloadPosition:payloadPosition+8], 1) },
		func(b []byte) { binary.BigEndian.PutUint64(b[offsetPosition:offsetPosition+8], uint64(headerLen+1)) },
		func(b []byte) { b[chainPosition] ^= 1 },
	} {
		forged := append([]byte(nil), empty...)
		mutate(forged)
		resignCheckpoint(forged)
		if _, err := ParseCheckpoint(forged, Limits{}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("forged empty checkpoint = %v", err)
		}
	}

	cp, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 1, Body: strings.NewReader("a")})
	if err != nil {
		t.Fatal(err)
	}
	one, err := cp.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	headerLen = int(binary.BigEndian.Uint16(one[10:12]))
	offsetPosition = 12 + headerLen + 24
	for _, offset := range []uint64{0, math.MaxUint64} {
		forged := append([]byte(nil), one...)
		binary.BigEndian.PutUint64(forged[offsetPosition:offsetPosition+8], offset)
		resignCheckpoint(forged)
		if _, err := ParseCheckpoint(forged, Limits{MaxRecords: 2, MaxRecordBytes: math.MaxInt64, MaxTotalBytes: math.MaxUint64}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("forged offset %d = %v", offset, err)
		}
	}
}
