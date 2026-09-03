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
		{name: "schema limit minus one", value: strings.Repeat("s", maxSchemaBytes-1), max: maxSchemaBytes, valid: true},
		{name: "schema limit", value: strings.Repeat("s", maxSchemaBytes), max: maxSchemaBytes, valid: true},
		{name: "schema limit plus one", value: strings.Repeat("s", maxSchemaBytes+1), max: maxSchemaBytes},
		{name: "schema materially beyond", value: strings.Repeat("s", maxSchemaBytes*4), max: maxSchemaBytes},
		{name: "kind limit minus one", value: strings.Repeat("k", maxKindBytes-1), max: maxKindBytes, valid: true},
		{name: "kind limit", value: strings.Repeat("k", maxKindBytes), max: maxKindBytes, valid: true},
		{name: "kind limit plus one", value: strings.Repeat("k", maxKindBytes+1), max: maxKindBytes},
		{name: "kind materially beyond", value: strings.Repeat("k", maxKindBytes*4), max: maxKindBytes},
		{name: "key limit minus one", value: strings.Repeat("q", maxKeyBytes-1), max: maxKeyBytes, valid: true},
		{name: "key limit", value: strings.Repeat("q", maxKeyBytes), max: maxKeyBytes, valid: true},
		{name: "key limit plus one", value: strings.Repeat("q", maxKeyBytes+1), max: maxKeyBytes},
		{name: "key materially beyond", value: strings.Repeat("q", maxKeyBytes*4), max: maxKeyBytes},
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
		ex, err := NewExporter(context.Background(), &out, testHeader(), Limits{MaxRecords: 3, MaxRecordBytes: 3, MaxTotalBytes: 6})
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
	ex, err := NewExporter(context.Background(), &out, testHeader(), Limits{MaxRecords: 2, MaxRecordBytes: 3, MaxTotalBytes: 5})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 3, Body: strings.NewReader("abc")}); err != nil {
		t.Fatal(err)
	}
	if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 3, Body: strings.NewReader("def")}); !errors.Is(err, ErrLimit) {
		t.Fatalf("total limit plus one = %v", err)
	}
	for _, tc := range []struct {
		name  string
		sizes []int
		want  error
	}{
		{name: "total limit minus one", sizes: []int{2, 2}},
		{name: "total limit", sizes: []int{2, 3}},
		{name: "total limit plus one", sizes: []int{3, 3}, want: ErrLimit},
		{name: "total materially beyond", sizes: []int{5, 5}, want: ErrLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var target bytes.Buffer
			ex, err := NewExporter(context.Background(), &target, testHeader(), Limits{MaxRecords: 3, MaxRecordBytes: 5, MaxTotalBytes: 5})
			if err != nil {
				t.Fatal(err)
			}
			for _, size := range tc.sizes {
				_, err = ex.WriteRecord(context.Background(), Record{Kind: "x", Size: uint64(size), Body: strings.NewReader(strings.Repeat("x", size))})
				if err != nil {
					break
				}
			}
			if tc.want == nil && err != nil {
				t.Fatalf("unexpected total error: %v", err)
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("total error = %v, want %v", err, tc.want)
			}
		})
	}

	var countOut bytes.Buffer
	counted, err := NewExporter(context.Background(), &countOut, testHeader(), Limits{MaxRecords: 2, MaxRecordBytes: 1, MaxTotalBytes: 3})
	if err != nil {
		t.Fatal(err)
	}
	for record := 0; record < 2; record++ {
		if _, err := counted.WriteRecord(context.Background(), Record{Kind: "x", Size: 1, Body: strings.NewReader("x")}); err != nil {
			t.Fatalf("record count %d: %v", record+1, err)
		}
	}
	if _, err := counted.WriteRecord(context.Background(), Record{Kind: "x", Body: strings.NewReader("")}); !errors.Is(err, ErrLimit) {
		t.Fatalf("record count plus one = %v", err)
	}

	if _, overflow := checkedAdd(math.MaxUint64-1, 3); !overflow {
		t.Fatal("checkedAdd missed unsigned overflow")
	}
}

func TestImporterRejectsRecordSizeBeforeOpeningSink(t *testing.T) {
	t.Parallel()

	data := archiveBytes(t, Record{Kind: "x", Size: 4, Body: strings.NewReader("data")})
	im, err := NewImporter(context.Background(), bytes.NewReader(data), Limits{MaxRecords: 1, MaxRecordBytes: 3, MaxTotalBytes: 10}, acceptExact)
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

func TestImporterRejectsCumulativeSizeBeforeOpeningNextSink(t *testing.T) {
	t.Parallel()

	data := archiveBytes(t,
		Record{Kind: "x", Size: 3, Body: strings.NewReader("one")},
		Record{Kind: "x", Size: 3, Body: strings.NewReader("two")},
	)
	im, err := NewImporter(context.Background(), bytes.NewReader(data), Limits{MaxRecords: 2, MaxRecordBytes: 3, MaxTotalBytes: 5}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	begins := 0
	backing := &memorySink{}
	sink := SinkFunc(func(ctx context.Context, header Header, meta RecordMeta) (RecordSink, error) {
		begins++
		return backing.Begin(ctx, header, meta)
	})
	if _, err := im.Next(context.Background(), sink); err != nil {
		t.Fatal(err)
	}
	if _, err := im.Next(context.Background(), sink); !errors.Is(err, ErrLimit) {
		t.Fatalf("cumulative limit = %v", err)
	}
	if begins != 1 {
		t.Fatalf("sink began %d times, want only first record", begins)
	}
}
