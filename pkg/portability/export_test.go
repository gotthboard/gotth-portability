package portability

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

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
