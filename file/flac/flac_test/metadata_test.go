package flac_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wetfloo/voidh/file/flac"
)

func TestDataSubset56JpgPicture(t *testing.T) {
	f, err := os.Open("../testdata/flac-test-files/subset/56 - JPG PICTURE.flac")
	assert.Nil(t, err)

	stream, err := flac.ReadStream(f)
	assert.Nil(t, err)

	assert.GreaterOrEqual(t, len(stream.Metadata), 2) // one for StreamInfo, one for Picture
	for _, block := range stream.Metadata {
		switch v := block.Data.(type) {
		case flac.StreamInfo:
			assert.EqualValues(t, 44100, v.SampleRate)
			assert.EqualValues(t, 16, v.BitsPerSample)
			assert.EqualValues(t, 2, v.Channels)
		case flac.Picture:
			assert.Equal(t, flac.PicTypeCoverFront, v.PicType)
			assert.Equal(t, "image/jpg", v.MimeType)
			assert.EqualValues(t, 1920, v.Width)
			assert.EqualValues(t, 1080, v.Height)
		}
	}
}
