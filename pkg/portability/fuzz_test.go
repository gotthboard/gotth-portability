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
		im, err := NewImporter(context.Background(), bytes.NewReader(data), Limits{}, acceptExact)
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
		manifest, err := im.Manifest()
		if err != nil || manifest.Records != 1 || manifest.PayloadBytes != uint64(len(payload)) {
			t.Fatalf("manifest=%#v error=%v", manifest, err)
		}
		if !bytes.Equal([]byte(sink.committed[0]), payload) {
			t.Fatal("payload mismatch")
		}
	})
}

func FuzzCheckpointParserNeverPanics(f *testing.F) {
	var out bytes.Buffer
	ex, err := NewExporter(context.Background(), &out, testHeader(), Limits{})
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

func FuzzArbitraryArchiveTerminatesWithinRecordLimit(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("GOTTHP1\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 8192 {
			t.Skip()
		}
		im, err := NewImporter(context.Background(), bytes.NewReader(data), Limits{MaxRecords: 16, MaxRecordBytes: 4096, MaxTotalBytes: 8192}, func(context.Context, Header) error { return nil })
		if err != nil {
			return
		}
		sink := &memorySink{}
		for step := 0; step < 18; step++ {
			checkpoint, nextErr := im.Next(context.Background(), sink)
			if errors.Is(nextErr, ErrComplete) {
				manifest, manifestErr := im.Manifest()
				if manifestErr != nil {
					t.Fatalf("complete without manifest: %v", manifestErr)
				}
				if manifest.Records != uint64(sink.commits) || manifest.Records != checkpoint.Records() || manifest.PayloadBytes != checkpoint.PayloadBytes() {
					t.Fatalf("complete state mismatch: manifest=%#v checkpoint=%#v commits=%d", manifest, checkpoint, sink.commits)
				}
				return
			}
			if nextErr != nil {
				return
			}
		}
		t.Fatal("untrusted archive exceeded bounded record count without error")
	})
}

func FuzzValidArchiveMutationOracle(f *testing.F) {
	f.Add([]byte("payload"), uint8(0), uint16(0))
	f.Add([]byte{}, uint8(1), uint16(0))
	f.Add([]byte("x"), uint8(2), uint16(4))
	f.Add([]byte("tail"), uint8(3), uint16(0))
	f.Fuzz(func(t *testing.T, payload []byte, mode uint8, position uint16) {
		if len(payload) > 4096 {
			t.Skip()
		}
		original := archiveBytes(t, Record{Kind: "opaque", Size: uint64(len(payload)), Body: bytes.NewReader(payload)})
		candidate := append([]byte(nil), original...)
		mutation := mode % 4
		switch mutation {
		case 0: // exact valid archive
		case 1: // payload byte, or digest byte for an empty payload
			offset := len(encodeHeader(testHeader())) + len(encodeRecordPrefix(RecordMeta{Sequence: 0, Kind: "opaque", Size: uint64(len(payload))}))
			if len(payload) > 0 {
				offset += int(position) % len(payload)
			}
			candidate[offset] ^= 1
		case 2: // arbitrary truncation, including a valid committed prefix
			candidate = candidate[:int(position)%len(candidate)]
		case 3: // otherwise-valid archive with trailing data
			candidate = append(candidate, byte(position))
		}

		im, err := NewImporter(context.Background(), bytes.NewReader(candidate), Limits{}, acceptExact)
		if err != nil {
			if mutation == 0 {
				t.Fatalf("valid archive rejected: %v", err)
			}
			return
		}
		sink := &memorySink{}
		complete := false
		for step := 0; step < 3; step++ {
			_, nextErr := im.Next(context.Background(), sink)
			if errors.Is(nextErr, ErrComplete) {
				complete = true
				break
			}
			if nextErr != nil {
				break
			}
		}

		switch mutation {
		case 0:
			if !complete {
				t.Fatal("valid archive did not complete")
			}
			manifest, manifestErr := im.Manifest()
			if manifestErr != nil || manifest.Records != 1 || manifest.PayloadBytes != uint64(len(payload)) || !bytes.Equal([]byte(sink.committed[0]), payload) {
				t.Fatalf("valid oracle mismatch: manifest=%#v error=%v committed=%q", manifest, manifestErr, sink.committed[0])
			}
		case 1:
			if complete || sink.commits != 0 {
				t.Fatalf("integrity mutation completed=%v commits=%d", complete, sink.commits)
			}
		case 2:
			if complete {
				t.Fatal("truncated archive completed")
			}
		case 3:
			if complete || sink.commits != 1 {
				t.Fatalf("trailing-data archive completed=%v commits=%d", complete, sink.commits)
			}
		}
	})
}
