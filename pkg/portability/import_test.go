package portability

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type memorySink struct {
	staged     bytes.Buffer
	committed  map[uint64]string
	aborts     int
	commits    int
	failBegin  bool
	failWrite  bool
	failCommit bool
	failAbort  bool
}

func (s *memorySink) Begin(_ context.Context, _ Header, meta RecordMeta) (RecordSink, error) {
	if s.failBegin {
		return nil, errors.New("begin-secret")
	}
	s.staged.Reset()
	return &memoryRecordSink{owner: s, sequence: meta.Sequence}, nil
}

type memoryRecordSink struct {
	owner    *memorySink
	sequence uint64
}

func (s *memoryRecordSink) Write(p []byte) (int, error) {
	if s.owner.failWrite {
		return 0, errors.New("sink-secret")
	}
	return s.owner.staged.Write(p)
}

func (s *memoryRecordSink) Commit(context.Context) error {
	s.owner.commits++
	if s.owner.failCommit {
		return errors.New("commit-secret")
	}
	if s.owner.committed == nil {
		s.owner.committed = make(map[uint64]string)
	}
	s.owner.committed[s.sequence] = s.owner.staged.String()
	return nil
}

func (s *memoryRecordSink) Abort(context.Context) error {
	s.owner.aborts++
	s.owner.staged.Reset()
	if s.owner.failAbort {
		return errors.New("abort-secret")
	}
	return nil
}

