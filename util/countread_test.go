package util_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wetfloo/voidh/util"
)

func TestCountReadEmpty(t *testing.T) {
	reader := bytes.NewReader([]byte{})
	rc := util.WrapReaderWithCounter(reader)
	_, err := rc.ReadByte()
	assert.NotNil(t, err)
	assert.EqualValues(t, 0, rc.Count())
}

func TestCountReadBytes(t *testing.T) {
	input := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	reader := bytes.NewReader(input)
	rc := util.WrapReaderWithCounter(reader)

	for i, item := range input {
		b, err := rc.ReadByte()
		assert.Nil(t, err)
		assert.Equal(t, item, b)
		assert.Equal(t, i+1, rc.Count())
	}
}
