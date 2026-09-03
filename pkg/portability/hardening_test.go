package portability

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type countingWriter struct {
	bytes.Buffer
	calls int
}

func (w *countingWriter) Write(p []byte) (int, error) {
	w.calls++
	return w.Buffer.Write(p)
}

type cancelingWriter struct {
	countingWriter
	cancel   context.CancelFunc
	cancelAt int
}

func (w *cancelingWriter) Write(p []byte) (int, error) {
	n, err := w.countingWriter.Write(p)
	if w.calls == w.cancelAt {
		w.cancel()
	}
	return n, err
}

type countingReader struct {
	r     io.Reader
	calls int
}

func (r *countingReader) Read(p []byte) (int, error) {
	r.calls++
	return r.r.Read(p)
}

type cancelingReader struct {
	countingReader
	cancel   context.CancelFunc
	cancelAt int
}

func (r *cancelingReader) Read(p []byte) (int, error) {
	n, err := r.countingReader.Read(p)
	if r.calls == r.cancelAt {
		r.cancel()
	}
	return n, err
}

type zeroThenReader struct {
	empty int
	r     io.Reader
}

func (r *zeroThenReader) Read(p []byte) (int, error) {
	if r.empty > 0 {
		r.empty--
		return 0, nil
	}
	return r.r.Read(p)
}

type negativeCountReader struct{}

func (negativeCountReader) Read([]byte) (int, error) { return -1, nil }

type dataAndErrorReader struct {
	data []byte
	err  error
	done bool
}

func (r *dataAndErrorReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	return copy(p, r.data), r.err
}

type eofWithDataReader struct{ r io.Reader }

func (r eofWithDataReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 {
		return n, io.EOF
	}
	return n, err
}

func TestPreCanceledConstructionDoesNoIO(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out := &countingWriter{}
	if _, err := NewExporter(ctx, out, testHeader(), Limits{}); !errors.Is(err, ErrIO) || !causeIs(err, context.Canceled) {
		t.Fatalf("export = %v", err)
	}
	if out.calls != 0 || out.Len() != 0 {
		t.Fatalf("pre-canceled export wrote calls=%d bytes=%d", out.calls, out.Len())
	}

	in := &countingReader{r: bytes.NewReader(archiveBytes(t))}
	if _, err := NewImporter(ctx, in, Limits{}, acceptExact); !errors.Is(err, ErrIO) || !causeIs(err, context.Canceled) {
		t.Fatalf("import = %v", err)
	}
	if in.calls != 0 {
		t.Fatalf("pre-canceled import read %d times", in.calls)
	}
}