func archiveBytes(t *testing.T, records ...Record) []byte {
	t.Helper()
	var out bytes.Buffer
	ex, err := NewExporter(context.Background(), &out, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range records {
		if _, err := ex.WriteRecord(context.Background(), record); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ex.Finalize(context.Background()); err != nil {
		t.Fatal(err)
	}
	return append([]byte(nil), out.Bytes()...)
}

func acceptExact(_ context.Context, header Header) error {
	if header.Schema != "example" || header.SchemaVersion != 7 {
		return errors.New("unsupported")
	}
	return nil
}

func TestImporterRoundTripAndCompletenessOracle(t *testing.T) {
	t.Parallel()

	data := archiveBytes(t,
		Record{Kind: "post", Key: "1", Size: 3, Body: strings.NewReader("one")},
		Record{Kind: "post", Key: "2", Size: 3, Body: strings.NewReader("two")},
	)
	im, err := NewImporter(context.Background(), bytes.NewReader(data), Limits{}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := im.Manifest(); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("early manifest = %v, want ErrIncomplete", err)
	}
	sink := &memorySink{}
	for i := uint64(0); i < 2; i++ {
		checkpoint, err := im.Next(context.Background(), sink)
		if err != nil {
			t.Fatalf("next %d: %v", i, err)
		}
		if checkpoint.Records() != i+1 {
			t.Fatalf("checkpoint records = %d", checkpoint.Records())
		}
	}
	if _, err := im.Next(context.Background(), sink); !errors.Is(err, ErrComplete) {
		t.Fatalf("footer next = %v, want ErrComplete", err)
	}
	manifest, err := im.Manifest()
	if err != nil || manifest.Records != 2 || manifest.PayloadBytes != 6 {
		t.Fatalf("manifest = %#v, %v", manifest, err)
	}
	if sink.committed[0] != "one" || sink.committed[1] != "two" || sink.commits != 2 || sink.aborts != 0 {
		t.Fatalf("sink = %#v", sink)
	}
}

func TestImporterRejectsBeforeOrInsteadOfCommit(t *testing.T) {
	t.Parallel()

	data := archiveBytes(t, Record{Kind: "post", Key: "1", Size: 3, Body: strings.NewReader("one")})
	t.Run("compatibility before sink", func(t *testing.T) {
		begins := 0
		_, err := NewImporter(context.Background(), bytes.NewReader(data), Limits{}, func(context.Context, Header) error {
			return errors.New("schema-secret")
		})
		if !errors.Is(err, ErrIncompatible) || strings.Contains(err.Error(), "secret") {
			t.Fatalf("compatibility error = %v", err)
		}
		if begins != 0 {
			t.Fatalf("begins = %d", begins)
		}
	})

	t.Run("payload corruption aborts", func(t *testing.T) {
		corrupt := append([]byte(nil), data...)
		headerLen := len(encodeHeader(testHeader()))
		prefixLen := len(encodeRecordPrefix(RecordMeta{Sequence: 0, Kind: "post", Key: "1", Size: 3}))
		corrupt[headerLen+prefixLen] ^= 0xff
		im, err := NewImporter(context.Background(), bytes.NewReader(corrupt), Limits{}, acceptExact)
		if err != nil {
			t.Fatal(err)
		}
		sink := &memorySink{}
		if _, err := im.Next(context.Background(), sink); !errors.Is(err, ErrIntegrity) {
			t.Fatalf("next = %v, want ErrIntegrity", err)
		}
		if sink.aborts != 1 || sink.commits != 0 {
			t.Fatalf("sink abort/commit = %d/%d", sink.aborts, sink.commits)
		}
	})

	t.Run("sink error is redacted", func(t *testing.T) {
		im, err := NewImporter(context.Background(), bytes.NewReader(data), Limits{}, acceptExact)
		if err != nil {
			t.Fatal(err)
		}
		sink := &memorySink{failWrite: true}
		_, err = im.Next(context.Background(), sink)
		if !errors.Is(err, ErrSink) || strings.Contains(err.Error(), "secret") || sink.aborts != 1 {
			t.Fatalf("sink failure = %v, aborts=%d", err, sink.aborts)
		}
	})
}

func TestImporterRejectsTruncationTrailingDataAndManifestCorruption(t *testing.T) {
	t.Parallel()

	data := archiveBytes(t, Record{Kind: "x", Size: 1, Body: strings.NewReader("a")})
	for _, tc := range []struct {
		name string
		data []byte
		want error
	}{
		{name: "truncated", data: data[:len(data)-1], want: ErrTruncated},
		{name: "trailing", data: append(append([]byte(nil), data...), 1), want: ErrMalformed},
		{name: "manifest", data: func() []byte { b := append([]byte(nil), data...); b[len(b)-1] ^= 1; return b }(), want: ErrIntegrity},
	} {
		t.Run(tc.name, func(t *testing.T) {
			im, err := NewImporter(context.Background(), bytes.NewReader(tc.data), Limits{}, acceptExact)
			if err != nil {
				t.Fatal(err)
			}
			sink := &memorySink{}
			if _, err := im.Next(context.Background(), sink); err != nil {
				t.Fatal(err)
			}
			if _, err := im.Next(context.Background(), sink); !errors.Is(err, tc.want) {
				t.Fatalf("footer = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestImporterResumeAtCheckpoint(t *testing.T) {
	t.Parallel()

	data := archiveBytes(t,
		Record{Kind: "x", Key: "1", Size: 1, Body: strings.NewReader("a")},
		Record{Kind: "x", Key: "2", Size: 1, Body: strings.NewReader("b")},
	)
	im, err := NewImporter(context.Background(), bytes.NewReader(data), Limits{}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	sink := &memorySink{}
	cp, err := im.Next(context.Background(), sink)
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := ResumeImporter(context.Background(), bytes.NewReader(data[cp.Offset():]), cp, Limits{}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resumed.Next(context.Background(), sink); err != nil {
		t.Fatal(err)
	}
	if _, err := resumed.Next(context.Background(), sink); !errors.Is(err, ErrComplete) {
		t.Fatalf("finish = %v", err)
	}
	if got := sink.committed[1]; got != "b" {
		t.Fatalf("resumed payload = %q", got)
	}
}

func TestResumeImporterRejectsArgumentsLimitsCheckpointAndCancellationWithoutIO(t *testing.T) {
	var archive bytes.Buffer
	ex, err := NewExporter(context.Background(), &archive, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	checkpoint := ex.Checkpoint()

	t.Run("nil context", func(t *testing.T) {
		input := &countingReader{r: bytes.NewReader(nil)}
		called := false
		_, err := ResumeImporter(nil, input, checkpoint, Limits{}, func(context.Context, Header) error {
			called = true
			return nil
		})
		if !errors.Is(err, ErrInvalid) || input.calls != 0 || called {
			t.Fatalf("resume = %v, reads=%d, compatibility=%v", err, input.calls, called)
		}
	})

	t.Run("nil reader", func(t *testing.T) {
		called := false
		_, err := ResumeImporter(context.Background(), nil, checkpoint, Limits{}, func(context.Context, Header) error {
			called = true
			return nil
		})
		if !errors.Is(err, ErrInvalid) || called {
			t.Fatalf("resume = %v, compatibility=%v", err, called)
		}
	})

	t.Run("nil compatibility", func(t *testing.T) {
		input := &countingReader{r: bytes.NewReader(nil)}
		_, err := ResumeImporter(context.Background(), input, checkpoint, Limits{}, nil)
		if !errors.Is(err, ErrInvalid) || input.calls != 0 {
			t.Fatalf("resume = %v, reads=%d", err, input.calls)
		}
	})

	t.Run("limits precede checkpoint context and compatibility", func(t *testing.T) {
		input := &countingReader{r: bytes.NewReader(nil)}
		badCheckpoint := checkpoint
		badCheckpoint.offset = 0
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		called := false
		_, err := ResumeImporter(ctx, input, badCheckpoint, Limits{MaxRecordBytes: 2, MaxTotalBytes: 1}, func(context.Context, Header) error {
			called = true
			return nil
		})
		if !errors.Is(err, ErrInvalid) || input.calls != 0 || called {
			t.Fatalf("resume = %v, reads=%d, compatibility=%v", err, input.calls, called)
		}
	})

	t.Run("checkpoint precedes context and compatibility", func(t *testing.T) {
		input := &countingReader{r: bytes.NewReader(nil)}
		badCheckpoint := checkpoint
		badCheckpoint.offset = 0
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		called := false
		_, err := ResumeImporter(ctx, input, badCheckpoint, Limits{}, func(context.Context, Header) error {
			called = true
			return nil
		})
		if !errors.Is(err, ErrInvalid) || input.calls != 0 || called {
			t.Fatalf("resume = %v, reads=%d, compatibility=%v", err, input.calls, called)
		}
	})

	t.Run("entry cancellation precedes compatibility", func(t *testing.T) {
		input := &countingReader{r: bytes.NewReader(nil)}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		called := false
		_, err := ResumeImporter(ctx, input, checkpoint, Limits{}, func(context.Context, Header) error {
			called = true
			return nil
		})
		if !errors.Is(err, ErrIO) || !causeIs(err, context.Canceled) || input.calls != 0 || called {
			t.Fatalf("resume = %v, reads=%d, compatibility=%v", err, input.calls, called)
		}
	})
}

func TestResumeImporterCompatibilityFailureCancellationAndRestoration(t *testing.T) {
	var archive bytes.Buffer
	ex, err := NewExporter(context.Background(), &archive, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	checkpoint := ex.Checkpoint()

	t.Run("incompatible is classified redacted and explicitly caused", func(t *testing.T) {
		input := &countingReader{r: bytes.NewReader(nil)}
		secret := errors.New("resume-schema-secret")
		_, err := ResumeImporter(context.Background(), input, checkpoint, Limits{}, func(_ context.Context, header Header) error {
			if header != testHeader() {
				t.Fatalf("compatibility header = %#v", header)
			}
			return secret
		})
		if !errors.Is(err, ErrIncompatible) || errors.Is(err, secret) || !causeIs(err, secret) || strings.Contains(err.Error(), "secret") || input.calls != 0 {
			t.Fatalf("resume = %v, reads=%d", err, input.calls)
		}
	})

	t.Run("cancellation after compatibility is IO failure", func(t *testing.T) {
		input := &countingReader{r: bytes.NewReader(nil)}
		ctx, cancel := context.WithCancel(context.Background())
		called := 0
		_, err := ResumeImporter(ctx, input, checkpoint, Limits{}, func(context.Context, Header) error {
			called++
			cancel()
			return nil
		})
		if !errors.Is(err, ErrIO) || !causeIs(err, context.Canceled) || input.calls != 0 || called != 1 {
			t.Fatalf("resume = %v, reads=%d, compatibility calls=%d", err, input.calls, called)
		}
	})

	t.Run("success restores exact committed state without reading", func(t *testing.T) {
		input := &countingReader{r: bytes.NewReader(nil)}
		called := 0
		resumed, err := ResumeImporter(context.Background(), input, checkpoint, Limits{}, func(_ context.Context, header Header) error {
			called++
			if header != checkpoint.Header() {
				t.Fatalf("compatibility header = %#v", header)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if input.calls != 0 || called != 1 || resumed.Header() != checkpoint.Header() || resumed.Checkpoint() != checkpoint || resumed.readOffset != checkpoint.Offset() {
			t.Fatalf("reads=%d compatibility=%d importer=%#v", input.calls, called, resumed)
		}
	})
}

func TestImporterEmptyArchive(t *testing.T) {
	t.Parallel()

	im, err := NewImporter(context.Background(), bytes.NewReader(archiveBytes(t)), Limits{}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := im.Next(context.Background(), &memorySink{}); !errors.Is(err, ErrComplete) {
		t.Fatalf("next = %v", err)
	}
	if _, err := im.Next(context.Background(), &memorySink{}); !errors.Is(err, ErrComplete) {
		t.Fatalf("repeat next = %v", err)
	}
}

type unknownOutcomeSink struct {
	staged    bytes.Buffer
	committed map[uint64]string
	sequence  uint64
	failOnce  bool
	aborts    int
}

func (s *unknownOutcomeSink) Begin(_ context.Context, _ Header, meta RecordMeta) (RecordSink, error) {
	s.sequence = meta.Sequence
	s.staged.Reset()
	return unknownOutcomeRecord{s}, nil
}

type unknownOutcomeRecord struct{ owner *unknownOutcomeSink }

func (r unknownOutcomeRecord) Write(p []byte) (int, error) { return r.owner.staged.Write(p) }
func (r unknownOutcomeRecord) Abort(context.Context) error {
	r.owner.aborts++
	return nil
}
func (r unknownOutcomeRecord) Commit(context.Context) error {
	if r.owner.committed == nil {
		r.owner.committed = make(map[uint64]string)
	}
	value := r.owner.staged.String()
	if prior, ok := r.owner.committed[r.owner.sequence]; ok {
		if prior != value {
			return errors.New("idempotency conflict")
		}
		return nil
	}
	r.owner.committed[r.owner.sequence] = value
	if r.owner.failOnce {
		r.owner.failOnce = false
		return errors.New("unknown outcome")
	}
	return nil
}

func TestImporterInitialCheckpointRecoversUnknownCommitOutcome(t *testing.T) {
	t.Parallel()

	data := archiveBytes(t, Record{Kind: "x", Size: 3, Body: strings.NewReader("abc")})
	im, err := NewImporter(context.Background(), bytes.NewReader(data), Limits{}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	initial := im.Checkpoint()
	sink := &unknownOutcomeSink{failOnce: true}
	if _, err := im.Next(context.Background(), sink); !errors.Is(err, ErrSink) {
		t.Fatalf("commit outcome = %v", err)
	}
	if sink.committed[0] != "abc" || sink.aborts != 0 {
		t.Fatalf("unknown outcome state = %#v, aborts=%d", sink.committed, sink.aborts)
	}
	if im.Checkpoint() != initial {
		t.Fatal("unknown outcome advanced importer checkpoint")
	}
	resumed, err := ResumeImporter(context.Background(), bytes.NewReader(data[initial.Offset():]), initial, Limits{}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resumed.Next(context.Background(), sink); err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	if _, err := resumed.Next(context.Background(), sink); !errors.Is(err, ErrComplete) {
		t.Fatalf("complete replay: %v", err)
	}
	if sink.committed[0] != "abc" || sink.aborts != 0 {
		t.Fatalf("replayed state = %#v, aborts=%d", sink.committed, sink.aborts)
	}
}

var _ io.Writer = (*memoryRecordSink)(nil)
