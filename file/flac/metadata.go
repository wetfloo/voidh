package flac

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/wetfloo/voidh/file"
	"github.com/wetfloo/voidh/util"
)

type metadataBlockType byte
type picType uint32

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

const (
	picTypeOther picType = iota
	picTypeFileIcon
	picTypeCoverFront
	picTypeCoverBack
	picTypeLeafletPage
	picTypeMedia
	picTypeLeadArtist
	picTypeArtist
	picTypeConductor
	picTypeBandOrchestra
	picTypeComposer
	picTypeLyricist
	picTypeRecordingLocation
	picTypeDuringRecording
	picTypeDuringPerformance
	picTypeMovie
	picTypeBrightFish
	picTypeIllustration
	picTypeBandLogo
	picTypePublisherLogo
)

type metadata struct {
	streamInfo streamInfo
}

type metadataBlock struct {
	header metadataHeader
	data   any
}

type metadataHeader struct {
	isLast    bool
	blockType metadataBlockType
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
	cuesheetTracks  []cuesheetTrack
}

type cuesheetTrack struct {
	offset      uint64
	trackNum    uint8
	isrc        [12]byte
	isAudio     bool
	preEmphasis bool
	reserved    [14]byte
	indicies    []cuesheetTrackIndex
}

type cuesheetTrackIndex struct {
	offset        uint64
	indexPointNum uint8
	reserved      [3]byte
}

type picture struct {
	picType     picType
	mimeType    string
	desc        string
	width       uint32
	height      uint32
	colorDepth  uint32
	colorsCount uint32
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

	isLast := util.FindBit(b, 7)
	result.header.isLast = isLast

	blockType := b & 0b0111_1111

	metadataFollowLen, err := util.ReadUint24(input)
	if err != nil {
		return nil, err
	}

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
		readCuesheet(input)
	case byte(typePicture):
		readPicture(input, metadataFollowLen)
	case byte(typeInvalid):
		return nil, fmt.Errorf("TODO: invalid metadata block")
	}

	return &result, nil
}

