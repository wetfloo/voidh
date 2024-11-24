package util

import (
	"encoding/binary"
	"io"
)

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

// Try to read uint24 from input, consuming 3 bytes
func ReadUint24(input io.ByteReader) (uint32, error) {
	var out [4]byte
	for i, _ := range out[:len(out)-1] {
		b, err := input.ReadByte()
		if err != nil {
			return 0, err
		}
		out[i] = b
	}

	return binary.BigEndian.Uint32(out[:]) >> 8, nil
}

type Md5 struct {
	Bytes [16]byte
}
