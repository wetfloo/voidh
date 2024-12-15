package flac

import (
	"bufio"
	"io"

	"github.com/wetfloo/voidh/file"
)

var refFlacHeader = [...]byte{0x66, 0x4c, 0x61, 0x43}

type Stream struct {
	Metadata []any
	Frames   []Frame // TODO: implement reading
}

type ReadCfg struct {
	ReadMetadata bool
	ReadFrames   bool
	// TODO: more fields, like strict assertions, etc.
}

func DefaultReadCfg() ReadCfg {
	return ReadCfg{
		ReadMetadata: true,
		ReadFrames:   true,
	}
}

func ReadStream(r io.Reader, cfg ReadCfg) (Stream, error) {
	input := bufio.NewReader(r)

	var result Stream
	var fileHeader [4]byte

	for i := range fileHeader {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		fileHeader[i] = b
	}

	if fileHeader != refFlacHeader {
		return result, file.InvalidTag{
			Offset:   0,
			Expected: refFlacHeader[:],
			Actual:   fileHeader[:],
		}
	}

	if cfg.ReadMetadata {
		result.Metadata = []any{}
		for {
			mb, isLast, err := readMetadataBlock(input)
			if err != nil {
				return result, err
			}
			if mb != nil {
				result.Metadata = append(result.Metadata, mb)
			}
			if isLast {
				break
			}
		}
	}

	if cfg.ReadFrames {
		result.Frames = []Frame{}
		// TODO
	}

	return result, nil
}
