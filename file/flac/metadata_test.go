package flac

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnpacker(t *testing.T) {
	shifter := unpacker{}
	packedValue := uint64(0xFF_EE_DD_CC_BB_AA_99_88)

	assert.Equal(t, uint64(0xFF_E), shifter.unpack(packedValue, 12))
	assert.Equal(t, uint(12), shifter.bitsCount)

	assert.Equal(t, uint64(0x0E_DD_CC), shifter.unpack(packedValue, 20))
	assert.Equal(t, uint(32), shifter.bitsCount)

	assert.Equal(t, uint64(0b10111), shifter.unpack(packedValue, 5))
	assert.Equal(t, uint(37), shifter.bitsCount)

	assert.Equal(t, uint64(0b011), shifter.unpack(packedValue, 3))
	assert.Equal(t, uint(40), shifter.bitsCount)

	assert.Equal(t, uint64(0xAA_99_88), shifter.unpack(packedValue, 24))
	assert.Equal(t, uint(64), shifter.bitsCount)
}
