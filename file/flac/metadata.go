package flac

import (
	"bufio"
	"fmt"
	"io"

	"github.com/wetfloo/voidh/file"
	"github.com/wetfloo/voidh/util"
)

type metadataBlockType byte

const (
	typeStreamInfo metadataBlockType = iota
	typePadding
	typeApplication
	typeSeekTable
	typeVorbisComment
	typeCuesheet
	typePicture
	typeInvalid = 127
)

type metadata struct {
	streamInfo streamInfo
}

type metadataBlock struct {
	header metadataHeader
	data   any
}

type metadataHeader struct {
	isLast    bool // TODO
	blockType metadataBlockType
	dataLen   uint32 // TODO
}

type streamInfo struct {
	minBlockSize   uint16
	maxBlockSize   uint16
	minFrameSize   uint32
	maxFrameSize   uint32
	sampleRate     uint32
	channels       uint8
	bitsPerSample  uint8
	samplesTotal   uint64
	audioUnencHash util.Md5
}

type application struct {
	appId   uint32
	appData []byte // TODO
}

// The number of seek points is implied by the metadata header 'length' field, i.e. equal to length / 18.
type seekTable struct {
	seekPoints []seekPoint
}

type seekPoint struct {
	sampleNum              uint64
	offset                 uint64
	targetFrameSampleCount uint16
}

type vorbisComment struct {
	bytes []byte // TODO
}

type cuesheet struct {
	// TODO, says it's ascii readable, meaning that we could use utf-8 string here
	mediaCatalogNum [128]byte
	leadInSamples   uint64
	isCompactDisc   bool
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

func readMetadata(r io.Reader) (metadata, error) {
	input := bufio.NewReader(r)
	var result metadata
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

func readMetadataBlock(input *bufio.Reader) (*metadataBlock, error) {
	var result metadataBlock
	b, err := input.ReadByte()
	if err != nil {
		return nil, err
	}

	isLast := util.FindBit(b, 0)
	result.header.isLast = isLast

	blockType := b & 0b0111_1111

	metadataFollowLen, err := util.ReadUint24(input)
	if err != nil {
		return nil, err
	}
	result.header.dataLen = metadataFollowLen

	switch blockType {
	case byte(typeStreamInfo):
		readStreamInfo(input)
	case byte(typePadding):
		if _, err := input.Discard(int(metadataFollowLen)); err != nil {
			return nil, err
		}
	case byte(typeApplication):
		readApplication(input, metadataFollowLen)
	case byte(typeSeekTable):
		readSeekTable(input, metadataFollowLen)
	case byte(typeVorbisComment):
		readVorbisComment(input, metadataFollowLen)
	case byte(typeCuesheet):
		readCuesheet(input, metadataFollowLen)
	case byte(typePicture):
		readPicture(input, metadataFollowLen)
	case byte(typeInvalid):
		return nil, fmt.Errorf("TODO: invalid metadata block")
	}

	return &result, nil
}

func readStreamInfo(input io.ByteReader) (streamInfo, error) {
	var result streamInfo

	minBlockSize, err := util.ReadUint16(input)
	if err != nil {
		return result, err
	}
	result.minBlockSize = minBlockSize

	maxBlockSize, err := util.ReadUint16(input)
	if err != nil {
		return result, err
	}
	result.maxBlockSize = maxBlockSize

	minFrameSize, err := util.ReadUint24(input)
	if err != nil {
		return result, err
	}
	result.minFrameSize = minFrameSize

	maxFrameSize, err := util.ReadUint24(input)
	if err != nil {
		return result, err
	}
	result.maxFrameSize = maxFrameSize

	num, err := util.ReadUint64(input)
	if err != nil {
		return result, err
	}

	unpacker := util.NewUnpacker()
	result.sampleRate = uint32(unpacker.Unpack(num, 20))
	result.channels = uint8(unpacker.Unpack(num, 3))
	result.bitsPerSample = uint8(unpacker.Unpack(num, 5))
	result.samplesTotal = unpacker.Unpack(num, 36)

	var audioUnencHash [16]byte
	for i := range audioUnencHash {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		audioUnencHash[i] = b
	}
	result.audioUnencHash = util.Md5{Bytes: audioUnencHash}

	return result, nil
}

func readApplication(input io.ByteReader, l uint32) (application, error) {
	result := application{
		appData: []byte{},
	}

	appId, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.appId = appId

	for i := uint32(0); i < l-2; i += 1 {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		result.appData = append(result.appData, b)
	}

	return result, nil
}

func readSeekTable(input io.ByteReader, l uint32) (seekTable, error) {
	result := seekTable{
		seekPoints: []seekPoint{},
	}

	for i := uint32(0); i < l; {
		var point seekPoint
		sampleNum, err := util.ReadUint64(input)
		if err != nil {
			return result, err
		}
		point.sampleNum = sampleNum
		i += 8

		offset, err := util.ReadUint64(input)
		if err != nil {
			return result, err
		}
		point.offset = offset
		i += 8

		targetFrameSampleCount, err := util.ReadUint16(input)
		if err != nil {
			return result, err
		}
		point.targetFrameSampleCount = targetFrameSampleCount
		i += 2
	}

	return result, nil
}

func readVorbisComment(input io.ByteReader, l uint32) (vorbisComment, error) {
	result := vorbisComment{
		bytes: []byte{},
	}

	for i := uint32(l); i < l; i += 1 {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		result.bytes = append(result.bytes, b)
	}

	return result, nil
}

func readCuesheet(input *bufio.Reader, l uint32) (cuesheet, error) {
	result := cuesheet{
		cuesheetTracks: []cuesheetTrack{},
	}

	for i := uint32(l); i < l; {
		for j := 0; j < 128; j += 1 {
			b, err := input.ReadByte()
			if err != nil {
				return result, err
			}
			result.mediaCatalogNum[j] = b
			i += 1
		}

		leadInSamples, err := util.ReadUint64(input)
		if err != nil {
			return result, err
		}
		result.leadInSamples = leadInSamples
		i += 8

		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		result.isCompactDisc = util.FindBit(b, 0)
		i += 1

		if _, err := input.Discard(258); err != nil {
			return result, err
		}

		tracksNum, err := util.ReadUint8(input)
		if err != nil {
			return result, err
		}
		result.tracksNum = tracksNum
		i += 1

		// TODO: finish later, too sleepy now >.<
	}

	return result, nil
}

func readCuesheetTrack(input *bufio.Scanner) (cuesheetTrack, error) {
	result := cuesheetTrack{
		indicies: []cuesheetTrackIndex{},
	}

	return result, nil
}

func readPicture(input io.ByteReader, l uint32) (picture, error) {
	var result picture

	return result, nil
}
