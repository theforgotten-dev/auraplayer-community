package codecs

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"

	"playerdoit/internal/audio"
)

type beepFormat uint8

const (
	beepWAV beepFormat = iota
	beepFLAC
	beepVorbis
)

type beepDecoder struct {
	format beepFormat
}

func (decoder beepDecoder) Decode(source io.Reader) (Stream, error) {
	var (
		decoded beep.StreamSeekCloser
		format  beep.Format
		err     error
		name    string
	)
	switch decoder.format {
	case beepWAV:
		name = "WAV"
		decoded, format, err = wav.Decode(source)
	case beepFLAC:
		name = "FLAC"
		decoded, format, err = flac.Decode(source)
	case beepVorbis:
		name = "Ogg Vorbis"
		var seekable io.ReadSeeker
		if candidate, ok := source.(io.ReadSeeker); ok {
			seekable = candidate
		} else {
			encoded, readErr := io.ReadAll(source)
			if readErr != nil {
				return Stream{}, fmt.Errorf("read Ogg Vorbis: %w", readErr)
			}
			seekable = bytes.NewReader(encoded)
		}
		decoded, format, err = vorbis.Decode(readSeekNopCloser{ReadSeeker: seekable})
	default:
		return Stream{}, errors.New("unknown Beep decoder")
	}
	if err != nil {
		return Stream{}, fmt.Errorf("decode %s: %w", name, err)
	}
	if format.SampleRate <= 0 || (format.NumChannels != 1 && format.NumChannels != 2) {
		_ = decoded.Close()
		return Stream{}, fmt.Errorf("decode %s: unsupported PCM layout (%d Hz, %d channels)", name, format.SampleRate, format.NumChannels)
	}

	config := audio.Config{
		SampleRate:   int(format.SampleRate),
		ChannelCount: 2,
		Format:       audio.FormatSignedInt16LE,
	}
	return Stream{
		PCM:      newPCMTask(newBeepPCM(decoded)),
		Config:   config,
		Duration: sampleDuration(decoded.Len(), int(format.SampleRate)),
	}, nil
}

type readSeekNopCloser struct {
	io.ReadSeeker
}

func (readSeekNopCloser) Close() error { return nil }

func sampleDuration(samples, sampleRate int) time.Duration {
	if samples <= 0 || sampleRate <= 0 {
		return 0
	}
	return time.Duration(samples) * time.Second / time.Duration(sampleRate)
}

type beepPCM struct {
	stream   beep.StreamSeekCloser
	frames   [][2]float64
	encoded  []byte
	pending  []byte
	position int64
	drained  bool
}

func newBeepPCM(stream beep.StreamSeekCloser) *beepPCM {
	return &beepPCM{
		stream:  stream,
		frames:  make([][2]float64, 512),
		encoded: make([]byte, 512*4),
	}
}

func (pcm *beepPCM) Read(destination []byte) (int, error) {
	if len(destination) == 0 {
		return 0, nil
	}
	written := 0
	for written < len(destination) {
		if len(pcm.pending) > 0 {
			count := copy(destination[written:], pcm.pending)
			pcm.pending = pcm.pending[count:]
			written += count
			continue
		}
		if pcm.drained {
			break
		}
		count, ok := pcm.stream.Stream(pcm.frames)
		if count > 0 {
			encoded := pcm.encoded[:count*4]
			for index, sample := range pcm.frames[:count] {
				offset := index * 4
				binary.LittleEndian.PutUint16(encoded[offset:offset+2], uint16(floatToS16(sample[0])))
				binary.LittleEndian.PutUint16(encoded[offset+2:offset+4], uint16(floatToS16(sample[1])))
			}
			pcm.pending = encoded
		}
		if !ok {
			pcm.drained = true
			if err := pcm.stream.Err(); err != nil && written == 0 && count == 0 {
				return 0, err
			}
		}
	}
	pcm.position += int64(written)
	if written == 0 && pcm.drained {
		if err := pcm.stream.Err(); err != nil {
			return 0, err
		}
		return 0, io.EOF
	}
	return written, nil
}

func (pcm *beepPCM) Seek(offset int64, whence int) (int64, error) {
	length := int64(pcm.stream.Len()) * 4
	var absolute int64
	switch whence {
	case io.SeekStart:
		absolute = offset
	case io.SeekCurrent:
		absolute = pcm.position + offset
	case io.SeekEnd:
		absolute = length + offset
	default:
		return pcm.position, errors.New("invalid seek origin")
	}
	if absolute < 0 || absolute > length {
		return pcm.position, errors.New("seek position is outside the decoded stream")
	}
	if absolute%4 != 0 {
		return pcm.position, errors.New("seek position must align to a stereo 16-bit PCM frame")
	}
	frame := absolute / 4
	if frame > int64(math.MaxInt) {
		return pcm.position, errors.New("seek position is too large")
	}
	if err := pcm.stream.Seek(int(frame)); err != nil {
		return pcm.position, err
	}
	pcm.position = absolute
	pcm.pending = nil
	pcm.drained = absolute == length
	return absolute, nil
}

func (pcm *beepPCM) Close() error {
	return pcm.stream.Close()
}

func floatToS16(sample float64) int16 {
	if sample <= -1 {
		return math.MinInt16
	}
	if sample >= 1 {
		return math.MaxInt16
	}
	return int16(math.Round(sample * 32767))
}
