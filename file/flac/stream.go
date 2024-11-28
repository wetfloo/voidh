package flac

import (
	"bufio"
	"io"

	"github.com/wetfloo/voidh/file"
)

type stream struct {
	metadata []metadataBlock
	frames   []frame // TODO
}

func readStream(r io.Reader) (stream, error) {
	input := bufio.NewReader(r)

	result := stream{
		metadata: []metadataBlock{},
		frames:   []frame{},
	}
	var fileHeader [4]byte

	for i := range fileHeader {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		fileHeader[i] = b
	}

	var refFlacHeader = [...]byte{0x66, 0x4c, 0x61, 0x63}
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
		result.metadata = append(result.metadata, *mb)
		if isLast {
			break
		}
	}

	return result, nil
}
