package portability

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCheckpointBinaryRoundTrip(t *testing.T) {
	t.Parallel()

	var archive bytes.Buffer
	ex, err := NewExporter(&archive, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	cp, err := ex.WriteRecord(context.Background(), Record{Kind: "x", Key: "k", Size: 4, Body: strings.NewReader("data")})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := cp.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := ParseCheckpoint(encoded, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if decoded != cp {
		t.Fatalf("decoded checkpoint differs:\n got %#v\nwant %#v", decoded, cp)
	}
}

func TestCheckpointRejectsCorruptionAndImpossibleState(t *testing.T) {
	t.Parallel()

	var archive bytes.Buffer
	ex, err := NewExporter(&archive, testHeader(), Limits{})
	if err != nil {
		t.Fatal(err)
	}
	cp := ex.cp
	encoded, err := cp.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	corrupt := append([]byte(nil), encoded...)
	corrupt[len(corrupt)/2] ^= 1
	if _, err := ParseCheckpoint(corrupt, Limits{}); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("corruption = %v, want ErrIntegrity", err)
	}
	if _, err := ParseCheckpoint(encoded[:len(encoded)-1], Limits{}); !errors.Is(err, ErrMalformed) {
		t.Fatalf("truncation = %v, want ErrMalformed", err)
	}
	cp.nextSequence = 1
	if _, err := cp.MarshalBinary(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("impossible state = %v, want ErrInvalid", err)
	}
}
