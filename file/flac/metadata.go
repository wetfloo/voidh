package flac

import (
	"io"

	"github.com/wetfloo/voidh/file"
	"github.com/wetfloo/voidh/util"
)

type metadataBlockType uint8

const (
	typeStreamInfo metadataBlockType = iota
	typePadding
	typeApplication
	typeSeekTable
	typeVorbisComment
	typeCuesheet
	typePicture
)

type header struct {
	streamInfo streamInfo
}

type metadataHeader struct {
	isLast    bool // TODO
	blockType metadataBlockType
	dataLen   uint32 // TODO
}

type streamInfo struct {
	minSize        uint16
	maxSize        uint16
	minFrameSize   uint32
	maxFrameSize   uint32
	sampleRate     uint32
	channels       uint8
	bitsPerSample  uint8
	samplesTotal   uint64
	audioUnencHash util.Md5
}

// TODO: this struct only exists for Alice to not be stupid
// and not forget to skip over paddings
type padding struct {
	bytes []byte
}

type application struct {
	appId   uint32
	appData []byte // TODO
}

// The number of seek points is implied by the metadata header 'length' field, i.e. equal to length / 18.
type seektable struct {
	seekpoints []seekpoint
}

type seekpoint struct {
	sampleNumber           uint64
	offset                 uint64
	targetFrameSampleCount uint16
}

type vorbisComment struct {
	bytes []byte // TODO
}

type cuesheet struct {
	mediaCatalogNum [128]byte
	leadInSamples   uint64
	isCompactDisc   bool
	reserved        [259]byte
	tracksNum       uint8
	cuesheetTracks  []cuesheetTrack
}

type cuesheetTrack struct {
	offset      uint64
	trackNum    uint8
	isrc        [12]byte
	isAudio     bool
	preEmphasis bool
	reserved    [14]byte
	indexPoints uint8
	indicies    []cuesheetTrackIndex
}

type cuesheetTrackIndex struct {
	offset        uint64
	indexPointNum uint8
	reserved      [3]byte
}

type picture struct {
	picType     uint32
	mimeTypeLen uint32 // TODO
	mimeType    string
	descLen     uint32 // TODO
	desc        string
	width       uint32
	height      uint32
	colorDepth  uint32
	colorsCount uint32
	dataLen     uint32 // TODO
	data        []byte
}

func newHeader(input io.ByteReader) (header, error) {
	var result header
	var fileHeader [4]byte

	for i := range fileHeader {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		fileHeader[i] = b
	}

	var refFlacHeader = [...]byte{0x66, 0x4c, 0x61, 0x63}
	if fileHeader != refFlacHeader {
		return result, file.InvalidTag{
			Offset:   0,
			Expected: refFlacHeader[:],
			Actual:   fileHeader[:],
		}
	}

	return result, nil
}
