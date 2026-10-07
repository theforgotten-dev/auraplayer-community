package codecs

import (
	"fmt"
	"io"

	gomp3 "github.com/hajimehoshi/go-mp3"

	"playerdoit/internal/audio"
)

type mp3Decoder struct{}

func (mp3Decoder) Decode(source io.Reader) (Stream, error) {
	decoded, err := gomp3.NewDecoder(source)
	if err != nil {
		return Stream{}, fmt.Errorf("decode MP3: %w", err)
	}

	return Stream{
		PCM: decoded,
		Config: audio.Config{
			SampleRate:   decoded.SampleRate(),
			ChannelCount: 2,
			Format:       audio.FormatSignedInt16LE,
		},
	}, nil
}
