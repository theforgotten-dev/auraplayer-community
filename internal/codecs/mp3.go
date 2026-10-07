package codecs

import (
	"fmt"
	"io"
	"time"

	gomp3 "github.com/hajimehoshi/go-mp3"

	"playerdoit/internal/audio"
)

type mp3Decoder struct{}

func (mp3Decoder) Decode(source io.Reader) (stream Stream, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			stream = Stream{}
			err = fmt.Errorf("initialize MP3 decoder: panic: %v", recovered)
		}
	}()

	decoded, err := gomp3.NewDecoder(source)
	if err != nil {
		return Stream{}, fmt.Errorf("decode MP3: %w", err)
	}

	config := audio.Config{
		SampleRate:   decoded.SampleRate(),
		ChannelCount: 2,
		Format:       audio.FormatSignedInt16LE,
	}

	return Stream{
		PCM:      newPCMTask(decoded),
		Config:   config,
		Duration: pcmDuration(decoded.Length(), config.SampleRate, 4),
	}, nil
}

func pcmDuration(byteLength int64, sampleRate, bytesPerFrame int) time.Duration {
	if byteLength <= 0 || sampleRate <= 0 || bytesPerFrame <= 0 {
		return 0
	}
	bytesPerSecond := int64(sampleRate * bytesPerFrame)
	seconds := byteLength / bytesPerSecond
	remainder := byteLength % bytesPerSecond
	return time.Duration(seconds)*time.Second + time.Duration(remainder)*time.Second/time.Duration(bytesPerSecond)
}
