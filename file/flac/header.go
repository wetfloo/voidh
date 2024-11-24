package flac

import (
	"io"
)

type header struct {
}

func newHeader(input io.ByteReader) (header, error) {
	return header{}, nil
}
