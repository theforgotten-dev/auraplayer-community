//go:build windows

package codecs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"git.sr.ht/~jackmordaunt/nativeaudio"

	"playerdoit/internal/audio"
)

const maxCompressedAudioBytes = 512 << 20

type platformDecoder struct{}

func (platformDecoder) Decode(source io.Reader) (Stream, error) {
	compressed, err := io.ReadAll(io.LimitReader(source, maxCompressedAudioBytes+1))
	if err != nil {
		return Stream{}, fmt.Errorf("read compressed audio: %w", err)
	}
	if len(compressed) > maxCompressedAudioBytes {
		return Stream{}, fmt.Errorf("compressed audio exceeds the %d MiB limit", maxCompressedAudioBytes>>20)
	}

	decoder, err := nativeaudio.New(nativeaudio.WithLimits(nativeaudio.Limits{
		MaxBytes:    16 << 30,
		MaxDuration: 2 * time.Minute,
	}))
	if err != nil {
		return Stream{}, fmt.Errorf("initialize Windows Media Foundation: %w", err)
	}
	decoded, err := decoder.Stream(compressed)
	if err != nil {
		return Stream{}, errors.Join(fmt.Errorf("decode with Windows Media Foundation: %w", err), decoder.Close())
	}
	format := decoded.Format()
	if format.BytesPerSample != 2 || (format.Channels != 1 && format.Channels != 2) || format.SampleRate <= 0 {
		return Stream{}, errors.Join(
			fmt.Errorf("unsupported decoded PCM layout (%d Hz, %d channels, %d bytes/sample)", format.SampleRate, format.Channels, format.BytesPerSample),
			decoded.Close(),
			decoder.Close(),
		)
	}

	pcm, err := newSpooledPCM(decoded, decoder, format.Channels*format.BytesPerSample)
	if err != nil {
		return Stream{}, errors.Join(err, decoded.Close(), decoder.Close())
	}
	return Stream{
		PCM: newPCMTask(pcm),
		Config: audio.Config{
			SampleRate:   format.SampleRate,
			ChannelCount: format.Channels,
			Format:       audio.FormatSignedInt16LE,
		},
	}, nil
}

type spooledPCM struct {
	stream       *nativeaudio.Stream
	decoder      *nativeaudio.Decoder
	cache        *os.File
	cachePath    string
	frameBytes   int64
	position     int64
	cached       int64
	terminalErr  error
	decodeBuffer []byte
}

func newSpooledPCM(stream *nativeaudio.Stream, decoder *nativeaudio.Decoder, frameBytes int) (*spooledPCM, error) {
	cache, err := os.CreateTemp("", "aura-pcm-*.cache")
	if err != nil {
		return nil, fmt.Errorf("create PCM seek cache: %w", err)
	}
	return &spooledPCM{
		stream:       stream,
		decoder:      decoder,
		cache:        cache,
		cachePath:    cache.Name(),
		frameBytes:   int64(frameBytes),
		decodeBuffer: make([]byte, 32*1024),
	}, nil
}

func (pcm *spooledPCM) Read(destination []byte) (int, error) {
	if len(destination) == 0 {
		return 0, nil
	}
	written := 0
	if pcm.position < pcm.cached {
		available := min(int64(len(destination)), pcm.cached-pcm.position)
		count, err := pcm.cache.ReadAt(destination[:available], pcm.position)
		pcm.position += int64(count)
		written += count
		if err != nil && !errors.Is(err, io.EOF) {
			return written, fmt.Errorf("read PCM seek cache: %w", err)
		}
	}
	if written == len(destination) {
		return written, nil
	}
	if pcm.terminalErr != nil {
		if written > 0 {
			return written, nil
		}
		return 0, pcm.terminalErr
	}

	count, err := pcm.stream.Read(destination[written:])
	if count > 0 {
		if _, writeErr := pcm.cache.WriteAt(destination[written:written+count], pcm.cached); writeErr != nil {
			return written, fmt.Errorf("write PCM seek cache: %w", writeErr)
		}
		pcm.cached += int64(count)
		pcm.position += int64(count)
		written += count
	}
	if err != nil {
		pcm.terminalErr = err
		if written > 0 {
			return written, nil
		}
		return 0, err
	}
	return written, nil
}

func (pcm *spooledPCM) Seek(offset int64, whence int) (int64, error) {
	var absolute int64
	switch whence {
	case io.SeekStart:
		absolute = offset
	case io.SeekCurrent:
		absolute = pcm.position + offset
	case io.SeekEnd:
		if !errors.Is(pcm.terminalErr, io.EOF) {
			return pcm.position, errors.New("seek from end requires a fully decoded stream")
		}
		absolute = pcm.cached + offset
	default:
		return pcm.position, errors.New("invalid seek origin")
	}
	if absolute < 0 {
		return pcm.position, errors.New("seek position cannot be negative")
	}
	if absolute%pcm.frameBytes != 0 {
		return pcm.position, errors.New("seek position must align to a PCM frame")
	}
	if absolute > pcm.cached {
		if err := pcm.decodeUntil(absolute); err != nil {
			return pcm.position, err
		}
	}
	pcm.position = absolute
	return absolute, nil
}

func (pcm *spooledPCM) decodeUntil(target int64) error {
	for pcm.cached < target {
		remaining := target - pcm.cached
		buffer := pcm.decodeBuffer
		if int64(len(buffer)) > remaining {
			buffer = buffer[:remaining]
		}
		count, err := pcm.stream.Read(buffer)
		if count > 0 {
			if _, writeErr := pcm.cache.WriteAt(buffer[:count], pcm.cached); writeErr != nil {
				return fmt.Errorf("write PCM seek cache: %w", writeErr)
			}
			pcm.cached += int64(count)
		}
		if err != nil {
			pcm.terminalErr = err
			if errors.Is(err, io.EOF) {
				return errors.New("seek position is past the end of the decoded stream")
			}
			return err
		}
		if count == 0 {
			return io.ErrNoProgress
		}
	}
	return nil
}

func (pcm *spooledPCM) Close() error {
	streamErr := pcm.stream.Close()
	decoderErr := pcm.decoder.Close()
	cacheErr := pcm.cache.Close()
	removeErr := os.Remove(pcm.cachePath)
	if errors.Is(removeErr, os.ErrNotExist) {
		removeErr = nil
	}
	return errors.Join(streamErr, decoderErr, cacheErr, removeErr)
}
