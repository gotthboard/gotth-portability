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
	ex, err := NewExporter(&archive, header, limits)
	if err != nil {
		return err
	}
	for record := 0; record < workload.records; record++ {
		if _, err := ex.WriteRecord(context.Background(), Record{Kind: "opaque", Key: fmt.Sprint(record), Size: workload.payload, Body: &repeatedByteReader{remaining: workload.payload}}); err != nil {
			return err
		}
	}
	if _, err := ex.Finalize(); err != nil {
		return err
	}
	im, err := NewImporter(bytes.NewReader(archive.Bytes()), limits, func(Header) error { return nil })
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

func BenchmarkExport(b *testing.B) {
	for _, size := range []uint64{0, 1 << 10, 1 << 20, 16 << 20} {
		b.Run(fmt.Sprintf("payload-%d", size), func(b *testing.B) {
			for b.Loop() {
				ex, err := NewExporter(io.Discard, testHeader(), Limits{MaxRecords: 1, MaxRecordBytes: max64(size, 1), MaxTotalBytes: max64(size, 1)})
				if err != nil {
					b.Fatal(err)
				}
				if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: size, Body: &repeatedByteReader{remaining: size}}); err != nil {
					b.Fatal(err)
				}
				if _, err := ex.Finalize(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
