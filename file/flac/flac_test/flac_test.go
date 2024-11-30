package flac_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wetfloo/voidh/file/flac"
)

const baseDir = "../testdata/flac-test-files/"

func TestReadAll(t *testing.T) {
	dir := baseDir + "subset/"
	list, err := os.ReadDir(dir)
	assert.Nil(t, err)
	assert.Greater(t, len(list), 0)
	flacList := []string{}

	for _, item := range list {
		if strings.HasSuffix(item.Name(), ".flac") {
			flacList = append(flacList, item.Name())
		}
	}

	for _, item := range flacList {
		f, err := os.Open(dir + item)
		assert.Nil(t, err)

		stream, err := flac.ReadStream(f)
		assert.Nil(t, err)
		assert.Greater(t, len(stream.Metadata), 0)
		if len(stream.Metadata) > 0 {
			assert.IsType(t, flac.StreamInfo{}, stream.Metadata[0])
		}
	}
}

func TestDataSubset56JpgPicture(t *testing.T) {
	f, err := os.Open(baseDir + "subset/56 - JPG PICTURE.flac")
	assert.Nil(t, err)

	stream, err := flac.ReadStream(f)
	assert.Nil(t, err)

	assert.GreaterOrEqual(t, len(stream.Metadata), 2) // one for StreamInfo, one for Picture
	for _, block := range stream.Metadata {
		switch v := block.(type) {
		case flac.StreamInfo:
			assert.EqualValues(t, 44100, v.SampleRate)
			assert.EqualValues(t, 16, v.BitsPerSample + 1)
			assert.EqualValues(t, 2, v.Channels + 1)
		case flac.Picture:
			assert.Equal(t, flac.PicTypeCoverBack, v.PicType)
			assert.Equal(t, "image/jpeg", v.MimeType)
			assert.EqualValues(t, 1920, v.Width)
			assert.EqualValues(t, 1080, v.Height)
		}
	}
}
