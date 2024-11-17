package file

import (
	"encoding/hex"
	"fmt"
)

type InvalidTag struct {
	Offset int64
	Expected []byte
	Actual   []byte
}

func (err InvalidTag) Error() string {
	return fmt.Sprintf(
		"Unexpected byte sequence at offset %x. Expected %s, but got %s. Maybe attempt to read it as different tags?",
		err.Offset,
		hex.EncodeToString(err.Expected),
		hex.EncodeToString(err.Actual),
	)
}
