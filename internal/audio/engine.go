// Package audio defines the contracts shared by the audio engine and its users.
package audio

import (
	"errors"
	"io"
	"time"
)

// Format describes the representation of one PCM sample.
type Format uint8

const (
	FormatFloat32LE Format = iota
	FormatUnsignedInt8
	FormatSignedInt16LE
)

// Config fixes the PCM format used by an Engine for its whole lifetime.
type Config struct {
	SampleRate   int
	ChannelCount int
	Format       Format
	BufferSize   time.Duration
}

// DefaultConfig is suitable for CD-quality stereo PCM.
func DefaultConfig() Config {
	return Config{
		SampleRate:   44_100,
		ChannelCount: 2,
		Format:       FormatSignedInt16LE,
	}
}

// Validate reports unsupported audio settings before a device is opened.
func (c Config) Validate() error {
	if c.SampleRate <= 0 {
		return errors.New("sample rate must be greater than zero")
	}
	if c.ChannelCount != 1 && c.ChannelCount != 2 {
		return errors.New("channel count must be 1 or 2")
	}
	if c.Format > FormatSignedInt16LE {
		return errors.New("unsupported PCM format")
	}
	if c.BufferSize < 0 {
		return errors.New("buffer size cannot be negative")
	}
	return nil
}

// Engine owns the process-wide audio context and creates players that share
// its PCM configuration.
type Engine interface {
	NewPlayer(source io.ReadSeeker) Player
	Suspend() error
	Resume() error
	Err() error
}

// Player controls one seekable PCM stream. Implementations must be safe for
// concurrent use.
type Player interface {
	Play()
	Pause()
	Stop() error
	Seek(position time.Duration) error
	SetVolume(volume float64) error
	Volume() float64
	IsPlaying() bool
	Err() error
	Close() error
}
