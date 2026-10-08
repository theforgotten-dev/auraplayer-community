// Package codecs converts encoded audio sources into PCM streams.
package codecs

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"playerdoit/internal/audio"
)

// Stream is decoded PCM together with the format required by the audio engine.
type Stream struct {
	PCM      io.ReadSeeker
	Config   audio.Config
	Duration time.Duration
}

// Decoder converts an encoded source into a seekable PCM stream.
type Decoder interface {
	Decode(source io.Reader) (Stream, error)
}

var supportedExtensions = []string{".mp3", ".wav", ".flac", ".ogg", ".aac", ".m4a", ".wma"}

// SupportedExtensions returns the encoded formats accepted by the player.
func SupportedExtensions() []string {
	return slices.Clone(supportedExtensions)
}

// IsSupported reports whether an extension has a registered decoder.
func IsSupported(extension string) bool {
	return slices.Contains(supportedExtensions, strings.ToLower(extension))
}

// Name returns the display name for a registered extension.
func Name(extension string) string {
	switch strings.ToLower(extension) {
	case ".mp3":
		return "MP3"
	case ".wav":
		return "WAV"
	case ".flac":
		return "FLAC"
	case ".ogg":
		return "OGG/VORBIS"
	case ".aac":
		return "AAC"
	case ".m4a":
		return "M4A/AAC"
	case ".wma":
		return "WMA"
	default:
		return strings.TrimPrefix(strings.ToUpper(extension), ".")
	}
}

// Decode selects a decoder from the file extension and returns its PCM stream.
func Decode(extension string, source io.Reader) (Stream, error) {
	decoder, err := DecoderFor(extension)
	if err != nil {
		return Stream{}, err
	}
	return decoder.Decode(source)
}

// DecoderFor returns the decoder registered for a file extension.
func DecoderFor(extension string) (Decoder, error) {
	switch strings.ToLower(extension) {
	case ".mp3":
		return mp3Decoder{}, nil
	case ".wav":
		return beepDecoder{format: beepWAV}, nil
	case ".flac":
		return beepDecoder{format: beepFLAC}, nil
	case ".ogg":
		return beepDecoder{format: beepVorbis}, nil
	case ".aac", ".m4a", ".wma":
		return platformDecoder{}, nil
	default:
		return nil, fmt.Errorf("unsupported audio format %q", extension)
	}
}
