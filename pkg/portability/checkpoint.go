package portability

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
)

var checkpointMagic = [8]byte{'G', 'P', 'T', 'C', 'H', 'K', '1', '\n'}

const checkpointVersion uint16 = 1

// Checkpoint records a committed archive boundary. The underlying stream must
// be positioned at Offset before resumption.
type Checkpoint struct {
	header       Header
	nextSequence uint64
	records      uint64
	payloadBytes uint64
	offset       uint64
	chain        [32]byte
}

// Header returns the immutable-by-value archive header.
// Complexity: time O(1), Omega(1), tight Theta(1); auxiliary space O(1),
// Omega(1), tight Theta(1); strings share immutable Go string storage.
func (c Checkpoint) Header() Header { return c.header }

// NextSequence returns the sequence required for the next record.
// Complexity: time and auxiliary space O(1), Omega(1), tight Theta(1).
func (c Checkpoint) NextSequence() uint64 { return c.nextSequence }

// Records returns the committed record count.
// Complexity: time and auxiliary space O(1), Omega(1), tight Theta(1).
func (c Checkpoint) Records() uint64 { return c.records }

// PayloadBytes returns the committed cumulative payload size.
// Complexity: time and auxiliary space O(1), Omega(1), tight Theta(1).
func (c Checkpoint) PayloadBytes() uint64 { return c.payloadBytes }

// Offset returns the exact byte offset immediately after the committed frame.
// Complexity: time and auxiliary space O(1), Omega(1), tight Theta(1).
func (c Checkpoint) Offset() uint64 { return c.offset }

// validate rejects impossible or over-limit resume state.
// Complexity: time O(a+s), Omega(1), tight Theta(a+s) for valid state;
// auxiliary space O(1), Omega(1), tight Theta(1); variables: a and s are
// header identifier lengths.
func (c Checkpoint) validate(l Limits) error {
	if err := c.validateStructure(); err != nil {
		return err
	}
	if c.records > l.MaxRecords || c.payloadBytes > l.MaxTotalBytes {
		return wrap(ErrInvalid, "checkpoint counters", nil)
	}
	return nil
}

// validateStructure checks policy-independent checkpoint invariants.
// Complexity: time O(a+s), Omega(1), tight Theta(a+s) for valid state;
// auxiliary space O(a+s), Omega(1), tight Theta(a+s) because header length is
// reconstructed; variables a/s are identifier lengths.
func (c Checkpoint) validateStructure() error {
	if err := c.header.validate(); err != nil {
		return wrap(ErrInvalid, "checkpoint header", err)
	}
	if c.nextSequence != c.records {
		return wrap(ErrInvalid, "checkpoint counters", nil)
	}
	header := encodeHeader(c.header)
	headerLen := uint64(len(header))
	if c.records == 0 {
		initial := sha256.Sum256(header)
		if c.payloadBytes != 0 || c.offset != headerLen || c.chain != initial {
			return wrap(ErrInvalid, "empty checkpoint state", nil)
		}
		return nil
	}
	const minRecordOverhead = uint64(1 + 8 + 2 + 1 + 2 + 8 + sha256.Size)
	const maxRecordOverhead = uint64(1 + 8 + 2 + maxKindBytes + 2 + maxKeyBytes + 8 + sha256.Size)
	minFrames, overflow := checkedMul(c.records, minRecordOverhead)
	if overflow {
		return wrap(ErrInvalid, "checkpoint offset", nil)
	}
	minimum, overflow := checkedAdd(headerLen, c.payloadBytes)
	if overflow {
		return wrap(ErrInvalid, "checkpoint offset", nil)
	}
	minimum, overflow = checkedAdd(minimum, minFrames)
	if overflow || c.offset < minimum {
		return wrap(ErrInvalid, "checkpoint offset", nil)
	}
	maxFrames, maxOverflow := checkedMul(c.records, maxRecordOverhead)
	maximum, addOverflow := checkedAdd(headerLen, c.payloadBytes)
	if !maxOverflow && !addOverflow {
		maximum, addOverflow = checkedAdd(maximum, maxFrames)
	}
	if !maxOverflow && !addOverflow && c.offset > maximum {
		return wrap(ErrInvalid, "checkpoint offset", nil)
	}
	return nil
}

