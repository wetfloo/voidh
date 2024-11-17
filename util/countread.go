package util

import (
	"bufio"
	"io"
)

type combinedReader interface {
	io.Reader
	io.ByteReader
}

func WrapReaderWithCounter(r io.Reader) *ReaderCounter {
	var reader combinedReader
	switch v := r.(type) {
	case combinedReader:
		reader = v
	default:
		reader = bufio.NewReader(r)
	}
	return &ReaderCounter{r: reader}
}

type ReaderCounter struct {
	r     combinedReader
	count int
}

func (rc *ReaderCounter) Read(p []byte) (n int, err error) {
	read, err := rc.r.Read(p)
	rc.count += read
	return read, err
}

func (rc *ReaderCounter) ReadByte() (byte, error) {
	b, err := rc.r.ReadByte()
	if err == nil {
		rc.count += 1
	}
	return b, err
}

func (rc *ReaderCounter) Count() int {
	return rc.count
}
