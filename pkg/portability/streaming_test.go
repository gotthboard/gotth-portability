package portability

import (
	"bytes"
	"context"
	"io"
	"testing"
)

type maxWriteRecorder struct {
	max int
	n   uint64
}

func (w *maxWriteRecorder) Write(p []byte) (int, error) {
	if len(p) > w.max {
		w.max = len(p)
	}
	w.n += uint64(len(p))
	return len(p), nil
}

type repeatedByteReader struct{ remaining uint64 }

func (r *repeatedByteReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	if uint64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	for i := range p {
		p[i] = 'x'
	}
	r.remaining -= uint64(len(p))
	return len(p), nil
}

type recordingSink struct{ writer *maxWriteRecorder }

func (s recordingSink) Begin(context.Context, Header, RecordMeta) (RecordSink, error) {
	return recordingRecord{s.writer}, nil
}

type recordingRecord struct{ *maxWriteRecorder }

func (recordingRecord) Commit(context.Context) error { return nil }
func (recordingRecord) Abort(context.Context) error  { return nil }

func TestLargePayloadUsesBoundedWrites(t *testing.T) {
	t.Parallel()

	const size = uint64(5<<20 + 17)
	archive := &maxWriteRecorder{}
	ex, err := NewExporter(context.Background(), archive, testHeader(), Limits{MaxRecordBytes: size, MaxTotalBytes: size})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ex.WriteRecord(context.Background(), Record{Kind: "large", Size: size, Body: &repeatedByteReader{remaining: size}}); err != nil {
		t.Fatal(err)
	}
	if _, err := ex.Finalize(context.Background()); err != nil {
		t.Fatal(err)
	}
	if archive.max > copyBufferBytes {
		t.Fatalf("largest export write = %d, bound = %d", archive.max, copyBufferBytes)
	}

	var encoded bytes.Buffer
	ex, err = NewExporter(context.Background(), &encoded, testHeader(), Limits{MaxRecordBytes: size, MaxTotalBytes: size})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ex.WriteRecord(context.Background(), Record{Kind: "large", Size: size, Body: &repeatedByteReader{remaining: size}}); err != nil {
		t.Fatal(err)
	}
	if _, err := ex.Finalize(context.Background()); err != nil {
		t.Fatal(err)
	}
	im, err := NewImporter(context.Background(), bytes.NewReader(encoded.Bytes()), Limits{MaxRecordBytes: size, MaxTotalBytes: size}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	staged := &maxWriteRecorder{}
	if _, err := im.Next(context.Background(), recordingSink{staged}); err != nil {
		t.Fatal(err)
	}
	if staged.max > copyBufferBytes || staged.n != size {
		t.Fatalf("import writes max/total = %d/%d", staged.max, staged.n)
	}
}
