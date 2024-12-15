package ffmpeg

import (
	"context"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
)

type FfmpegHashAlgo string

const (
	Md5         FfmpegHashAlgo = "MD5"
	Murmur3                    = "murmur3"
	Ripemd128                  = "RIPEMD128"
	Ripemd160                  = "RIPEMD160"
	Ripemd256                  = "RIPEMD256"
	Ripemd320                  = "RIPEMD320"
	Sha160                     = "SHA160"
	Sha224                     = "SHA224"
	Sha256                     = "SHA256"
	Sha512tr224                = "SHA512/224"
	Sha512tr256                = "SHA512/256"
	Sha384                     = "SHA384"
	Sha512                     = "SHA512"
	Crc32                      = "CRC32"
	Adler32                    = "adler32"
)

// Decodes audio data hash, without considering metadata, using ffmpeg.
// Might return [*util.ExitError] if the command starts successfully, but fails to complete
func AudioDataHash(ctx context.Context, path string, algo FfmpegHashAlgo) ([]byte, error) {
	// TODO: implement handling of raw, decoded audio streams for all kinds of codecs... Maybe not here?
	// Uncoupling that from ffmpeg would be a good idea.

	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-i",
		path,
		"-codec",
		"copy",
		"-f",
		"hash",
		"-hash",
		string(algo),
		"-loglevel",
		"warning",
		"-",
	)

	var stdout strings.Builder
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	output := strings.TrimSpace(stdout.String())
	kv := strings.SplitN(output, "=", 2)
	if len(kv) != 2 {
		return nil, fmt.Errorf("invalid ffmpeg output len, expected 2, got %d instead", len(kv))
	}

	return hex.DecodeString(kv[1])
}
