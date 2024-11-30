package flac

import (
	"bufio"
	"fmt"
	"github.com/wetfloo/voidh/util"
	"io"
	"strings"
)

type MetadataBlockType byte
type PicType uint32

const (
	MetadataBlockTypeStreamInfo MetadataBlockType = iota
	MetadataBlockTypePadding
	MetadataTypeApplication
	MetadataTypeSeekTable
	MetadataTypeVorbisComment
	MetadataTypeCuesheet
	MetadataTypePicture
	MetadataTypeInvalid = 127
)

const (
	PicTypeOther PicType = iota
	PicTypeFileIcon
	PicTypeCoverFront
	PicTypeCoverBack
	PicTypeLeafletPage
	PicTypeMedia
	PicTypeLeadArtist
	PicTypeArtist
	PicTypeConductor
	PicTypeBandOrchestra
	PicTypeComposer
	PicTypeLyricist
	PicTypeRecordingLocation
	PicTypeDuringRecording
	PicTypeDuringPerformance
	PicTypeMovie
	PicTypeBrightFish
	PicTypeIllustration
	PicTypeBandLogo
	PicTypePublisherLogo
)

type StreamInfo struct {
	MinBlockSize   uint16
	MaxBlockSize   uint16
	MinFrameSize   uint32
	MaxFrameSize   uint32
	SampleRate     uint32
	Channels       uint8
	BitsPerSample  uint8
	SamplesTotal   uint64
	AudioUnencHash util.Md5
}

type Application struct {
	AppId   uint32
	AppData []byte // TODO
}

// The number of seek points is implied by the metadata header 'length' field, i.e. equal to length / 18.
type SeekTable struct {
	SeekPoints []SeekPoint
}

type SeekPoint struct {
	SampleNum              uint64
	Offset                 uint64
	TargetFrameSampleCount uint16
}

type VorbisComment struct {
	Bytes []byte // TODO
}

type Cuesheet struct {
	// TODO, says it's ascii readable, meaning that we could use utf-8 string here
	MediaCatalogNum [128]byte
	LeadInSamples   uint64
	IsCompactDisc   bool
	CuesheetTracks  []CuesheetTrack
}

type CuesheetTrack struct {
	offset      uint64
	trackNum    uint8
	isrc        [12]byte
	isAudio     bool
	preEmphasis bool
	indicies    []cuesheetTrackIndex
}

type cuesheetTrackIndex struct {
	offset        uint64
	indexPointNum uint8
}

type Picture struct {
	PicType     PicType
	MimeType    string
	Desc        string
	Width       uint32
	Height      uint32
	ColorDepth  uint32
	ColorsCount uint32
	Data        []byte
}

func readMetadataBlock(input *bufio.Reader) (any, bool, error) {
	isLast := false

	b, err := input.ReadByte()
	if err != nil {
		return nil, isLast, err
	}

	isLast = util.FindBit(b, 7)

	blockType := b & 0b0111_1111

	metadataFollowLen, err := util.ReadUint24(input)
	if err != nil {
		return nil, isLast, err
	}

	switch blockType {
	case byte(MetadataBlockTypeStreamInfo):
		result, err := readStreamInfo(input)
		if err != nil {
			return nil, isLast, err
		}
		result.AssertReadBytesEq(uint64(metadataFollowLen))
		return result.Value, isLast, err
	case byte(MetadataBlockTypePadding):
		if _, err := input.Discard(int(metadataFollowLen)); err != nil {
			return nil, isLast, err
		}
	case byte(MetadataTypeApplication):
		result, err := readApplication(input, metadataFollowLen)
		if err != nil {
			return nil, isLast, err
		}
		result.AssertReadBytesEq(uint64(metadataFollowLen))
		return result.Value, isLast, err
	case byte(MetadataTypeSeekTable):
		result, err := readSeekTable(input, metadataFollowLen)
		if err != nil {
			return nil, isLast, err
		}
		result.AssertReadBytesEq(uint64(metadataFollowLen))
		return result.Value, isLast, err
	case byte(MetadataTypeVorbisComment):
		result, err := readVorbisComment(input, metadataFollowLen)
		if err != nil {
			return nil, isLast, err
		}
		result.AssertReadBytesEq(uint64(metadataFollowLen))
		return result.Value, isLast, err
	case byte(MetadataTypeCuesheet):
		result, err := readCuesheet(input)
		if err != nil {
			return nil, isLast, err
		}
		result.AssertReadBytesEq(uint64(metadataFollowLen))
		return result.Value, isLast, err
	case byte(MetadataTypePicture):
		result, err := readPicture(input, metadataFollowLen)
		if err != nil {
			return nil, isLast, err
		}
		result.AssertReadBytesEq(uint64(metadataFollowLen))
		return result.Value, isLast, err
	case byte(MetadataTypeInvalid):
		return nil, isLast, fmt.Errorf("TODO: invalid metadata block")
	}

	return nil, isLast, nil
}

