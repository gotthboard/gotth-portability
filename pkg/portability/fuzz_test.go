package portability

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func FuzzArchiveRoundTrip(f *testing.F) {
	f.Add("kind", "key", []byte("payload"))
	f.Add("x", "", []byte{})
	f.Fuzz(func(t *testing.T, kind, key string, payload []byte) {
		if len(payload) > 4096 {
			t.Skip()
		}
		record := RecordMeta{Kind: kind, Key: key, Size: uint64(len(payload))}
		if err := record.validate(); err != nil {
			return
		}
		data := archiveBytes(t, Record{Kind: kind, Key: key, Size: uint64(len(payload)), Body: bytes.NewReader(payload)})
		im, err := NewImporter(bytes.NewReader(data), Limits{}, acceptExact)
		if err != nil {
			t.Fatal(err)
		}
		sink := &memorySink{}
		if _, err := im.Next(context.Background(), sink); err != nil {
			t.Fatal(err)
		}
		if _, err := im.Next(context.Background(), sink); !errors.Is(err, ErrComplete) {
			t.Fatalf("finish: %v", err)
		}
		if !bytes.Equal([]byte(sink.committed[0]), payload) {
			t.Fatal("payload mismatch")
		}
	})
}

func FuzzCheckpointParserNeverPanics(f *testing.F) {
	var out bytes.Buffer
	ex, err := NewExporter(&out, testHeader(), Limits{})
	if err != nil {
		f.Fatal(err)
	}
	seed, err := ex.cp.MarshalBinary()
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte("not a checkpoint"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 4096 {
			t.Skip()
		}
		_, _ = ParseCheckpoint(data, Limits{})
	})
}

func FuzzMalformedArchiveNeverCompletesSilently(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("GOTTHP1\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 8192 {
			t.Skip()
		}
		im, err := NewImporter(bytes.NewReader(data), Limits{MaxRecords: 16, MaxRecordBytes: 4096, MaxTotalBytes: 8192}, func(Header) error { return nil })
		if err != nil {
			return
		}
		sink := &memorySink{}
		for step := 0; step < 18; step++ {
			_, err = im.Next(context.Background(), sink)
			if err != nil {
				return
			}
		}
		t.Fatal("untrusted archive exceeded bounded record count without error")
	})
}
