// Package codecs converts encoded audio sources into PCM streams.
package codecs

import (
	"fmt"
	"io"
	"strings"

	"playerdoit/internal/audio"
)

// Stream is decoded PCM together with the format required by the audio engine.
type Stream struct {
	PCM    io.ReadSeeker
	Config audio.Config
}

// Decoder converts an encoded source into a seekable PCM stream.
type Decoder interface {
	Decode(source io.Reader) (Stream, error)
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
	default:
		return nil, fmt.Errorf("unsupported audio format %q", extension)
	}
}
