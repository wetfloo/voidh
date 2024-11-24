package util

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadUint24(t *testing.T) {
	b := [...]byte{0x40, 0xBB, 0xF4}
	expected := uint32(4242420)
	reader := bytes.NewReader(b[:])
	actual, err := ReadUint24(reader)
	assert.Nil(t, err)
	assert.Equal(t, expected, actual)
}
