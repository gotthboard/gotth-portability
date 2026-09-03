package portability

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"testing"
	"time"
)

type perfWorkload struct {
	name    string
	records int
	payload uint64
	samples int
}

func TestPerformanceSamples(t *testing.T) {
	if os.Getenv("GOTTH_PORTABILITY_PERF") != "1" {
		t.Skip("set GOTTH_PORTABILITY_PERF=1 for admission measurements")
	}
	workloads := []perfWorkload{
		{name: "empty", records: 0, payload: 0, samples: 200},
		{name: "small-1x1KiB", records: 1, payload: 1 << 10, samples: 200},
		{name: "typical-16x64KiB", records: 16, payload: 64 << 10, samples: 80},
		{name: "large-4x1MiB", records: 4, payload: 1 << 20, samples: 20},
		{name: "pathological-1000-empty", records: 1000, payload: 0, samples: 10},
	}
	for _, workload := range workloads {
		durations := make([]time.Duration, workload.samples)
		for sample := range workload.samples {
			start := time.Now()
			if err := runPerfRoundTrip(workload); err != nil {
				t.Fatalf("%s sample %d: %v", workload.name, sample, err)
			}
			durations[sample] = time.Since(start)
		}
		sort.Slice(durations, func(a, b int) bool { return durations[a] < durations[b] })
		total := time.Duration(0)
		for _, duration := range durations {
			total += duration
		}
		throughput := float64(workload.samples) / total.Seconds()
		t.Logf("PERF name=%s samples=%d p50=%s p95=%s p99=%s throughput=%.2f archives/s payload_bytes=%d records=%d",
			workload.name, workload.samples, percentile(durations, 50), percentile(durations, 95), percentile(durations, 99), throughput, workload.payload, workload.records)
	}
}

func runPerfRoundTrip(workload perfWorkload) error {
	header := Header{WireVersion: WireVersion, ArchiveID: "performance", Schema: "performance", SchemaVersion: 1}
	total := workload.payload * uint64(workload.records)
	limits := Limits{MaxRecords: uint64(workload.records) + 1, MaxRecordBytes: max64(workload.payload, 1), MaxTotalBytes: max64(total, 1)}
	var archive bytes.Buffer
	ex, err := NewExporter(context.Background(), &archive, header, limits)
	if err != nil {
		return err
	}
	for record := 0; record < workload.records; record++ {
		if _, err := ex.WriteRecord(context.Background(), Record{Kind: "opaque", Key: fmt.Sprint(record), Size: workload.payload, Body: &repeatedByteReader{remaining: workload.payload}}); err != nil {
			return err
		}
	}
	if _, err := ex.Finalize(context.Background()); err != nil {
		return err
	}
	im, err := NewImporter(context.Background(), bytes.NewReader(archive.Bytes()), limits, func(context.Context, Header) error { return nil })
	if err != nil {
		return err
	}
	sink := performanceSink{}
	for {
		_, err := im.Next(context.Background(), sink)
		if errors.Is(err, ErrComplete) {
			break
		}
		if err != nil {
			return err
		}
	}
	_, err = im.Manifest()
	return err
}

func percentile(sorted []time.Duration, percent int) time.Duration {
	index := (len(sorted)*percent + 99) / 100
	if index == 0 {
		index = 1
	}
	return sorted[index-1]
}

func max64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

type performanceSink struct{}

func (performanceSink) Begin(context.Context, Header, RecordMeta) (RecordSink, error) {
	return performanceRecord{}, nil
}

type performanceRecord struct{}

func (performanceRecord) Write(p []byte) (int, error)  { return len(p), nil }
func (performanceRecord) Commit(context.Context) error { return nil }
func (performanceRecord) Abort(context.Context) error  { return nil }

var allocationWorkloads = []perfWorkload{
	{name: "zero-records", records: 0, payload: 0},
	{name: "one-empty", records: 1, payload: 0},
	{name: "1000-empty", records: 1000, payload: 0},
	{name: "1x1KiB", records: 1, payload: 1 << 10},
	{name: "1x1MiB", records: 1, payload: 1 << 20},
	{name: "1x16MiB", records: 1, payload: 16 << 20},
}

func benchmarkLimits(workload perfWorkload) Limits {
	total := uint64(workload.records) * workload.payload
	return Limits{MaxRecords: uint64(workload.records) + 1, MaxRecordBytes: max64(workload.payload, 1), MaxTotalBytes: max64(total, 1)}
}

func BenchmarkExport(b *testing.B) {
	for _, workload := range allocationWorkloads {
		b.Run(workload.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(workload.records) * int64(workload.payload))
			limits := benchmarkLimits(workload)
			for b.Loop() {
				ex, err := NewExporter(context.Background(), io.Discard, testHeader(), limits)
				if err != nil {
					b.Fatal(err)
				}
				for record := 0; record < workload.records; record++ {
					if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: workload.payload, Body: &repeatedByteReader{remaining: workload.payload}}); err != nil {
						b.Fatal(err)
					}
				}
				if _, err := ex.Finalize(context.Background()); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkImport(b *testing.B) {
	for _, workload := range allocationWorkloads {
		b.Run(workload.name, func(b *testing.B) {
			limits := benchmarkLimits(workload)
			var archive bytes.Buffer
			ex, err := NewExporter(context.Background(), &archive, testHeader(), limits)
			if err != nil {
				b.Fatal(err)
			}
			for record := 0; record < workload.records; record++ {
				if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: workload.payload, Body: &repeatedByteReader{remaining: workload.payload}}); err != nil {
					b.Fatal(err)
				}
			}
			if _, err := ex.Finalize(context.Background()); err != nil {
				b.Fatal(err)
			}
			data := archive.Bytes()
			b.ReportAllocs()
			b.SetBytes(int64(workload.records) * int64(workload.payload))
			b.ResetTimer()
			for b.Loop() {
				im, err := NewImporter(context.Background(), bytes.NewReader(data), limits, func(context.Context, Header) error { return nil })
				if err != nil {
					b.Fatal(err)
				}
				for {
					_, err = im.Next(context.Background(), performanceSink{})
					if errors.Is(err, ErrComplete) {
						break
					}
					if err != nil {
						b.Fatal(err)
					}
				}
				if _, err := im.Manifest(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