func TestExporterCancellationSeamsKeepPriorCheckpoint(t *testing.T) {
	t.Run("header write", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		w := &cancelingWriter{cancel: cancel, cancelAt: 1}
		_, err := NewExporter(ctx, w, testHeader(), Limits{})
		if !errors.Is(err, ErrIO) || !causeIs(err, context.Canceled) {
			t.Fatalf("error = %v", err)
		}
	})

	for _, tc := range []struct {
		name     string
		cancelAt int
	}{
		{name: "metadata write", cancelAt: 2},
		{name: "payload write", cancelAt: 3},
		{name: "digest write", cancelAt: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			w := &cancelingWriter{cancel: cancel, cancelAt: tc.cancelAt}
			ex, err := NewExporter(ctx, w, testHeader(), Limits{})
			if err != nil {
				t.Fatal(err)
			}
			prior := ex.Checkpoint()
			_, err = ex.WriteRecord(ctx, Record{Kind: "x", Size: 1, Body: strings.NewReader("a")})
			if !errors.Is(err, ErrIO) || !causeIs(err, context.Canceled) || ex.Checkpoint() != prior {
				t.Fatalf("error=%v checkpoint=%#v prior=%#v", err, ex.Checkpoint(), prior)
			}
		})
	}

	t.Run("payload read", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		var out bytes.Buffer
		ex, err := NewExporter(ctx, &out, testHeader(), Limits{})
		if err != nil {
			t.Fatal(err)
		}
		prior := ex.Checkpoint()
		body := &cancelingReader{countingReader: countingReader{r: strings.NewReader("a")}, cancel: cancel, cancelAt: 1}
		_, err = ex.WriteRecord(ctx, Record{Kind: "x", Size: 1, Body: body})
		if !errors.Is(err, ErrIO) || !causeIs(err, context.Canceled) || ex.Checkpoint() != prior {
			t.Fatalf("error=%v checkpoint=%#v", err, ex.Checkpoint())
		}
	})

	t.Run("zero-payload EOF probe", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		var out bytes.Buffer
		ex, err := NewExporter(ctx, &out, testHeader(), Limits{})
		if err != nil {
			t.Fatal(err)
		}
		prior := ex.Checkpoint()
		body := &cancelingReader{countingReader: countingReader{r: strings.NewReader("")}, cancel: cancel, cancelAt: 1}
		_, err = ex.WriteRecord(ctx, Record{Kind: "x", Body: body})
		if !errors.Is(err, ErrIO) || !causeIs(err, context.Canceled) || ex.Checkpoint() != prior {
			t.Fatalf("error=%v checkpoint=%#v", err, ex.Checkpoint())
		}
	})

	t.Run("footer write", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		w := &cancelingWriter{cancel: cancel, cancelAt: 2}
		ex, err := NewExporter(ctx, w, testHeader(), Limits{})
		if err != nil {
			t.Fatal(err)
		}
		prior := ex.Checkpoint()
		_, err = ex.Finalize(ctx)
		if !errors.Is(err, ErrIO) || !causeIs(err, context.Canceled) || ex.Checkpoint() != prior {
			t.Fatalf("error=%v checkpoint=%#v", err, ex.Checkpoint())
		}
	})

	t.Run("pre-canceled finalize is retryable", func(t *testing.T) {
		var out countingWriter
		ex, err := NewExporter(context.Background(), &out, testHeader(), Limits{})
		if err != nil {
			t.Fatal(err)
		}
		calls := out.calls
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := ex.Finalize(ctx); !errors.Is(err, ErrIO) || !causeIs(err, context.Canceled) {
			t.Fatalf("canceled finalize = %v", err)
		}
		if out.calls != calls {
			t.Fatalf("canceled finalize wrote %d calls", out.calls-calls)
		}
		if _, err := ex.Finalize(context.Background()); err != nil {
			t.Fatalf("retry finalize = %v", err)
		}
	})
}

type cancelingStage struct {
	cancel        context.CancelFunc
	cancelCommit  bool
	aborts        int
	commits       int
	abortDeadline time.Time
}

type abortBehaviorStage struct {
	abortFn     func(context.Context) error
	aborts      int
	entryErr    error
	deadlineSet bool
}

func (*abortBehaviorStage) Write(p []byte) (int, error)  { return len(p), nil }
func (*abortBehaviorStage) Commit(context.Context) error { return nil }
func (s *abortBehaviorStage) Abort(ctx context.Context) error {
	s.aborts++
	s.entryErr = ctx.Err()
	_, s.deadlineSet = ctx.Deadline()
	if s.abortFn != nil {
		return s.abortFn(ctx)
	}
	return nil
}

