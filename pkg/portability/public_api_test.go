package portability_test

import (
	"bytes"
	"context"
	"errors"
	"io"
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

var _ portability.Sink = discardSink{}
var _ portability.RecordSink = discardRecord{}
