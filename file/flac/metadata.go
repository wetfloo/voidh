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
type seekTable struct {
	seekPoints []seekPoint
}

type seekPoint struct {
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
		input.Discard(int(metadataFollowLen))
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

	// only 20 bits for sample rate
	shifter := shifter{toShift: 64}
	sampleRateMask := shifter.shiftLeft(0xFF_FF_F, 20)
	// then move according to spec
	channelsMask := shifter.shiftLeft(0b111, 3)
	bitsPerSampleMask := shifter.shiftLeft(channelsMask, 5)
	samplesTotalMask := shifter.shiftLeft(bitsPerSampleMask, 36)
	result.sampleRate = uint32(num & sampleRateMask)
	result.channels = uint8(num & channelsMask)
	result.bitsPerSample = uint8(num & bitsPerSampleMask)
	result.samplesTotal = num & samplesTotalMask

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

type shifter struct {
	toShift int
}

func (s *shifter) shiftLeft(value uint64, amount int) uint64 {
	if s.toShift <= 0 {
		return 0
	}

	result := value << (s.toShift - amount)
	s.toShift -= amount
	return result
}

func (s *shifter) reset() {
	s.toShift = 0
}

func readApplication(input io.ByteReader, l uint32) (application, error) {
	var result application

	appId, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.appId = appId

	appData := []byte{}
	for i := uint32(0); i < l-2; i += 1 {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		appData = append(appData, b)
	}
	result.appData = appData

	return result, nil
}

func readSeekTable(input io.ByteReader, l uint32) (seekTable, error) {
	var result seekTable

	return result, nil
}

func readVorbisComment(input io.ByteReader, l uint32) (vorbisComment, error) {
	var result vorbisComment

	return result, nil
}

func readCuesheet(input io.ByteReader, l uint32) (cuesheet, error) {
	var result cuesheet

	return result, nil
}

func readPicture(input io.ByteReader, l uint32) (picture, error) {
	var result picture

	return result, nil
}
