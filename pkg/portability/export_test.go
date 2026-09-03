package portability

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func TestWireV1EmptyArchiveGolden(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ex, err := NewExporter(&out, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ex.Finalize(); err != nil {
		t.Fatal(err)
	}
	want, err := hex.DecodeString(
		"474f54544850310a000100086578706f72742d3100076578616d706c6500000007" +
			"ff00000000000000000000000000000000" +
			"cdcf8383d6c422d112c0c170bd81d373a712afcc5250d448d88fbedae1cd2c18",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), want) {
		t.Fatalf("wire V1 changed:\n got %x\nwant %x", out.Bytes(), want)
	}
}

func TestWireV1RecordAndCheckpointGolden(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ex, err := NewExporter(&out, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	cp, err := ex.WriteRecord(context.Background(), Record{Kind: "post", Key: "1", Size: 3, Body: strings.NewReader("one")})
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err := cp.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ex.Finalize(); err != nil {
		t.Fatal(err)
	}
	wantArchive, err := hex.DecodeString(
		"474f54544850310a000100086578706f72742d3100076578616d706c6500000007" +
			"0100000000000000000004706f737400013100000000000000036f6e65" +
			"7692c3ad3540bb803c020b3aee66cd8887123234ea0c6e7143c0add73ff431ed" +
			"ff00000000000000010000000000000003" +
			"a9517b3ac8f5644f831fe898241fa161fb67f95458823461d3dcc24c4ce71309",
	)
	if err != nil {
		t.Fatal(err)
	}
	wantCheckpoint, err := hex.DecodeString(
		"47505443484b310a00010021" +
			"474f54544850310a000100086578706f72742d3100076578616d706c6500000007" +
			"000000000000000100000000000000010000000000000003000000000000005e" +
			"a9517b3ac8f5644f831fe898241fa161fb67f95458823461d3dcc24c4ce71309" +
			"cde487ddd1a09d3932b9f2e9c0082bddad473cfb60ba3883edd11d206cceca24",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), wantArchive) {
		t.Fatalf("record wire V1 changed:\n got %x\nwant %x", out.Bytes(), wantArchive)
	}
	if !bytes.Equal(checkpoint, wantCheckpoint) {
		t.Fatalf("checkpoint V1 changed:\n got %x\nwant %x", checkpoint, wantCheckpoint)
	}
}

func testHeader() Header {
	return Header{WireVersion: WireVersion, ArchiveID: "export-1", Schema: "example", SchemaVersion: 7}
}

func TestExporterWritesEmptyAndMultiRecordArchives(t *testing.T) {
	t.Parallel()

	for _, records := range [][]Record{
		nil,
		{{Kind: "post", Key: "1", Size: 3, Body: strings.NewReader("one")}, {Kind: "blob", Key: "2", Size: 0, Body: strings.NewReader("")}},
	} {
		var out bytes.Buffer
		ex, err := NewExporter(&out, testHeader(), Limits{})
		if err != nil {
			t.Fatalf("new exporter: %v", err)
		}
		for _, record := range records {
			if _, err := ex.WriteRecord(context.Background(), record); err != nil {
				t.Fatalf("write record: %v", err)
			}
		}
		manifest, err := ex.Finalize()
		if err != nil {
			t.Fatalf("finalize: %v", err)
		}
		if manifest.Records != uint64(len(records)) || out.Len() == 0 {
			t.Fatalf("manifest/output = %#v/%d", manifest, out.Len())
		}
		if _, err := ex.Finalize(); !errors.Is(err, ErrFinalized) {
			t.Fatalf("second finalize = %v, want ErrFinalized", err)
		}
	}
}