// MarshalBinary encodes a versioned, checksummed committed boundary.
// Complexity: time O(a+s), Omega(a+s), tight Theta(a+s); auxiliary space
// O(a+s), Omega(a+s), tight Theta(a+s); variables a/s are identifier lengths.
func (c Checkpoint) MarshalBinary() ([]byte, error) {
	if err := c.validateStructure(); err != nil {
		return nil, err
	}
	header := encodeHeader(c.header)
	b := make([]byte, 0, 8+2+2+len(header)+8*4+sha256.Size*2)
	b = append(b, checkpointMagic[:]...)
	b = binary.BigEndian.AppendUint16(b, checkpointVersion)
	b = binary.BigEndian.AppendUint16(b, uint16(len(header)))
	b = append(b, header...)
	b = binary.BigEndian.AppendUint64(b, c.nextSequence)
	b = binary.BigEndian.AppendUint64(b, c.records)
	b = binary.BigEndian.AppendUint64(b, c.payloadBytes)
	b = binary.BigEndian.AppendUint64(b, c.offset)
	b = append(b, c.chain[:]...)
	digest := sha256.Sum256(b)
	b = append(b, digest[:]...)
	return b, nil
}

// ParseCheckpoint verifies and decodes one exact checkpoint representation.
// Complexity: time O(n), Omega(n), tight Theta(n); auxiliary space O(a+s),
// Omega(1), tight Theta(a+s) for valid input; variables: n is len(data), a/s
// are decoded identifier lengths.
func ParseCheckpoint(data []byte, limits Limits) (Checkpoint, error) {
	normalized, err := limits.normalize()
	if err != nil {
		return Checkpoint{}, err
	}
	const fixedWithoutHeader = 8 + 2 + 2 + 8*4 + sha256.Size*2
	if len(data) < fixedWithoutHeader {
		return Checkpoint{}, wrap(ErrMalformed, "checkpoint length", nil)
	}
	if !bytes.Equal(data[:8], checkpointMagic[:]) {
		return Checkpoint{}, wrap(ErrMalformed, "checkpoint magic", nil)
	}
	if binary.BigEndian.Uint16(data[8:10]) != checkpointVersion {
		return Checkpoint{}, wrap(ErrIncompatible, "checkpoint version", nil)
	}
	headerLen := int(binary.BigEndian.Uint16(data[10:12]))
	const maxHeaderLen = 8 + 2 + 2 + maxArchiveIDBytes + 2 + maxSchemaBytes + 4
	if headerLen > maxHeaderLen || len(data) != fixedWithoutHeader+headerLen {
		return Checkpoint{}, wrap(ErrMalformed, "checkpoint header length", nil)
	}
	payload := data[:len(data)-sha256.Size]
	digest := sha256.Sum256(payload)
	if !bytes.Equal(digest[:], data[len(data)-sha256.Size:]) {
		return Checkpoint{}, wrap(ErrIntegrity, "checkpoint digest", nil)
	}
	headerReader := bytes.NewReader(data[12 : 12+headerLen])
	header, _, err := readHeader(headerReader)
	if err != nil || headerReader.Len() != 0 {
		return Checkpoint{}, wrap(ErrMalformed, "checkpoint header", err)
	}
	position := 12 + headerLen
	readUint64 := func() uint64 {
		value := binary.BigEndian.Uint64(data[position : position+8])
		position += 8
		return value
	}
	c := Checkpoint{
		header:       header,
		nextSequence: readUint64(),
		records:      readUint64(),
		payloadBytes: readUint64(),
		offset:       readUint64(),
	}
	copy(c.chain[:], data[position:position+sha256.Size])
	if err := c.validate(normalized); err != nil {
		return Checkpoint{}, err
	}
	return c, nil
}
