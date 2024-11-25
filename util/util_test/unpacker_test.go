package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wetfloo/voidh/util"
)

func TestUnpacker(t *testing.T) {
	unpacker := util.NewUnpacker()
	packedValue := uint64(0xFF_EE_DD_CC_BB_AA_99_88)

	assert.Equal(t, uint64(0xFF_E), unpacker.Unpack(packedValue, 12))
	assert.Equal(t, uint8(12), unpacker.UnpackedBitsCount())

	assert.Equal(t, uint64(0x0E_DD_CC), unpacker.Unpack(packedValue, 20))
	assert.Equal(t, uint8(32), unpacker.UnpackedBitsCount())

	assert.Equal(t, uint64(0b10111), unpacker.Unpack(packedValue, 5))
	assert.Equal(t, uint8(37), unpacker.UnpackedBitsCount())

	assert.Equal(t, uint64(0b011), unpacker.Unpack(packedValue, 3))
	assert.Equal(t, uint8(40), unpacker.UnpackedBitsCount())

	assert.Equal(t, uint64(0xAA_99_88), unpacker.Unpack(packedValue, 24))
	assert.Equal(t, uint8(64), unpacker.UnpackedBitsCount())
}
