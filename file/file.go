package file

type FsFile struct {
	Hash []byte
	Name string
}

type AudioFile struct {
	fsFile FsFile
	audioStreamHash []byte
}
