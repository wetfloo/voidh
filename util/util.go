package util

// Gets the n-th (0-indexed) bit out of Most Significant Bit byte
func FindBit(b byte, n uint64) bool {
	sb := byte(1 << n)
	return sb == (sb & b)
}

type ReadResult[T any] struct {
	Value     T
	readBytes uint64
}

// Add the amount of read bytes to total amount
func (r *ReadResult[any]) AddReadBytes(value uint64) {
	r.readBytes += value
}

func (r *ReadResult[any]) ReadBytes() uint64 {
	return r.readBytes
}
