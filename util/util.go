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
