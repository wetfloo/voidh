package util

// Gets the n-th (0-indexed) bit out of Most Significant Bit byte
func FindBit(b byte, n uint64) bool {
	sb := byte(1 << n)
	return sb == (sb & b)
}
