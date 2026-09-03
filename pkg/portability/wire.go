package portability

import (
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

// writeExact rejects unexplained short writes.
// Complexity: delegated time is W(n); local time O(n)+W(n), Omega(1), with
// tight Theta(n)+W(n) when the writer accepts all n bytes; auxiliary space
// O(1), Omega(1), tight Theta(1); variable n is len(p).
func writeExact(w io.Writer, p []byte) (int, error) {
	written := 0
	for written < len(p) {
		n, err := w.Write(p[written:])
		if n < 0 || n > len(p)-written {
			return written, io.ErrShortWrite
		}
		written += n
		if err != nil {
			return written, err
		}
		if n == 0 {
			return written, io.ErrShortWrite
		}
	}
	return written, nil
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
