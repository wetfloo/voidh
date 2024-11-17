package id3v2

import (
	"encoding/binary"
	"fmt"
	"github.com/wetfloo/voidh/file"
	"io"
)

const (
	_ byte = 1 << iota
	_
	_
	_
	footerPresent
	experimental
	extendedHeaderPresent
	unsync
)
const (
	_ byte = 1 << iota
	_
	_
	_
	extTagRestrictions
	extCrcPresent
	extTagIsAnUpdate
)
const minorVerUpperBound = 0xFF - 1
const tagSizeUpperBound = 0x80 - 1

var majorByteSeq = [...]byte{0x49, 0x44, 0x33}

type header struct {
	minorVer       uint8
	revision       uint8
	flags          byte
	tagSize        uint32
	extendedHeader *extendedHeader
}

func newHeader(input io.ByteScanner) (header, error) {
	var result header

	// Check the major id3 version. Should always be v2
	var major [3]byte
	for i, _ := range major {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		major[i] = b
	}

	if major != majorByteSeq {
		return result, file.InvalidTag{
			Offset:   0,
			Expected: majorByteSeq[:],
			Actual:   major[:],
		}
	}

	// Check the minor version. Always just 1 byte
	b, err := input.ReadByte()
	if err != nil {
		return result, err
	}
	result.minorVer = uint8(b)
	if result.minorVer > minorVerUpperBound {
		// TODO: update error type here?
		return result, fmt.Errorf("invalid id3v2 minor version, expected max of %x, but got %x", minorVerUpperBound, result.minorVer)
	}

	// Check revision, also just 1 byte
	b, err = input.ReadByte()
	if err != nil {
		return result, err
	}
	result.revision = uint8(b)

	// Check flags, 1 byte
	b, err = input.ReadByte()
	if err != nil {
		return result, err
	}
	result.flags = uint8(b)

	// Check tag size, 4 bytes
	var tagSize [4]byte
	for i, _ := range tagSize {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		tagSize[i] = b
	}
	result.tagSize = binary.BigEndian.Uint32(tagSize[:])
	if result.tagSize > tagSizeUpperBound {
		// TODO: update error type here?
		return result, fmt.Errorf("invalid tag size, expected max of %x, but got %x", tagSizeUpperBound, result.tagSize)
	}

	if err := result.attachExtendedHeader(input); err != nil {
		return result, err
	}

	return result, nil
}

func (header *header) attachExtendedHeader(input io.ByteReader) error {
	var selfSize [4]byte
	var result extendedHeader

	for i, _ := range selfSize {
		b, err := input.ReadByte()
		if err != nil {
			return err
		}
		selfSize[i] = b
	}
	result.selfSize = binary.BigEndian.Uint32(selfSize[:])

	b, err := input.ReadByte()
	if err != nil {
		return err
	}
	extFlags := []byte{}
	flagBytesCount := uint8(b)
	for i := uint8(0); i < flagBytesCount; i++ {
		b, err = input.ReadByte()
		if err != nil {
			return err
		}
		extFlags = append(extFlags, b)
	}

	for i, extFlag := range extFlags {
		switch {
		case extFlag&extTagIsAnUpdate > 0:
			result.flags = append(result.flags, updateFlag{})
		case extFlag&extCrcPresent > 0:
			var flag crcFlag
			for j := 0; j < len(flag.data); j++ {
				i += 1
				flag.data[j] = extFlags[i]
			}
			result.flags = append(result.flags, flag)
		case extFlag&extTagRestrictions > 0:
			i += 1
			result.flags = append(result.flags, restrictionsFlag{data: extFlags[i]})
		}
	}

	return nil
}

type extendedHeaderFlag interface {
	raw() []byte
}

type updateFlag struct{}

func (_ updateFlag) raw() []byte {
	return nil
}

type crcFlag struct {
	data [5]byte
}

func (flag crcFlag) raw() []byte {
	return flag.data[:]
}

// TODO: update this with methods for checking specific restrictions if needed
type restrictionsFlag struct {
	data byte
}

func (flag restrictionsFlag) raw() []byte {
	return []byte{flag.data}
}

type extendedHeader struct {
	selfSize uint32
	flags    []extendedHeaderFlag
}
