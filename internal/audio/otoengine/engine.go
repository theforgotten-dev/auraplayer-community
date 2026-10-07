// Package otoengine implements the audio contracts with Oto v3.
package otoengine

import (
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/ebitengine/oto/v3"

	"playerdoit/internal/audio"
)

// Engine owns Oto's single process-wide audio context.
type Engine struct {
	context *oto.Context
	config  audio.Config
}

// New opens the default audio device and waits until it is ready.
// Oto supports only one Engine per process.
func New(config audio.Config) (*Engine, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid audio configuration: %w", err)
	}

	context, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:      config.SampleRate,
		ChannelCount:    config.ChannelCount,
		Format:          otoFormat(config.Format),
		BufferSize:      config.BufferSize,
		ApplicationName: "playerdoit",
	})
	if err != nil {
		return nil, fmt.Errorf("create Oto context: %w", err)
	}
	<-ready
	if err := context.Err(); err != nil {
		return nil, fmt.Errorf("initialize Oto context: %w", err)
	}

	return &Engine{context: context, config: config}, nil
}

func (e *Engine) NewPlayer(source io.ReadSeeker) audio.Player {
	otoPlayer := e.context.NewPlayer(source)
	return &player{
		player:           otoPlayer,
		bytesPerFrame:    int64(e.config.ChannelCount * bytesPerSample(e.config.Format)),
		samplesPerSecond: int64(e.config.SampleRate),
	}
}

func (e *Engine) Suspend() error { return e.context.Suspend() }
func (e *Engine) Resume() error  { return e.context.Resume() }
func (e *Engine) Err() error     { return e.context.Err() }

type player struct {
	player           *oto.Player
	bytesPerFrame    int64
	samplesPerSecond int64
}

func (p *player) Play()           { p.player.Play() }
func (p *player) Pause()          { p.player.Pause() }
func (p *player) IsPlaying() bool { return p.player.IsPlaying() }
func (p *player) Volume() float64 { return p.player.Volume() }
func (p *player) Err() error      { return p.player.Err() }
func (p *player) Close() error {
	p.player.PauseAndStopReading()
	return nil
}

func (p *player) Stop() error {
	p.player.Pause()
	if err := p.seekBytes(0); err != nil {
		return err
	}
	p.player.PauseAndStopReading()
	return nil
}

func (p *player) Seek(position time.Duration) error {
	offset, err := byteOffset(position, p.samplesPerSecond, p.bytesPerFrame)
	if err != nil {
		return err
	}
	return p.seekBytes(offset)
}

func (p *player) SetVolume(volume float64) error {
	if math.IsNaN(volume) || volume < 0 || volume > 1 {
		return errors.New("volume must be between 0 and 1")
	}
	p.player.SetVolume(volume)
	return nil
}

func (p *player) seekBytes(offset int64) error {
	if _, err := p.player.Seek(offset, io.SeekStart); err != nil {
		return fmt.Errorf("seek PCM stream: %w", err)
	}
	return nil
}

func byteOffset(position time.Duration, sampleRate, bytesPerFrame int64) (int64, error) {
	if position < 0 {
		return 0, errors.New("seek position cannot be negative")
	}

	seconds := int64(position / time.Second)
	nanoseconds := int64(position % time.Second)
	if seconds > math.MaxInt64/sampleRate {
		return 0, errors.New("seek position is too large")
	}
	samples := seconds*sampleRate + nanoseconds*sampleRate/int64(time.Second)
	if samples > math.MaxInt64/bytesPerFrame {
		return 0, errors.New("seek position is too large")
	}
	return samples * bytesPerFrame, nil
}

func otoFormat(format audio.Format) oto.Format {
	switch format {
	case audio.FormatFloat32LE:
		return oto.FormatFloat32LE
	case audio.FormatUnsignedInt8:
		return oto.FormatUnsignedInt8
	default:
		return oto.FormatSignedInt16LE
	}
}

func bytesPerSample(format audio.Format) int {
	switch format {
	case audio.FormatFloat32LE:
		return 4
	case audio.FormatUnsignedInt8:
		return 1
	default:
		return 2
	}
}