func TestBeginStageAndErrorRunsBoundedAbort(t *testing.T) {
	data := archiveBytes(t, Record{Kind: "x", Size: 1, Body: strings.NewReader("a")})
	beginErr := errors.New("begin-sensitive")
	abortErr := errors.New("abort-sensitive")

	for _, tc := range []struct {
		name           string
		returnStage    bool
		cancelBegin    bool
		abortFn        func(context.Context) error
		wantAbortCause error
		wantClasses    []error
	}{
		{name: "no stage", wantClasses: []error{ErrSink}},
		{name: "cleanup success", returnStage: true, wantClasses: []error{ErrSink}},
		{name: "cleanup failure", returnStage: true, abortFn: func(context.Context) error { return abortErr }, wantAbortCause: abortErr, wantClasses: []error{ErrSink}},
		{name: "canceled no stage", cancelBegin: true, wantClasses: []error{ErrSink, ErrIO}},
		{name: "canceled with stage", returnStage: true, cancelBegin: true, wantClasses: []error{ErrSink, ErrIO}},
		{name: "canceled with stage and cleanup failure", returnStage: true, cancelBegin: true, abortFn: func(context.Context) error { return abortErr }, wantAbortCause: abortErr, wantClasses: []error{ErrSink, ErrIO}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			im, err := NewImporter(ctx, bytes.NewReader(data), Limits{}, acceptExact)
			if err != nil {
				t.Fatal(err)
			}
			stage := &abortBehaviorStage{abortFn: tc.abortFn}
			_, err = im.Next(ctx, SinkFunc(func(context.Context, Header, RecordMeta) (RecordSink, error) {
				if tc.cancelBegin {
					cancel()
				}
				if tc.returnStage {
					return stage, beginErr
				}
				return nil, beginErr
			}))
			assertClassSet(t, err, tc.wantClasses...)
			wantAborts := 0
			if tc.returnStage {
				wantAborts = 1
			}
			if stage.aborts != wantAborts || (tc.returnStage && (stage.entryErr != nil || !stage.deadlineSet)) {
				t.Fatalf("aborts=%d entryErr=%v deadline=%v", stage.aborts, stage.entryErr, stage.deadlineSet)
			}
			if !containsExplicitCause(err, beginErr) {
				t.Fatalf("primary begin cause lost: %v", err)
			}
			if tc.cancelBegin && !containsExplicitCause(err, context.Canceled) {
				t.Fatalf("cancellation cause lost: %v", err)
			}
			if tc.wantAbortCause != nil && !containsExplicitCause(err, tc.wantAbortCause) {
				t.Fatalf("cleanup cause %v missing: %v", tc.wantAbortCause, err)
			}
			if strings.Contains(err.Error(), beginErr.Error()) || strings.Contains(err.Error(), abortErr.Error()) {
				t.Fatalf("public error leaked callback text: %q", err)
			}
		})
	}
}

func TestAbortReportsCleanupDeadline(t *testing.T) {
	beginErr := errors.New("begin-sensitive")
	primary := wrap(ErrSink, "begin record", beginErr)
	for _, tc := range []struct {
		name    string
		abortFn func(context.Context) error
	}{
		{name: "nil after deadline", abortFn: func(ctx context.Context) error { <-ctx.Done(); return nil }},
		{name: "context error after deadline", abortFn: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stage := &abortBehaviorStage{abortFn: tc.abortFn}
			started := time.Now()
			err := abortWithTimeout(context.Background(), stage, primary, 10*time.Millisecond)
			if elapsed := time.Since(started); elapsed > time.Second {
				t.Fatalf("bounded cleanup took %s", elapsed)
			}
			assertClassSet(t, err, ErrSink)
			if stage.aborts != 1 || !containsExplicitCause(err, beginErr) || !containsExplicitCause(err, context.DeadlineExceeded) {
				t.Fatalf("error=%v aborts=%d", err, stage.aborts)
			}
		})
	}
}

func (s *cancelingStage) Write(p []byte) (int, error) { return len(p), nil }
func (s *cancelingStage) Commit(context.Context) error {
	s.commits++
	if s.cancelCommit {
		s.cancel()
	}
	return nil
}
func (s *cancelingStage) Abort(ctx context.Context) error {
	s.aborts++
	s.abortDeadline, _ = ctx.Deadline()
	return nil
}

