package util

import (
	"encoding/binary"
	"io"
)

// Gets the n-th (0-indexed) bit out of Most Significant Bit byte
func FindBit(b byte, n uint64) bool {
	sb := byte(1 << n)
	return sb == (sb & b)
}

// Try to read uint32 from input, consuming 4 bytes
func ReadUint32(input io.ByteReader) (uint32, error) {
	var out [4]byte
	for i, _ := range out {
		b, err := input.ReadByte()
		if err != nil {
			return 0, err
		}
		out[i] = b
	}

	return binary.BigEndian.Uint32(out[:]), nil
}

type ByteReaderCounter struct {
	br    io.ByteReader
	count uintptr
}

func WrapByteReaderWithCounter(br io.ByteReader) io.ByteReader {
	return &ByteReaderCounter{br: br}
}

func (r *ByteReaderCounter) ReadByte() (byte, error) {
	b, err := r.br.ReadByte()
	if err != nil {
		return 0, err
	}
	r.count += 1
	return b, nil
}

func (r *ByteReaderCounter) Count() uintptr {
	return r.count
}
