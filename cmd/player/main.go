package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"playerdoit/internal/audio"
	"playerdoit/internal/audio/otoengine"
	"playerdoit/internal/codecs"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "playerdoit:", err)
		os.Exit(1)
	}
}

func run() error {
	config := audio.DefaultConfig()
	formatName := "s16le"
	flag.IntVar(&config.SampleRate, "rate", config.SampleRate, "raw PCM sample rate in Hz")
	flag.IntVar(&config.ChannelCount, "channels", config.ChannelCount, "raw PCM channels: 1 or 2")
	flag.StringVar(&formatName, "format", formatName, "raw PCM format: s16le, u8 or f32le")
	flag.Parse()

	if flag.NArg() != 1 {
		return errors.New("usage: playerdoit [flags] <audio.mp3|audio.pcm>")
	}

	path := flag.Arg(0)
	// #nosec G304 -- the CLI is explicitly designed to open its user-supplied path.
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open audio file: %w", err)
	}
	defer file.Close()

	var pcm io.ReadSeeker = file
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		stream, err := codecs.Decode(".mp3", file)
		if err != nil {
			return err
		}
		pcm = stream.PCM
		config = stream.Config
	case ".pcm", ".raw":
		format, err := parseFormat(formatName)
		if err != nil {
			return err
		}
		config.Format = format
	default:
		return fmt.Errorf("unsupported audio file %q: expected .mp3, .pcm or .raw", path)
	}

	engine, err := otoengine.New(config)
	if err != nil {
		return err
	}
	player := engine.NewPlayer(pcm)
	defer player.Close()

	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stopped)

	player.Play()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stopped:
			return player.Stop()
		case <-ticker.C:
			if err := player.Err(); err != nil {
				return fmt.Errorf("play PCM stream: %w", err)
			}
			if err := engine.Err(); err != nil {
				return fmt.Errorf("audio device: %w", err)
			}
			if !player.IsPlaying() {
				return nil
			}
		}
	}
}

func parseFormat(value string) (audio.Format, error) {
	switch strings.ToLower(value) {
	case "s16le":
		return audio.FormatSignedInt16LE, nil
	case "u8":
		return audio.FormatUnsignedInt8, nil
	case "f32le":
		return audio.FormatFloat32LE, nil
	default:
		return 0, fmt.Errorf("unsupported PCM format %q", value)
	}
}