func TestImporterCancellationSeamsAndBoundedAbort(t *testing.T) {
	recordArchive := archiveBytes(t, Record{Kind: "x", Size: 1, Body: strings.NewReader("a")})

	t.Run("compatibility", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		_, err := NewImporter(ctx, bytes.NewReader(recordArchive), Limits{}, func(context.Context, Header) error {
			cancel()
			return nil
		})
		if !errors.Is(err, ErrIO) || !causeIs(err, context.Canceled) {
			t.Fatalf("error=%v", err)
		}
	})

	t.Run("metadata read", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		r := &cancelingReader{countingReader: countingReader{r: bytes.NewReader(recordArchive)}, cancel: cancel, cancelAt: 9}
		im, err := NewImporter(ctx, r, Limits{}, acceptExact)
		if err != nil {
			t.Fatal(err)
		}
		begins := 0
		sink := SinkFunc(func(context.Context, Header, RecordMeta) (RecordSink, error) {
			begins++
			return &cancelingStage{}, nil
		})
		_, err = im.Next(ctx, sink)
		if !errors.Is(err, ErrIO) || !causeIs(err, context.Canceled) || begins != 0 {
			t.Fatalf("error=%v begins=%d", err, begins)
		}
	})

	t.Run("begin", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		im, err := NewImporter(ctx, bytes.NewReader(recordArchive), Limits{}, acceptExact)
		if err != nil {
			t.Fatal(err)
		}
		stage := &cancelingStage{}
		sink := SinkFunc(func(context.Context, Header, RecordMeta) (RecordSink, error) {
			cancel()
			return stage, nil
		})
		started := time.Now()
		_, err = im.Next(ctx, sink)
		if !errors.Is(err, ErrIO) || !causeIs(err, context.Canceled) || stage.aborts != 1 {
			t.Fatalf("error=%v aborts=%d", err, stage.aborts)
		}
		if stage.abortDeadline.IsZero() || stage.abortDeadline.Before(started) || stage.abortDeadline.After(started.Add(AbortTimeout+time.Second)) {
			t.Fatalf("abort deadline = %v", stage.abortDeadline)
		}
	})

	for _, tc := range []struct {
		name     string
		cancelAt int
	}{
		{name: "payload read", cancelAt: 14},
		{name: "digest read", cancelAt: 15},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			r := &cancelingReader{countingReader: countingReader{r: bytes.NewReader(recordArchive)}, cancel: cancel, cancelAt: tc.cancelAt}
			im, err := NewImporter(ctx, r, Limits{}, acceptExact)
			if err != nil {
				t.Fatal(err)
			}
			stage := &cancelingStage{}
			_, err = im.Next(ctx, SinkFunc(func(context.Context, Header, RecordMeta) (RecordSink, error) { return stage, nil }))
			if !errors.Is(err, ErrIO) || !causeIs(err, context.Canceled) || stage.aborts != 1 || stage.commits != 0 {
				t.Fatalf("error=%v reads=%d aborts=%d commits=%d", err, r.calls, stage.aborts, stage.commits)
			}
		})
	}

	t.Run("commit unknown outcome", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		im, err := NewImporter(ctx, bytes.NewReader(recordArchive), Limits{}, acceptExact)
		if err != nil {
			t.Fatal(err)
		}
		prior := im.Checkpoint()
		stage := &cancelingStage{cancel: cancel, cancelCommit: true}
		_, err = im.Next(ctx, SinkFunc(func(context.Context, Header, RecordMeta) (RecordSink, error) { return stage, nil }))
		if !errors.Is(err, ErrIO) || !causeIs(err, context.Canceled) || stage.commits != 1 || stage.aborts != 0 || im.Checkpoint() != prior {
			t.Fatalf("error=%v commits=%d aborts=%d checkpoint=%#v", err, stage.commits, stage.aborts, im.Checkpoint())
		}
	})

	t.Run("footer EOF probe", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		r := &cancelingReader{countingReader: countingReader{r: bytes.NewReader(archiveBytes(t))}, cancel: cancel, cancelAt: 12}
		im, err := NewImporter(ctx, r, Limits{}, acceptExact)
		if err != nil {
			t.Fatal(err)
		}
		_, err = im.Next(ctx, &memorySink{})
		if !errors.Is(err, ErrIO) || !causeIs(err, context.Canceled) {
			t.Fatalf("error=%v reads=%d", err, r.calls)
		}
		if _, manifestErr := im.Manifest(); !errors.Is(manifestErr, ErrIncomplete) {
			t.Fatalf("manifest = %v", manifestErr)
		}
	})
}

