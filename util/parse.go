package util

import (
	"encoding/binary"
	"io"
)

func ReadUint64(input io.ByteReader) (uint64, error) {
	var out [8]byte
	for i, _ := range out {
		b, err := input.ReadByte()
		if err != nil {
			return 0, err
		}
		out[i] = b
	}
	return binary.BigEndian.Uint64(out[:]), nil
}

// Try to read uint32 from input, consuming 4 bytes. Big endian
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

// Try to read uint24 from input, consuming 3 bytes. Big endian
func ReadUint24(input io.ByteReader) (uint32, error) {
	var out [4]byte
	for i, _ := range out[:len(out)-1] {
		b, err := input.ReadByte()
		if err != nil {
			return 0, err
		}
		out[i+1] = b
	}
	return binary.BigEndian.Uint32(out[:]), nil
}

// Try to read uint16 from input, consuming 2 bytes. Big endian
func ReadUint16(input io.ByteReader) (uint16, error) {
	var out [2]byte
	for i, _ := range out {
		b, err := input.ReadByte()
		if err != nil {
			return 0, err
		}
		out[i] = b
	}
	return binary.BigEndian.Uint16(out[:]), nil
}

type Md5 struct {
	Bytes [16]byte
}
