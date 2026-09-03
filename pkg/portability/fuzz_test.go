package portability

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"testing"
)

func expectedSingleRecordManifest(kind, key string, payload []byte) Manifest {
	digest := sha256.Sum256(payload)
	initial := sha256.Sum256(encodeHeader(testHeader()))
	chain := nextChain(initial, encodeRecordPrefix(RecordMeta{Sequence: 0, Kind: kind, Key: key, Size: uint64(len(payload))}), digest)
	return Manifest{Records: 1, PayloadBytes: uint64(len(payload)), Chain: chain}
}

func assertSingleCommittedRecord(t *testing.T, sink *memorySink, payload []byte) {
	t.Helper()
	if sink.commits != 1 || sink.aborts != 0 || len(sink.committed) != 1 {
		t.Fatalf("commit state: commits=%d aborts=%d entries=%d", sink.commits, sink.aborts, len(sink.committed))
	}
	got, ok := sink.committed[0]
	if !ok {
		t.Fatal("record sequence 0 was not committed")
	}
	if !bytes.Equal([]byte(got), payload) {
		t.Fatalf("payload mismatch: got=%x want=%x", []byte(got), payload)
	}
}

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
		checkpoint, err := im.Next(context.Background(), sink)
		if err != nil {
			t.Fatal(err)
		}
		completeCheckpoint, err := im.Next(context.Background(), sink)
		if !errors.Is(err, ErrComplete) {
			t.Fatalf("finish: %v", err)
		}
		manifest, err := im.Manifest()
		expected := expectedSingleRecordManifest(kind, key, payload)
		if err != nil || manifest != expected || checkpoint.records != 1 || checkpoint.payloadBytes != uint64(len(payload)) || completeCheckpoint != checkpoint {
			t.Fatalf("manifest=%#v error=%v", manifest, err)
		}
		assertSingleCommittedRecord(t, sink, payload)
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
				if manifest.Records != uint64(sink.commits) || manifest.Records != checkpoint.Records() || manifest.PayloadBytes != checkpoint.PayloadBytes() || manifest.Chain != checkpoint.chain || len(sink.committed) != sink.commits {
					t.Fatalf("complete state mismatch: manifest=%#v checkpoint=%#v commits=%d", manifest, checkpoint, sink.commits)
				}
				for sequence := uint64(0); sequence < manifest.Records; sequence++ {
					if _, ok := sink.committed[sequence]; !ok {
						t.Fatalf("missing committed sequence %d", sequence)
					}
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
			expected := expectedSingleRecordManifest("opaque", "", payload)
			if manifestErr != nil || manifest != expected {
				t.Fatalf("valid oracle mismatch: manifest=%#v expected=%#v error=%v", manifest, expected, manifestErr)
			}
			assertSingleCommittedRecord(t, sink, payload)
		case 1:
			if complete || sink.commits != 0 || sink.aborts != 1 || len(sink.committed) != 0 {
				t.Fatalf("integrity mutation completed=%v commits=%d aborts=%d entries=%d", complete, sink.commits, sink.aborts, len(sink.committed))
			}
			if _, manifestErr := im.Manifest(); !errors.Is(manifestErr, ErrIncomplete) {
				t.Fatalf("integrity mutation manifest=%v", manifestErr)
			}
		case 2:
			if complete {
				t.Fatal("truncated archive completed")
			}
			if sink.commits > 1 || len(sink.committed) != sink.commits {
				t.Fatalf("truncation commit state: commits=%d entries=%d", sink.commits, len(sink.committed))
			}
			if sink.commits == 1 {
				assertSingleCommittedRecord(t, sink, payload)
			}
			if _, manifestErr := im.Manifest(); !errors.Is(manifestErr, ErrIncomplete) {
				t.Fatalf("truncation manifest=%v", manifestErr)
			}
		case 3:
			if complete {
				t.Fatal("trailing-data archive completed")
			}
			assertSingleCommittedRecord(t, sink, payload)
			if _, manifestErr := im.Manifest(); !errors.Is(manifestErr, ErrIncomplete) {
				t.Fatalf("trailing-data manifest=%v", manifestErr)
			}
		}
	})
}
