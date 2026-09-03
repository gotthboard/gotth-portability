package portability

import (
	"errors"
	"testing"
)

func causeIs(err, target error) bool {
	var causer Causer
	return errors.As(err, &causer) && errors.Is(causer.Cause(), target)
}

func TestLimitsNormalizeAndValidate(t *testing.T) {
	t.Parallel()

	got, err := (Limits{}).normalize()
	if err != nil {
		t.Fatalf("normalize defaults: %v", err)
	}
	if got.MaxRecords != DefaultMaxRecords || got.MaxRecordBytes != DefaultMaxRecordBytes || got.MaxTotalBytes != DefaultMaxTotalBytes {
		t.Fatalf("unexpected defaults: %#v", got)
	}
	if _, err := (Limits{MaxRecordBytes: 10, MaxTotalBytes: 9}).normalize(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("inconsistent limits error = %v, want ErrInvalid", err)
	}
}

func TestHeaderAndRecordValidation(t *testing.T) {
	t.Parallel()

	valid := Header{WireVersion: WireVersion, ArchiveID: "archive", Schema: "schema", SchemaVersion: 3}
	if err := valid.validate(); err != nil {
		t.Fatalf("valid header: %v", err)
	}
	for _, tc := range []Header{
		{},
		{WireVersion: WireVersion + 1, ArchiveID: "archive", Schema: "schema"},
		{WireVersion: WireVersion, ArchiveID: "a\x00b", Schema: "schema"},
		{WireVersion: WireVersion, ArchiveID: "archive", Schema: "\xff"},
	} {
		if err := tc.validate(); !errors.Is(err, ErrInvalid) {
			t.Errorf("header %#v error = %v, want ErrInvalid", tc, err)
		}
	}

	if err := (RecordMeta{Sequence: 0, Kind: "post", Key: "42", Size: 12}).validate(); err != nil {
		t.Fatalf("valid record: %v", err)
	}
	if err := (RecordMeta{Kind: "", Key: "42"}).validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty kind error = %v, want ErrInvalid", err)
	}
}

func TestClassifiedErrorDoesNotExposeCauseText(t *testing.T) {
	t.Parallel()

	secret := errors.New("payload-secret")
	err := wrap(ErrMalformed, "read record", secret)
	if got := err.Error(); got != "portability: malformed stream: read record" {
		t.Fatalf("error text = %q", got)
	}
	if !errors.Is(err, ErrMalformed) || errors.Is(err, secret) {
		t.Fatalf("classification lost: %v", err)
	}
	var causer Causer
	if !errors.As(err, &causer) || !errors.Is(causer.Cause(), secret) {
		t.Fatalf("explicit cause unavailable: %v", err)
	}
}