func TestStrictIOContracts(t *testing.T) {
	t.Run("positive short nil writer is not retried", func(t *testing.T) {
		w := &positiveShortWriter{}
		if _, err := NewExporter(context.Background(), w, testHeader(), Limits{}); !errors.Is(err, ErrIO) {
			t.Fatalf("error = %v", err)
		}
		if w.calls != 1 {
			t.Fatalf("write calls = %d", w.calls)
		}
	})

	t.Run("transient empty reads", func(t *testing.T) {
		var out bytes.Buffer
		ex, err := NewExporter(context.Background(), &out, testHeader(), Limits{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 1, Body: &zeroThenReader{empty: MaxConsecutiveEmptyReads, r: strings.NewReader("a")}}); err != nil {
			t.Fatal(err)
		}

		data := archiveBytes(t)
		if _, err := NewImporter(context.Background(), &zeroThenReader{empty: MaxConsecutiveEmptyReads, r: bytes.NewReader(data)}, Limits{}, acceptExact); err != nil {
			t.Fatal(err)
		}
	})

	for _, tc := range []struct {
		name   string
		reader io.Reader
	}{
		{name: "negative", reader: negativeCountReader{}},
		{name: "oversized", reader: oversizedCountReader{}},
	} {
		t.Run(tc.name+" export", func(t *testing.T) {
			var out bytes.Buffer
			ex, err := NewExporter(context.Background(), &out, testHeader(), Limits{})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 1, Body: tc.reader}); !errors.Is(err, ErrIO) {
				t.Fatalf("error = %v", err)
			}
		})
		t.Run(tc.name+" import", func(t *testing.T) {
			if _, err := NewImporter(context.Background(), tc.reader, Limits{}, acceptExact); !errors.Is(err, ErrIO) {
				t.Fatalf("error = %v", err)
			}
		})
	}

	t.Run("data plus EOF", func(t *testing.T) {
		var out bytes.Buffer
		ex, err := NewExporter(context.Background(), &out, testHeader(), Limits{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 1, Body: &dataAndErrorReader{data: []byte("a"), err: io.EOF}}); err != nil {
			t.Fatal(err)
		}
		data := archiveBytes(t)
		if _, err := NewImporter(context.Background(), eofWithDataReader{r: bytes.NewReader(data)}, Limits{}, acceptExact); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("data plus non-EOF", func(t *testing.T) {
		var out bytes.Buffer
		ex, err := NewExporter(context.Background(), &out, testHeader(), Limits{})
		if err != nil {
			t.Fatal(err)
		}
		prior := ex.Checkpoint()
		_, err = ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 1, Body: &dataAndErrorReader{data: []byte("a"), err: errors.New("reader-secret")}})
		if !errors.Is(err, ErrIO) || ex.Checkpoint() != prior {
			t.Fatalf("error=%v checkpoint=%#v", err, ex.Checkpoint())
		}
	})

	t.Run("no progress bound", func(t *testing.T) {
		r := &countingReader{r: noProgressReader{}}
		var one [1]byte
		if _, err := readExactContext(context.Background(), r, one[:]); !errors.Is(err, io.ErrNoProgress) {
			t.Fatalf("error = %v", err)
		}
		if r.calls != MaxConsecutiveEmptyReads+1 {
			t.Fatalf("calls = %d", r.calls)
		}
	})
}

