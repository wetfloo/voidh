package id3v2

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wetfloo/voidh/file"
	"github.com/wetfloo/voidh/util"
)

func TestHeaderParsingId3Err(t *testing.T) {
	testDataErr := []byte{0x48, 0x44, 0x33, 0x00, 0x00, 0x00}

	reader := bytes.NewReader(testDataErr)
	counter := util.WrapReaderWithCounter(reader)
	_, err := newHeader(counter)

	assert.NotNil(t, err)
	e := err.(file.InvalidTag)
	assert.EqualValues(t, 0, e.Offset)
	assert.Equal(t, majorByteSeq[:], e.Expected)
	assert.Equal(t, testDataErr[:3], e.Actual)
}

