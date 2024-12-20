package id3v2

import (
	"fmt"
	"github.com/wetfloo/voidh/file"
	"github.com/wetfloo/voidh/util"
	"io"
)

const minorVerUpperBound = 0xFF - 1
const tagSizeUpperBound = 0x80 - 1

var majorByteSeq = [...]byte{0x49, 0x44, 0x33}

type header struct {
	minorVer       uint8
	revision       uint8
	flags          headerFlags
	tagSize        uint32
	extendedHeader *extendedHeader
}

type headerFlags struct {
	raw byte
}

func (flags headerFlags) footerPresent() bool {
	return util.FindBit(flags.raw, 4)
}

func (flags headerFlags) experimental() bool {
	return util.FindBit(flags.raw, 5)
}

func (flags headerFlags) extendedHeaderPresent() bool {
	return util.FindBit(flags.raw, 6)
}

func (flags headerFlags) unsync() bool {
	return util.FindBit(flags.raw, 7)
}

func newHeader(input io.ByteReader) (header, error) {
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
	result.flags = headerFlags{raw: b}

	// Check tag size, 4 bytes
	tagSize, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.tagSize = tagSize

	if result.tagSize > tagSizeUpperBound {
		// TODO: update error type here?
		return result, fmt.Errorf("invalid tag size, expected max of %x, but got %x", tagSizeUpperBound, result.tagSize)
	}

	if result.flags.extendedHeaderPresent() {
		if err := result.attachExtendedHeader(input); err != nil {
			return result, err
		}
	}

	return result, nil
}

func (header *header) attachExtendedHeader(input io.ByteReader) error {
	var result extendedHeader

	selfSize, err := util.ReadUint32(input)
	if err != nil {
		return err
	}
	if selfSize < 6 {
		return fmt.Errorf("invalid size of an extended header, must be at least 6 bytes, but is %d bytes instead", selfSize)
	}
	result.selfSize = selfSize

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

	for i, b := range extFlags {
		switch {
		case util.FindBit(b, 4):
			i += 1
			result.flags = append(result.flags, restrictionsFlag{data: extFlags[i]})
		case util.FindBit(b, 5):
			var flag crcFlag
			for j := 0; j < len(flag.data); j++ {
				i += 1
				flag.data[j] = extFlags[i]
			}
			result.flags = append(result.flags, flag)
		case util.FindBit(b, 6):
			result.flags = append(result.flags, updateFlag{})
		}
	}

	header.extendedHeader = &result
	return nil
}

type extendedHeader struct {
	selfSize uint32
	flags    []extendedHeaderFlag
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