func TestExporterRejectsShortLongAndOverLimitBodies(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		record     Record
		limits     Limits
		want       error
		wantPoison bool
	}{
		{name: "short", record: Record{Kind: "x", Size: 2, Body: strings.NewReader("a")}, want: ErrTruncated, wantPoison: true},
		{name: "long", record: Record{Kind: "x", Size: 1, Body: strings.NewReader("ab")}, want: ErrInvalid, wantPoison: true},
		{name: "record limit", record: Record{Kind: "x", Size: 3, Body: strings.NewReader("abc")}, limits: Limits{MaxRecordBytes: 2, MaxTotalBytes: 4}, want: ErrLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			ex, err := NewExporter(&out, testHeader(), tc.limits)
			if err != nil {
				t.Fatalf("new exporter: %v", err)
			}
			if _, err := ex.WriteRecord(context.Background(), tc.record); !errors.Is(err, tc.want) {
				t.Fatalf("write = %v, want %v", err, tc.want)
			}
			_, err = ex.WriteRecord(context.Background(), Record{Kind: "x", Body: strings.NewReader("")})
			if tc.wantPoison && !errors.Is(err, ErrFinalized) {
				t.Fatalf("write after partial failure = %v, want ErrFinalized", err)
			}
			if !tc.wantPoison && err != nil {
				t.Fatalf("write after preflight rejection = %v, want reusable exporter", err)
			}
		})
	}
}

func TestExporterResumeMatchesUninterruptedArchive(t *testing.T) {
	t.Parallel()

	records := []Record{
		{Kind: "one", Key: "a", Size: 5, Body: strings.NewReader("first")},
		{Kind: "two", Key: "b", Size: 6, Body: strings.NewReader("second")},
	}
	var uninterrupted bytes.Buffer
	full, err := NewExporter(&uninterrupted, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range records {
		if _, err := full.WriteRecord(context.Background(), record); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := full.Finalize(); err != nil {
		t.Fatal(err)
	}

	var resumed bytes.Buffer
	first, err := NewExporter(&resumed, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err := first.WriteRecord(context.Background(), Record{Kind: "one", Key: "a", Size: 5, Body: strings.NewReader("first")})
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.Offset() != uint64(resumed.Len()) {
		t.Fatalf("offset = %d, bytes = %d", checkpoint.Offset(), resumed.Len())
	}
	continued, err := ResumeExporter(&resumed, checkpoint, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := continued.WriteRecord(context.Background(), Record{Kind: "two", Key: "b", Size: 6, Body: strings.NewReader("second")}); err != nil {
		t.Fatal(err)
	}
	if _, err := continued.Finalize(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(resumed.Bytes(), uninterrupted.Bytes()) {
		t.Fatal("resumed archive differs from uninterrupted archive")
	}
}

type cutoffBuffer struct {
	bytes.Buffer
	remaining int
}

func (w *cutoffBuffer) Write(p []byte) (int, error) {
	if w.remaining <= 0 {
		return 0, errors.New("cutoff")
	}
	n := len(p)
	if n > w.remaining {
		n = w.remaining
	}
	w.remaining -= n
	written, _ := w.Buffer.Write(p[:n])
	if n < len(p) {
		return written, errors.New("cutoff")
	}
	return written, nil
}

func TestExporterInitialCheckpointRecoversFirstRecordFailure(t *testing.T) {
	t.Parallel()

	prefix := encodeRecordPrefix(RecordMeta{Sequence: 0, Kind: "x", Size: 3})
	target := &cutoffBuffer{remaining: len(encodeHeader(testHeader())) + len(prefix) + 1}
	ex, err := NewExporter(target, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	initial := ex.Checkpoint()
	if initial.Records() != 0 || initial.Offset() != uint64(len(encodeHeader(testHeader()))) {
		t.Fatalf("initial checkpoint = %#v", initial)
	}
	if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 3, Body: strings.NewReader("abc")}); !errors.Is(err, ErrIO) {
		t.Fatalf("partial write = %v", err)
	}
	if ex.Checkpoint() != initial {
		t.Fatal("failed write advanced checkpoint")
	}
	target.Buffer.Truncate(int(initial.Offset()))
	target.remaining = 1 << 20
	resumed, err := ResumeExporter(target, initial, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resumed.WriteRecord(context.Background(), Record{Kind: "x", Size: 3, Body: strings.NewReader("abc")}); err != nil {
		t.Fatal(err)
	}
	if _, err := resumed.Finalize(); err != nil {
		t.Fatal(err)
	}
	want := archiveBytes(t, Record{Kind: "x", Size: 3, Body: strings.NewReader("abc")})
	if !bytes.Equal(target.Bytes(), want) {
		t.Fatal("recovered first-record archive differs")
	}
}
