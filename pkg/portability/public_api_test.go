package portability_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/gotthboard/gotth-portability/pkg/portability"
)

type discardSink struct{}

func (discardSink) Begin(context.Context, portability.Header, portability.RecordMeta) (portability.RecordSink, error) {
	return discardRecord{}, nil
}

type discardRecord struct{ io.Writer }

func (discardRecord) Write(p []byte) (int, error)  { return len(p), nil }
func (discardRecord) Commit(context.Context) error { return nil }
func (discardRecord) Abort(context.Context) error  { return nil }

func TestExternalPackageCanUseCompleteAPI(t *testing.T) {
	header := portability.Header{WireVersion: portability.WireVersion, ArchiveID: "external", Schema: "consumer", SchemaVersion: 1}
	var archive bytes.Buffer
	exporter, err := portability.NewExporter(context.Background(), &archive, header, portability.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err := exporter.WriteRecord(context.Background(), portability.Record{Kind: "opaque", Body: bytes.NewReader(nil)})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := checkpoint.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := portability.ParseCheckpoint(encoded, portability.Limits{}); err != nil {
		t.Fatal(err)
	}
	manifest, err := exporter.Finalize(context.Background())
	if err != nil || manifest.Records != 1 {
		t.Fatalf("finalize: %#v, %v", manifest, err)
	}
	importer, err := portability.NewImporter(context.Background(), bytes.NewReader(archive.Bytes()), portability.Limits{}, func(_ context.Context, got portability.Header) error {
		if got.Schema != header.Schema {
			return errors.New("wrong schema")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := importer.Next(context.Background(), discardSink{}); err != nil {
		t.Fatal(err)
	}
	if _, err := importer.Next(context.Background(), discardSink{}); !errors.Is(err, portability.ErrComplete) {
		t.Fatalf("completion = %v", err)
	}
	if _, err := importer.Manifest(); err != nil {
		t.Fatal(err)
	}
}

type outcomeRecord struct {
	write  func([]byte) (int, error)
	commit func(context.Context) error
	abort  func(context.Context) error
}

func (r *outcomeRecord) Write(p []byte) (int, error) {
	if r.write != nil {
		return r.write(p)
	}
	return len(p), nil
}

func (r *outcomeRecord) Commit(ctx context.Context) error {
	if r.commit != nil {
		return r.commit(ctx)
	}
	return nil
}

func (r *outcomeRecord) Abort(ctx context.Context) error {
	if r.abort != nil {
		return r.abort(ctx)
	}
	return nil
}

func combinedOutcomeArchive(t *testing.T) ([]byte, portability.Checkpoint) {
	t.Helper()
	header := portability.Header{WireVersion: portability.WireVersion, ArchiveID: "combined", Schema: "consumer", SchemaVersion: 1}
	var archive bytes.Buffer
	exporter, err := portability.NewExporter(context.Background(), &archive, header, portability.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	initial := exporter.Checkpoint()
	if _, err := exporter.WriteRecord(context.Background(), portability.Record{Kind: "opaque", Size: 1, Body: strings.NewReader("x")}); err != nil {
		t.Fatal(err)
	}
	if _, err := exporter.Finalize(context.Background()); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes(), initial
}

func assertPublicCombinedError(t *testing.T, err error, classes []error, causes ...error) {
	t.Helper()
	if err == nil {
		t.Fatal("combined operation returned nil error")
	}
	for _, class := range classes {
		if !errors.Is(err, class) {
			t.Fatalf("errors.Is(%v, %v) = false", err, class)
		}
	}
	var causer portability.Causer
	if !errors.As(err, &causer) {
		t.Fatalf("errors.As(%v, *portability.Causer) = false", err)
	}
	aggregate := causer.Cause()
	if _, ok := aggregate.(interface{ Unwrap() []error }); !ok {
		t.Fatalf("Cause() type = %T, want standard multi-error", aggregate)
	}
	for _, cause := range causes {
		if !errors.Is(aggregate, cause) {
			t.Fatalf("Cause() = %v, missing %v", aggregate, cause)
		}
		if errors.Is(err, cause) {
			t.Fatalf("raw cause %v entered errors.Is traversal", cause)
		}
		if strings.Contains(err.Error(), cause.Error()) {
			t.Fatalf("error text leaked raw cause %q: %q", cause, err)
		}
	}
}

func TestCombinedErrorsExposeAllCausesThroughPublicCauser(t *testing.T) {
	archive, checkpoint := combinedOutcomeArchive(t)

	for _, tc := range []struct {
		name   string
		resume bool
	}{
		{name: "NewImporter"},
		{name: "ResumeImporter", resume: true},
	} {
		t.Run("compatibility "+tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			compatibilityErr := errors.New("compatibility-raw")
			compatibility := func(context.Context, portability.Header) error {
				cancel()
				return compatibilityErr
			}
			var err error
			if tc.resume {
				_, err = portability.ResumeImporter(ctx, bytes.NewReader(nil), checkpoint, portability.Limits{}, compatibility)
			} else {
				_, err = portability.NewImporter(ctx, bytes.NewReader(archive), portability.Limits{}, compatibility)
			}
			assertPublicCombinedError(t, err, []error{portability.ErrIncompatible, portability.ErrIO}, compatibilityErr, context.Canceled)
		})
	}

	t.Run("Begin", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		importer, err := portability.NewImporter(ctx, bytes.NewReader(archive), portability.Limits{}, func(context.Context, portability.Header) error { return nil })
		if err != nil {
			t.Fatal(err)
		}
		_, err = importer.Next(ctx, portability.SinkFunc(func(context.Context, portability.Header, portability.RecordMeta) (portability.RecordSink, error) {
			cancel()
			return nil, nil
		}))
		assertPublicCombinedError(t, err, []error{portability.ErrSink, portability.ErrIO}, context.Canceled)
	})

	t.Run("staged Write", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		writeErr := errors.New("write-raw")
		stage := &outcomeRecord{write: func([]byte) (int, error) {
			cancel()
			return 0, writeErr
		}}
		importer, err := portability.NewImporter(ctx, bytes.NewReader(archive), portability.Limits{}, func(context.Context, portability.Header) error { return nil })
		if err != nil {
			t.Fatal(err)
		}
		_, err = importer.Next(ctx, portability.SinkFunc(func(context.Context, portability.Header, portability.RecordMeta) (portability.RecordSink, error) {
			return stage, nil
		}))
		assertPublicCombinedError(t, err, []error{portability.ErrSink, portability.ErrIO}, writeErr, context.Canceled)
	})

	t.Run("Commit", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		commitErr := errors.New("commit-raw")
		stage := &outcomeRecord{commit: func(context.Context) error {
			cancel()
			return commitErr
		}}
		importer, err := portability.NewImporter(ctx, bytes.NewReader(archive), portability.Limits{}, func(context.Context, portability.Header) error { return nil })
		if err != nil {
			t.Fatal(err)
		}
		_, err = importer.Next(ctx, portability.SinkFunc(func(context.Context, portability.Header, portability.RecordMeta) (portability.RecordSink, error) {
			return stage, nil
		}))
		assertPublicCombinedError(t, err, []error{portability.ErrSink, portability.ErrIO}, commitErr, context.Canceled)
	})

	t.Run("Abort", func(t *testing.T) {
		writeErr := errors.New("write-before-abort-raw")
		abortErr := errors.New("abort-raw")
		stage := &outcomeRecord{
			write: func([]byte) (int, error) { return 0, writeErr },
			abort: func(context.Context) error { return abortErr },
		}
		importer, err := portability.NewImporter(context.Background(), bytes.NewReader(archive), portability.Limits{}, func(context.Context, portability.Header) error { return nil })
		if err != nil {
			t.Fatal(err)
		}
		_, err = importer.Next(context.Background(), portability.SinkFunc(func(context.Context, portability.Header, portability.RecordMeta) (portability.RecordSink, error) {
			return stage, nil
		}))
		assertPublicCombinedError(t, err, []error{portability.ErrSink}, writeErr, abortErr)
	})
}

var _ portability.Sink = discardSink{}
var _ portability.RecordSink = discardRecord{}
var _ portability.RecordSink = (*outcomeRecord)(nil)