type positiveShortWriter struct{ calls int }

func (w *positiveShortWriter) Write(p []byte) (int, error) {
	w.calls++
	if len(p) == 0 {
		return 0, nil
	}
	return len(p) - 1, nil
}

func TestPayloadBuffersAreLazyAndReused(t *testing.T) {
	var out bytes.Buffer
	ex, err := NewExporter(context.Background(), &out, testHeader(), Limits{MaxRecords: 3})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Body: strings.NewReader("")}); err != nil {
		t.Fatal(err)
	}
	if ex.buffer != nil {
		t.Fatal("empty export record allocated copy buffer")
	}
	if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 1, Body: strings.NewReader("a")}); err != nil {
		t.Fatal(err)
	}
	firstExportBuffer := &ex.buffer[0]
	if _, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 1, Body: strings.NewReader("b")}); err != nil {
		t.Fatal(err)
	}
	if firstExportBuffer != &ex.buffer[0] {
		t.Fatal("export buffer was not reused")
	}

	im, err := NewImporter(context.Background(), bytes.NewReader(archiveBytes(t,
		Record{Kind: "x", Body: strings.NewReader("")},
		Record{Kind: "x", Size: 1, Body: strings.NewReader("a")},
		Record{Kind: "x", Size: 1, Body: strings.NewReader("b")},
	)), Limits{}, acceptExact)
	if err != nil {
		t.Fatal(err)
	}
	sink := &memorySink{}
	if _, err := im.Next(context.Background(), sink); err != nil {
		t.Fatal(err)
	}
	if im.buffer != nil {
		t.Fatal("empty import record allocated copy buffer")
	}
	if _, err := im.Next(context.Background(), sink); err != nil {
		t.Fatal(err)
	}
	firstImportBuffer := &im.buffer[0]
	if _, err := im.Next(context.Background(), sink); err != nil {
		t.Fatal(err)
	}
	if firstImportBuffer != &im.buffer[0] {
		t.Fatal("import buffer was not reused")
	}
}

type injectedErrorWriter struct{ err error }

func (w injectedErrorWriter) Write([]byte) (int, error) { return 0, w.err }

type injectedErrorReader struct{ err error }

func (r injectedErrorReader) Read([]byte) (int, error) { return 0, r.err }

type injectedStage struct {
	writeErr  error
	commitErr error
	abortErr  error
}

func (s *injectedStage) Write(p []byte) (int, error) {
	if s.writeErr != nil {
		return 0, s.writeErr
	}
	return len(p), nil
}
func (s *injectedStage) Commit(context.Context) error { return s.commitErr }
func (s *injectedStage) Abort(context.Context) error  { return s.abortErr }

func librarySentinels() []error {
	return []error{
		ErrInvalid, ErrMalformed, ErrTruncated, ErrIO, ErrIncompatible, ErrLimit,
		ErrIntegrity, ErrSequence, ErrSink, ErrIncomplete, ErrFinalized, ErrComplete,
	}
}

func assertClassSet(t *testing.T, err error, expected ...error) {
	t.Helper()
	want := make(map[error]bool, len(expected))
	for _, class := range expected {
		want[class] = true
	}
	for _, class := range librarySentinels() {
		if got := errors.Is(err, class); got != want[class] {
			t.Fatalf("errors.Is(%v, %v)=%v, want %v", err, class, got, want[class])
		}
	}
}

func containsExplicitCause(err, target error) bool {
	if err == nil {
		return false
	}
	if causer, ok := err.(Causer); ok && errors.Is(causer.Cause(), target) {
		return true
	}
	switch current := err.(type) {
	case interface{ Unwrap() []error }:
		for _, child := range current.Unwrap() {
			if containsExplicitCause(child, target) {
				return true
			}
		}
	case interface{ Unwrap() error }:
		return containsExplicitCause(current.Unwrap(), target)
	}
	return false
}