func readStreamInfo(input io.ByteReader) (util.ReadResult[StreamInfo], error) {
	var result util.ReadResult[StreamInfo]

	minBlockSize, err := util.ReadUint16(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(2)
	result.Value.MinBlockSize = minBlockSize

	maxBlockSize, err := util.ReadUint16(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(2)
	result.Value.MaxBlockSize = maxBlockSize

	minFrameSize, err := util.ReadUint24(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(3)
	result.Value.MinFrameSize = minFrameSize

	maxFrameSize, err := util.ReadUint24(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(3)
	result.Value.MaxFrameSize = maxFrameSize

	num, err := util.ReadUint64(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(8)

	unpacker := util.NewUnpacker()
	result.Value.SampleRate = uint32(unpacker.Unpack(num, 20))
	result.Value.Channels = uint8(unpacker.Unpack(num, 3))
	result.Value.BitsPerSample = uint8(unpacker.Unpack(num, 5))
	result.Value.SamplesTotal = unpacker.Unpack(num, 36)

	var audioUnencHash [16]byte
	for i := range audioUnencHash {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		audioUnencHash[i] = b
	}
	result.AddReadBytes(16)
	result.Value.AudioUnencHash = util.Md5{Bytes: audioUnencHash}

	result.AssertReadBytesEq(34)

	return result, nil
}

func readApplication(input io.ByteReader, l uint32) (util.ReadResult[Application], error) {
	result := util.ReadResult[Application]{
		Value: Application{
			AppData: []byte{},
		},
	}

	appId, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(4)
	result.Value.AppId = appId

	for result.ReadBytes() < uint64(l) {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		result.AddReadBytes(1)
		result.Value.AppData = append(result.Value.AppData, b)
	}

	result.AssertReadBytesEq(uint64(l))

	return result, nil
}

func readSeekTable(input io.ByteReader, l uint32) (util.ReadResult[SeekTable], error) {
	result := util.ReadResult[SeekTable]{
		Value: SeekTable{
			SeekPoints: []SeekPoint{},
		},
	}

	for result.ReadBytes() < uint64(l) {
		var point SeekPoint
		sampleNum, err := util.ReadUint64(input)
		if err != nil {
			return result, err
		}
		result.AddReadBytes(8)
		point.SampleNum = sampleNum

		offset, err := util.ReadUint64(input)
		if err != nil {
			return result, err
		}
		result.AddReadBytes(8)
		point.Offset = offset

		targetFrameSampleCount, err := util.ReadUint16(input)
		if err != nil {
			return result, err
		}
		result.AddReadBytes(2)
		point.TargetFrameSampleCount = targetFrameSampleCount
	}

	result.AssertReadBytesEq(uint64(l))

	return result, nil
}

func readVorbisComment(input io.ByteReader, l uint32) (util.ReadResult[VorbisComment], error) {
	result := util.ReadResult[VorbisComment]{
		Value: VorbisComment{
			Bytes: []byte{},
		},
	}

	for result.ReadBytes() < uint64(l) {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		result.AddReadBytes(1)
		result.Value.Bytes = append(result.Value.Bytes, b)
	}

	result.AssertReadBytesEq(uint64(l))

	return result, nil
}

func readCuesheet(input *bufio.Reader) (util.ReadResult[Cuesheet], error) {
	result := util.ReadResult[Cuesheet]{
		Value: Cuesheet{
			CuesheetTracks: []CuesheetTrack{},
		},
	}

	for i := range result.Value.MediaCatalogNum {
		b, err := input.ReadByte()
		if err != nil {
			return result, err
		}
		result.AddReadBytes(1)
		result.Value.MediaCatalogNum[i] = b
	}

	leadInSamples, err := util.ReadUint64(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(8)
	result.Value.LeadInSamples = leadInSamples

	b, err := input.ReadByte()
	if err != nil {
		return result, err
	}
	result.AddReadBytes(1)
	result.Value.IsCompactDisc = util.FindBit(b, 7)

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
		result.Value.CuesheetTracks = append(result.Value.CuesheetTracks, cuesheetTrack.Value)
	}

	return result, nil
}

func readCuesheetTrack(input *bufio.Reader) (util.ReadResult[CuesheetTrack], error) {
	result := util.ReadResult[CuesheetTrack]{
		Value: CuesheetTrack{
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
	result.AddReadBytes(13)

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

func readPicture(input io.ByteReader, l uint32) (util.ReadResult[Picture], error) {
	result := util.ReadResult[Picture]{
		Value: Picture{
			Data: []byte{},
		},
	}

	pictureType, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(4)
	result.Value.PicType = PicType(pictureType)

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
	result.Value.MimeType = mimeType.String()

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
	result.Value.Desc = desc.String()

	width, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(4)
	result.Value.Width = width

	height, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(4)
	result.Value.Height = height

	colorDepth, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(4)
	result.Value.ColorDepth = colorDepth

	colorsCount, err := util.ReadUint32(input)
	if err != nil {
		return result, err
	}
	result.AddReadBytes(4)
	result.Value.ColorsCount = colorsCount

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
		result.Value.Data = append(result.Value.Data, b)
		result.AddReadBytes(1)
	}

	result.AssertReadBytesEq(uint64(l))

	return result, nil
}