func readStreamInfo(input io.ByteReader) (util.ReadResult[streamInfo], error) {
	var result util.ReadResult[streamInfo]

	minBlockSize, err := util.ReadUint16(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(2)
	result.Value.minBlockSize = minBlockSize

	maxBlockSize, err := util.ReadUint16(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(2)
	result.Value.maxBlockSize = maxBlockSize

	minFrameSize, err := util.ReadUint24(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(3)
	result.Value.minFrameSize = minFrameSize

	maxFrameSize, err := util.ReadUint24(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(3)
	result.Value.maxFrameSize = maxFrameSize

	num, err := util.ReadUint64(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(8)

	unpacker := util.NewUnpacker()
	result.Value.sampleRate = uint32(unpacker.Unpack(num, 20))
	result.Value.channels = uint8(unpacker.Unpack(num, 3))
	result.Value.bitsPerSample = uint8(unpacker.Unpack(num, 5))
	result.Value.samplesTotal = unpacker.Unpack(num, 36)

	var audioUnencHash [16]byte
	for i := range audioUnencHash {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		audioUnencHash[i] = b
	}
	result.AddReadBytes(16)
	result.Value.audioUnencHash = util.Md5{Bytes: audioUnencHash}

	result.AssertReadBytesEq(34)

	return result, nil
}

func readApplication(input io.ByteReader, l uint32) (util.ReadResult[application], error) {
	result := util.ReadResult[application]{
		Value: application{
			appData: []byte{},
		},
	}

	appId, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(4)
	result.Value.appId = appId

	for result.ReadBytes() < uint64(l) {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		result.AddReadBytes(1)
		result.Value.appData = append(result.Value.appData, b)
	}

	result.AssertReadBytesEq(uint64(l))

	return result, nil
}

func readSeekTable(input io.ByteReader, l uint32) (util.ReadResult[seekTable], error) {
	result := util.ReadResult[seekTable]{
		Value: seekTable{
			seekPoints: []seekPoint{},
		},
	}

	for result.ReadBytes() < uint64(l) {
		var point seekPoint
		sampleNum, err := util.ReadUint64(input)
		if err != nil {
			return result, err
		}
		result.AddReadBytes(8)
		point.sampleNum = sampleNum

		offset, err := util.ReadUint64(input)
		if err != nil {
			return result, err
		}
		result.AddReadBytes(8)
		point.offset = offset

		targetFrameSampleCount, err := util.ReadUint16(input)
		if err != nil {
			return result, err
		}
		result.AddReadBytes(2)
		point.targetFrameSampleCount = targetFrameSampleCount
	}

	result.AssertReadBytesEq(uint64(l))

	return result, nil
}

func readVorbisComment(input io.ByteReader, l uint32) (util.ReadResult[vorbisComment], error) {
	result := util.ReadResult[vorbisComment]{
		Value: vorbisComment{
			bytes: []byte{},
		},
	}

	for result.ReadBytes() < uint64(l) {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		result.AddReadBytes(1)
		result.Value.bytes = append(result.Value.bytes, b)
	}

	result.AssertReadBytesEq(uint64(l))

	return result, nil
}

func readCuesheet(input *bufio.Reader) (util.ReadResult[cuesheet], error) {
	result := util.ReadResult[cuesheet]{
		Value: cuesheet{
			cuesheetTracks: []cuesheetTrack{},
		},
	}

	for i := range result.Value.mediaCatalogNum {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		result.AddReadBytes(1)
		result.Value.mediaCatalogNum[i] = b
	}

	leadInSamples, err := util.ReadUint64(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(8)
	result.Value.leadInSamples = leadInSamples

	b, err := input.ReadByte()
	if err != nil {
		return result, err
	}
	result.AddReadBytes(1)
	result.Value.isCompactDisc = util.FindBit(b, 7)

	// reserved
	// TODO: check that those 258 bytes are all zero
	if _, err := input.Discard(258); err != nil {
		return result, err
	}
	result.AddReadBytes(258)

	tracksNum, err := util.ReadUint8(input)
	// TODO: check that tracksNum is >= 1
	if err != nil {
		return result, err
	}
	result.AddReadBytes(1)

	// 	result.AssertReadBytesEq(uint64(l)) // TODO

	for i := uint8(0); i < tracksNum; i += 1 {
		cuesheetTrack, err := readCuesheetTrack(input)
		if err != nil {
			return result, err
		}
		result.AddReadBytes(cuesheetTrack.ReadBytes())
		result.Value.cuesheetTracks = append(result.Value.cuesheetTracks, cuesheetTrack.Value)
	}

	return result, nil
}

func readCuesheetTrack(input *bufio.Reader) (util.ReadResult[cuesheetTrack], error) {
	result := util.ReadResult[cuesheetTrack]{
		Value: cuesheetTrack{
			indicies: []cuesheetTrackIndex{},
		},
	}

	offset, err := util.ReadUint64(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(8)
	result.Value.offset = offset

	trackNum, err := util.ReadUint8(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(1)
	result.Value.trackNum = trackNum

	var isrc [12]byte
	for i := range isrc {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		result.AddReadBytes(1)
		isrc[i] = b
	}
	result.Value.isrc = isrc

	b, err := input.ReadByte()
	if err != nil {
		return result, err
	}
	result.AddReadBytes(1)
	result.Value.isAudio = util.FindBit(b, 7)
	result.Value.preEmphasis = util.FindBit(b, 6)

	if _, err := input.Discard(13); err != nil {
		return result, err
	}

	indexPointsNum, err := util.ReadUint8(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(1)

	for i := uint8(0); i < indexPointsNum; i += 1 {
		index, err := readCuesheetTrackIndex(input)
		if err != nil {
			return result, err
		}
		result.AddReadBytes(index.ReadBytes())
		result.Value.indicies = append(result.Value.indicies, index.Value)
	}

	return result, nil
}

// Reads total of 12 bytes, if successful
func readCuesheetTrackIndex(input *bufio.Reader) (util.ReadResult[cuesheetTrackIndex], error) {
	var result util.ReadResult[cuesheetTrackIndex]

	offset, err := util.ReadUint64(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(8)
	result.Value.offset = offset

	indexPointNum, err := util.ReadUint8(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(1)
	result.Value.indexPointNum = indexPointNum

	// TODO: assert all zeroes
	if _, err := input.Discard(3); err != nil {
		return result, err
	}
	result.AddReadBytes(3)

	result.AssertReadBytesEq(12)

	return result, nil
}

func readPicture(input io.ByteReader, l uint32) (util.ReadResult[picture], error) {
	result := util.ReadResult[picture]{
		Value: picture{
			data: []byte{},
		},
	}

	pictureType, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(4)
	result.Value.picType = picType(pictureType)

	mimeTypeLen, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(4)

	var mimeType strings.Builder
	for i := uint32(0); i < mimeTypeLen; i += 1 {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		mimeType.WriteByte(b)
		result.AddReadBytes(1)
	}
	result.Value.mimeType = mimeType.String()

	descLen, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(4)

	var desc strings.Builder
	for i := uint32(0); i < descLen; i += 1 {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		desc.WriteByte(b)
		result.AddReadBytes(1)
	}
	result.Value.desc = desc.String()

	width, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(4)
	result.Value.width = width

	height, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(4)
	result.Value.height = height

	colorDepth, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(4)
	result.Value.colorDepth = colorDepth

	colorsCount, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(4)
	result.Value.colorsCount = colorsCount

	dataLen, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(4)

	for i := uint32(0); i < dataLen; i += 1 {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		result.Value.data = append(result.Value.data, b)
		result.AddReadBytes(1)
	}

	result.AssertReadBytesEq(uint64(l))

	return result, nil
}