func TestCallbackCausesCannotSpoofLibrarySentinels(t *testing.T) {
	data := archiveBytes(t, Record{Kind: "x", Size: 1, Body: strings.NewReader("a")})
	headerLen := len(encodeHeader(testHeader()))
	prefixLen := len(encodeRecordPrefix(RecordMeta{Sequence: 0, Kind: "x", Size: 1}))
	corrupt := append([]byte(nil), data...)
	corrupt[headerLen+prefixLen] ^= 1

	for _, cause := range librarySentinels() {
		cause := cause
		t.Run(cause.Error(), func(t *testing.T) {
			t.Run("archive writer", func(t *testing.T) {
				_, err := NewExporter(context.Background(), injectedErrorWriter{err: cause}, testHeader(), Limits{})
				assertClassSet(t, err, ErrIO)
				if !containsExplicitCause(err, cause) {
					t.Fatalf("explicit cause missing: %v", err)
				}
			})

			t.Run("payload reader", func(t *testing.T) {
				var out bytes.Buffer
				ex, err := NewExporter(context.Background(), &out, testHeader(), Limits{})
				if err != nil {
					t.Fatal(err)
				}
				_, err = ex.WriteRecord(context.Background(), Record{Kind: "x", Size: 1, Body: injectedErrorReader{err: cause}})
				assertClassSet(t, err, ErrIO)
				if !containsExplicitCause(err, cause) {
					t.Fatalf("explicit cause missing: %v", err)
				}
			})

			t.Run("compatibility", func(t *testing.T) {
				_, err := NewImporter(context.Background(), bytes.NewReader(data), Limits{}, func(context.Context, Header) error { return cause })
				assertClassSet(t, err, ErrIncompatible)
				if !containsExplicitCause(err, cause) {
					t.Fatalf("explicit cause missing: %v", err)
				}
			})

			t.Run("begin", func(t *testing.T) {
				im, err := NewImporter(context.Background(), bytes.NewReader(data), Limits{}, acceptExact)
				if err != nil {
					t.Fatal(err)
				}
				_, err = im.Next(context.Background(), SinkFunc(func(context.Context, Header, RecordMeta) (RecordSink, error) { return nil, cause }))
				assertClassSet(t, err, ErrSink)
				if !containsExplicitCause(err, cause) {
					t.Fatalf("explicit cause missing: %v", err)
				}
			})

			for _, callback := range []struct {
				name  string
				stage *injectedStage
			}{
				{name: "write", stage: &injectedStage{writeErr: cause}},
				{name: "commit", stage: &injectedStage{commitErr: cause}},
			} {
				t.Run(callback.name, func(t *testing.T) {
					im, err := NewImporter(context.Background(), bytes.NewReader(data), Limits{}, acceptExact)
					if err != nil {
						t.Fatal(err)
					}
					_, err = im.Next(context.Background(), SinkFunc(func(context.Context, Header, RecordMeta) (RecordSink, error) { return callback.stage, nil }))
					assertClassSet(t, err, ErrSink)
					if !containsExplicitCause(err, cause) {
						t.Fatalf("explicit cause missing: %v", err)
					}
				})
			}

			t.Run("abort", func(t *testing.T) {
				im, err := NewImporter(context.Background(), bytes.NewReader(corrupt), Limits{}, acceptExact)
				if err != nil {
					t.Fatal(err)
				}
				stage := &injectedStage{abortErr: cause}
				_, err = im.Next(context.Background(), SinkFunc(func(context.Context, Header, RecordMeta) (RecordSink, error) { return stage, nil }))
				assertClassSet(t, err, ErrIntegrity, ErrSink)
				if !containsExplicitCause(err, cause) {
					t.Fatalf("explicit cause missing: %v", err)
				}
			})
		})
	}
}
