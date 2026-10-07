// Package pipeline normalizes decoded PCM before it reaches the audio device.
package pipeline

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/gopxl/beep/v2"

	"playerdoit/internal/audio"
)

const (
	resampleQuality = 4
	streamFrames    = 512
)

// OutputConfig is the stable format used by the process-wide Oto context.
func OutputConfig() audio.Config {
	return audio.Config{
		SampleRate:   48_000,
		ChannelCount: 2,
		Format:       audio.FormatSignedInt16LE,
	}
}

// Normalize converts a seekable PCM source to the target sample rate while it
// is read. The current implementation accepts stereo signed 16-bit PCM, which
// is the format produced by go-mp3.
func Normalize(source io.ReadSeeker, sourceConfig, targetConfig audio.Config) (io.ReadSeeker, error) {
	if source == nil {
		return nil, errors.New("PCM source cannot be nil")
	}
	if err := sourceConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid source PCM configuration: %w", err)
	}
	if err := targetConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid target PCM configuration: %w", err)
	}
	if err := validateStereoS16(sourceConfig); err != nil {
		return nil, fmt.Errorf("source PCM: %w", err)
	}
	if err := validateStereoS16(targetConfig); err != nil {
		return nil, fmt.Errorf("target PCM: %w", err)
	}

	if sourceConfig.SampleRate == targetConfig.SampleRate {
		return source, nil
	}

	stream := &resampledPCM{
		source:     source,
		sourceRate: int64(sourceConfig.SampleRate),
		targetRate: int64(targetConfig.SampleRate),
		frames:     make([][2]float64, streamFrames),
		encoded:    make([]byte, streamFrames*4),
	}
	stream.resetResampler()
	return stream, nil
}

type resampledPCM struct {
	mu sync.Mutex

	source         io.ReadSeeker
	sourceRate     int64
	targetRate     int64
	streamer       *pcm16Streamer
	resampler      *beep.Resampler
	frames         [][2]float64
	encoded        []byte
	pending        []byte
	outputPosition int64
	drained        bool
}

func (r *resampledPCM) Read(destination []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(destination) == 0 {
		return 0, nil
	}

	written := 0
	for written < len(destination) {
		if len(r.pending) > 0 {
			n := copy(destination[written:], r.pending)
			r.pending = r.pending[n:]
			written += n
			continue
		}
		if r.drained {
			break
		}

		frameCount, ok := r.resampler.Stream(r.frames)
		if frameCount == 0 {
			if err := r.resampler.Err(); err != nil {
				if written > 0 {
					r.outputPosition += int64(written)
					return written, nil
				}
				return 0, err
			}
			if !ok {
				r.drained = true
				break
			}
			continue
		}

		encoded := r.encoded[:frameCount*4]
		encodeS16LE(encoded, r.frames[:frameCount])
		r.pending = encoded
	}

	r.outputPosition += int64(written)
	if written == 0 && r.drained {
		return 0, io.EOF
	}
	return written, nil
}

func (r *resampledPCM) Seek(offset int64, whence int) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var absolute int64
	switch whence {
	case io.SeekStart:
		absolute = offset
	case io.SeekCurrent:
		absolute = r.outputPosition + offset
	case io.SeekEnd:
		return r.outputPosition, errors.New("seek from end is not supported by the resampling pipeline")
	default:
		return r.outputPosition, errors.New("invalid seek origin")
	}
	if absolute < 0 {
		return r.outputPosition, errors.New("seek position cannot be negative")
	}
	if absolute%4 != 0 {
		return r.outputPosition, errors.New("seek position must align to a stereo 16-bit PCM frame")
	}

	targetFrame := absolute / 4
	sourceFrame, err := scaleFrames(targetFrame, r.sourceRate, r.targetRate)
	if err != nil {
		return r.outputPosition, err
	}
	if sourceFrame > math.MaxInt64/4 {
		return r.outputPosition, errors.New("seek position is too large")
	}
	if _, err := r.source.Seek(sourceFrame*4, io.SeekStart); err != nil {
		return r.outputPosition, fmt.Errorf("seek source PCM: %w", err)
	}

	r.outputPosition = absolute
	r.pending = nil
	r.drained = false
	r.resetResampler()
	return absolute, nil
}

func (r *resampledPCM) resetResampler() {
	r.streamer = &pcm16Streamer{
		source: r.source,
		buffer: make([]byte, streamFrames*4),
	}
	r.resampler = beep.Resample(
		resampleQuality,
		beep.SampleRate(r.sourceRate),
		beep.SampleRate(r.targetRate),
		r.streamer,
	)
}

type pcm16Streamer struct {
	source io.Reader
	buffer []byte
	err    error
}

func (s *pcm16Streamer) Stream(samples [][2]float64) (int, bool) {
	if s.err != nil || len(samples) == 0 {
		return 0, false
	}

	needed := len(samples) * 4
	if cap(s.buffer) < needed {
		s.buffer = make([]byte, needed)
	}
	buffer := s.buffer[:needed]
	read, err := io.ReadFull(s.source, buffer)
	if read%4 != 0 {
		s.err = io.ErrUnexpectedEOF
		return 0, false
	}
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		s.err = err
		return 0, false
	}

	frames := read / 4
	for index := 0; index < frames; index++ {
		offset := index * 4
		// #nosec G115 -- signed PCM uses the same 16 bits in two's-complement form.
		left := int16(binary.LittleEndian.Uint16(buffer[offset : offset+2]))
		// #nosec G115 -- signed PCM uses the same 16 bits in two's-complement form.
		right := int16(binary.LittleEndian.Uint16(buffer[offset+2 : offset+4]))
		samples[index][0] = float64(left) / 32768
		samples[index][1] = float64(right) / 32768
	}
	return frames, frames > 0
}

func (s *pcm16Streamer) Err() error { return s.err }

func encodeS16LE(destination []byte, samples [][2]float64) {
	for index, sample := range samples {
		offset := index * 4
		// #nosec G115 -- encoding S16LE requires the two's-complement bit pattern.
		binary.LittleEndian.PutUint16(destination[offset:offset+2], uint16(floatToInt16(sample[0])))
		// #nosec G115 -- encoding S16LE requires the two's-complement bit pattern.
		binary.LittleEndian.PutUint16(destination[offset+2:offset+4], uint16(floatToInt16(sample[1])))
	}
}

func floatToInt16(sample float64) int16 {
	if sample <= -1 {
		return math.MinInt16
	}
	if sample >= 1 {
		return math.MaxInt16
	}
	return int16(math.Round(sample * 32767))
}

func scaleFrames(frames, numerator, denominator int64) (int64, error) {
	if frames < 0 || numerator <= 0 || denominator <= 0 {
		return 0, errors.New("invalid frame conversion")
	}
	whole := frames / denominator
	remainder := frames % denominator
	if whole > math.MaxInt64/numerator || remainder > math.MaxInt64/numerator {
		return 0, errors.New("seek position is too large")
	}
	return whole*numerator + remainder*numerator/denominator, nil
}

func validateStereoS16(config audio.Config) error {
	if config.ChannelCount != 2 {
		return errors.New("only stereo PCM is supported")
	}
	if config.Format != audio.FormatSignedInt16LE {
		return errors.New("only signed 16-bit little-endian PCM is supported")
	}
	return nil
}
