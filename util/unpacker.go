package util

type Unpacker struct {
	unpackedBitsCount uint8
}

func NewUnpacker() Unpacker {
	return Unpacker{unpackedBitsCount: 0}
}

func (u *Unpacker) Reset() {
	u.unpackedBitsCount = 0
}

func (u *Unpacker) UnpackedBitsCount() uint8 {
	return u.unpackedBitsCount
}

func (u *Unpacker) Unpack(packed uint64, bitsCount uint8) uint64 {
	// handle overflows
	if u.unpackedBitsCount+bitsCount < u.unpackedBitsCount {
		return 0
	}

	mask := uint64(0)
	for i := uint8(0); i < bitsCount; i += 1 {
		mask += (1 << (64 - 1 - i - u.unpackedBitsCount))
	}
	result := (mask & packed) >> (64 - bitsCount - u.unpackedBitsCount)
	u.unpackedBitsCount += bitsCount
	return result
}
