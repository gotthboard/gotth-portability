package portability

import (
	"context"
	"encoding/binary"
	"io"
)

var streamMagic = [8]byte{'G', 'O', 'T', 'T', 'H', 'P', '1', '\n'}

const (
	recordFrame byte = 0x01
	footerFrame byte = 0xff
)

// encodeHeader returns the canonical bounded header representation.
// Complexity: time O(a+s), Omega(a+s), tight Theta(a+s); auxiliary space
// O(a+s), Omega(a+s), tight Theta(a+s); variables: a and s are identifier
// lengths.
func encodeHeader(h Header) []byte {
	b := make([]byte, 0, len(streamMagic)+2+2+len(h.ArchiveID)+2+len(h.Schema)+4)
	b = append(b, streamMagic[:]...)
	b = binary.BigEndian.AppendUint16(b, h.WireVersion)
	b = binary.BigEndian.AppendUint16(b, uint16(len(h.ArchiveID)))
	b = append(b, h.ArchiveID...)
	b = binary.BigEndian.AppendUint16(b, uint16(len(h.Schema)))
	b = append(b, h.Schema...)
	b = binary.BigEndian.AppendUint32(b, h.SchemaVersion)
	return b
}

// encodeRecordPrefix returns canonical frame metadata without payload bytes.
// Complexity: time O(k+r), Omega(k+r), tight Theta(k+r); auxiliary space
// O(k+r), Omega(k+r), tight Theta(k+r); variables: k and r are kind/key
// lengths.
func encodeRecordPrefix(m RecordMeta) []byte {
	b := make([]byte, 0, 1+8+2+len(m.Kind)+2+len(m.Key)+8)
	b = append(b, recordFrame)
	b = binary.BigEndian.AppendUint64(b, m.Sequence)
	b = binary.BigEndian.AppendUint16(b, uint16(len(m.Kind)))
	b = append(b, m.Kind...)
	b = binary.BigEndian.AppendUint16(b, uint16(len(m.Key)))
	b = append(b, m.Key...)
	b = binary.BigEndian.AppendUint64(b, m.Size)
	return b
}

// encodeFooter returns the fixed-size final oracle frame.
// Complexity: time O(1), Omega(1), tight Theta(1); auxiliary space O(1),
// Omega(1), tight Theta(1), with a fixed 49-byte allocation.
func encodeFooter(m Manifest) []byte {
	b := make([]byte, 0, 49)
	b = append(b, footerFrame)
	b = binary.BigEndian.AppendUint64(b, m.Records)
	b = binary.BigEndian.AppendUint64(b, m.PayloadBytes)
	b = append(b, m.Chain[:]...)
	return b
}

// writeExact performs exactly one Writer call and rejects every invalid or
// unexplained short result. Retrying a positive short nil write would hide a
// broken callback contract and could duplicate side effects.
// Complexity: delegated time is W(n); local time O(1)+W(n), Omega(1), tight
// Theta(1)+W(n); auxiliary space O(1), Omega(1), tight Theta(1).
func writeExact(w io.Writer, p []byte) (int, error) {
	n, err := w.Write(p)
	if n < 0 || n > len(p) {
		return 0, io.ErrShortWrite
	}
	if err != nil {
		return n, err
	}
	if n != len(p) {
		return n, io.ErrShortWrite
	}
	return n, nil
}

// writeExactContext brackets the single external Writer call with cancellation
// checks. A cancellation observed after the call means its outcome may exist,
// so callers must not advance a committed checkpoint.
// Complexity: for n=len(p), general time O(1)+W(n), Omega(1); a path that
// reaches the Writer has tight Theta(1)+W(n), while entry cancellation has
// tight Theta(1) and skips W. Auxiliary space is O(1), Omega(1), tight
// Theta(1). W is the cost of zero or one delegated Writer call; a context
// cannot interrupt that call while it is in flight.
func writeExactContext(ctx context.Context, w io.Writer, p []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	n, err := writeExact(w, p)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return n, ctxErr
	}
	return n, err
}

// readExactContext fills p while validating every Reader result. Legal
// transient (0, nil) results are bounded to prevent an infinite loop.
// Complexity: for n=len(p) and E=MaxConsecutiveEmptyReads, at most n positive
// progress calls and E empty calls between progress events give O(n*(E+1))
// local/call overhead plus aggregate delegated reader cost R(n,E); for fixed E
// this is tight Theta(n)+R(n,E) on success. Auxiliary space is O(1), Omega(1),
// tight Theta(1). A context cannot interrupt any Reader call in flight.
func readExactContext(ctx context.Context, r io.Reader, p []byte) (int, error) {
	total := 0
	emptyReads := 0
	for total < len(p) {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		n, err := r.Read(p[total:])
		if n < 0 || n > len(p)-total {
			return total, io.ErrShortBuffer
		}
		if n > 0 {
			total += n
			emptyReads = 0
		} else if err == nil {
			emptyReads++
			if emptyReads > MaxConsecutiveEmptyReads {
				return total, io.ErrNoProgress
			}
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return total, ctxErr
		}
		if err != nil {
			if err == io.EOF && total == len(p) {
				return total, nil
			}
			return total, err
		}
	}
	return total, nil
}

// probeEOFContext requires EOF after tolerating bounded transient empty reads.
// A positive count is returned to let callers classify trailing data and track
// physical progress; Reader counts are validated before use.
// Complexity: for E=MaxConsecutiveEmptyReads, time O(E)+R(E), Omega(1), tight
// Theta(E)+R(E) when all tolerated empty reads occur; auxiliary space O(1),
// Omega(1), tight Theta(1). At most E+1 Reader calls occur, and a context cannot
// interrupt any one of them while it is in flight.
func probeEOFContext(ctx context.Context, r io.Reader) (int, error) {
	var one [1]byte
	emptyReads := 0
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		n, err := r.Read(one[:])
		if n < 0 || n > len(one) {
			return 0, io.ErrShortBuffer
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return n, ctxErr
		}
		if n > 0 {
			if err != nil && err != io.EOF {
				return n, err
			}
			return n, nil
		}
		if err != nil {
			return 0, err
		}
		emptyReads++
		if emptyReads > MaxConsecutiveEmptyReads {
			return 0, io.ErrNoProgress
		}
	}
}

// checkedAdd returns a+b and whether unsigned overflow occurred.
// Complexity: time O(1), Omega(1), tight Theta(1); auxiliary space O(1),
// Omega(1), tight Theta(1).
func checkedAdd(a, b uint64) (uint64, bool) {
	c := a + b
	return c, c < a
}

// checkedMul returns a*b and whether unsigned overflow occurred.
// Complexity: time O(1), Omega(1), tight Theta(1); auxiliary space O(1),
// Omega(1), tight Theta(1).
func checkedMul(a, b uint64) (uint64, bool) {
	if a != 0 && b > ^uint64(0)/a {
		return 0, true
	}
	return a * b, false
}
