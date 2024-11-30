package flac

import (
	"bufio"
	"io"

	"github.com/wetfloo/voidh/file"
)

type Stream struct {
	Metadata []any
	Frames   []Frame // TODO
}

func ReadStream(r io.Reader) (Stream, error) {
	input := bufio.NewReader(r)

	result := Stream{
		Metadata: []any{},
		Frames:   []Frame{},
	}
	var fileHeader [4]byte

	for i := range fileHeader {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		fileHeader[i] = b
	}

	var refFlacHeader = [...]byte{0x66, 0x4c, 0x61, 0x43}
	if fileHeader != refFlacHeader {
		return result, file.InvalidTag{
			Offset:   0,
			Expected: refFlacHeader[:],
			Actual:   fileHeader[:],
		}
	}

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

	return result, nil
}
