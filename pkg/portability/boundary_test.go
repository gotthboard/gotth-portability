package portability

import (
	"bytes"
	"context"
	"errors"
	"math"
	"strings"
	"testing"
)

func TestMetadataLimitsAtBoundary(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		value string
		max   int
		valid bool
	}{
		{name: "archive limit minus one", value: strings.Repeat("a", maxArchiveIDBytes-1), max: maxArchiveIDBytes, valid: true},
		{name: "archive limit", value: strings.Repeat("a", maxArchiveIDBytes), max: maxArchiveIDBytes, valid: true},
		{name: "archive limit plus one", value: strings.Repeat("a", maxArchiveIDBytes+1), max: maxArchiveIDBytes},
		{name: "archive materially beyond", value: strings.Repeat("a", maxArchiveIDBytes*4), max: maxArchiveIDBytes},
		{name: "kind limit minus one", value: strings.Repeat("k", maxKindBytes-1), max: maxKindBytes, valid: true},
		{name: "kind limit", value: strings.Repeat("k", maxKindBytes), max: maxKindBytes, valid: true},
		{name: "kind limit plus one", value: strings.Repeat("k", maxKindBytes+1), max: maxKindBytes},
		{name: "key limit", value: strings.Repeat("q", maxKeyBytes), max: maxKeyBytes, valid: true},
		{name: "key limit plus one", value: strings.Repeat("q", maxKeyBytes+1), max: maxKeyBytes},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateText("field", tc.value, tc.max, false)
			if tc.valid && err != nil {
				t.Fatalf("valid boundary rejected: %v", err)
			}
			if !tc.valid && !errors.Is(err, ErrInvalid) {
				t.Fatalf("invalid boundary = %v", err)
			}
		})
	}
}

func TestRecordAndTotalLimitsAtBoundary(t *testing.T) {
	t.Parallel()

	for _, size := range []uint64{2, 3, 4, 12} {
		var out bytes.Buffer
		ex, err := NewExporter(&out, testHeader(), Limits{MaxRecords: 3, MaxRecordBytes: 3, MaxTotalBytes: 6})
		if err != nil {
			t.Fatal(err)
		}
		_, err = ex.WriteRecord(context.Background(), Record{Kind: "x", Size: size, Body: strings.NewReader(strings.Repeat("x", int(size)))})
		if size <= 3 && err != nil {
			t.Fatalf("size %d rejected: %v", size, err)
		}
		if size > 3 && !errors.Is(err, ErrLimit) {
			t.Fatalf("size %d = %v, want ErrLimit", size, err)
		}
	}

	var out bytes.Buffer
	ex, err := NewExporter(&out, testHeader(), Limits{MaxRecords: 2, MaxRecordBytes: 3, MaxTotalBytes: 5})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 3, Body: strings.NewReader("abc")}); err != nil {
		t.Fatal(err)
	}
	if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 3, Body: strings.NewReader("def")}); !errors.Is(err, ErrLimit) {
		t.Fatalf("total limit plus one = %v", err)
	}

	overflow := ex.cp
	overflow.payloadBytes = math.MaxUint64 - 1
	overflow.records = 0
	overflow.nextSequence = 0
	resume, err := ResumeExporter(&bytes.Buffer{}, overflow, Limits{MaxRecords: 1, MaxRecordBytes: 3, MaxTotalBytes: math.MaxUint64})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resume.WriteRecord(context.Background(), Record{Kind: "x", Size: 3, Body: strings.NewReader("abc")}); !errors.Is(err, ErrLimit) {
		t.Fatalf("overflow = %v", err)
	}
}

func TestImporterRejectsRecordSizeBeforeOpeningSink(t *testing.T) {
	t.Parallel()

	data := archiveBytes(t, Record{Kind: "x", Size: 4, Body: strings.NewReader("data")})
	im, err := NewImporter(bytes.NewReader(data), Limits{MaxRecords: 1, MaxRecordBytes: 3, MaxTotalBytes: 10}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	begins := 0
	sink := SinkFunc(func(context.Context, Header, RecordMeta) (RecordSink, error) {
		begins++
		return nil, nil
	})
	if _, err := im.Next(context.Background(), sink); !errors.Is(err, ErrLimit) {
		t.Fatalf("size limit = %v", err)
	}
	if begins != 0 {
		t.Fatalf("sink began %d times", begins)
	}
}
