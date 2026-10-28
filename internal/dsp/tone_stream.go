package dsp

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"playerdoit/internal/audio"
	"sync"
)

// ToneStream filters stereo S16 PCM with fixed storage and no allocations in Read.
// Read, Seek and Configure are serialized independently of the UI/controller lock.
type ToneStream struct {
	mu                          sync.Mutex
	source                      io.ReadSeeker
	rate                        float64
	config                      ToneConfig
	chain, previous             toneChain
	transition, totalTransition int
	buffer                      [2048]byte
	pending                     []byte
	readErr                     error
}

func NewToneStream(source io.ReadSeeker, format audio.Config, c ToneConfig) (*ToneStream, error) {
	if source == nil {
		return nil, errors.New("nil tone source")
	}
	if err := format.Validate(); err != nil {
		return nil, err
	}
	if format.Format != audio.FormatSignedInt16LE || format.ChannelCount != 2 || format.SampleRate < 44100 {
		return nil, errors.New("tone processing requires stereo S16 PCM at 44100 Hz or above")
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &ToneStream{source: source, rate: float64(format.SampleRate), config: c, chain: makeToneChain(c, float64(format.SampleRate))}, nil
}

func (t *ToneStream) Configure(c ToneConfig) error {
	if err := c.Validate(); err != nil {
		return err
	}
	chain := makeToneChain(c, t.rate)
	t.mu.Lock()
	defer t.mu.Unlock()
	if c == t.config {
		return nil
	}
	t.previous = t.chain
	t.chain = chain
	t.config = c
	t.totalTransition = int(t.rate * .02)
	t.transition = t.totalTransition
	return nil
}

func (t *ToneStream) Read(destination []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(destination) == 0 {
		return 0, nil
	}
	if len(t.pending) == 0 {
		if t.readErr != nil {
			return 0, t.readErr
		}
		n, err := io.ReadFull(t.source, t.buffer[:])
		t.readErr = err
		if n%4 != 0 {
			return 0, io.ErrUnexpectedEOF
		}
		if n == 0 {
			return 0, err
		}
		if err == io.ErrUnexpectedEOF {
			t.readErr = io.EOF
		}
		if t.chain.enabled || t.transition > 0 {
			t.process(t.buffer[:n])
		}
		t.pending = t.buffer[:n]
	}
	n := copy(destination, t.pending)
	t.pending = t.pending[n:]
	return n, nil
}

func (t *ToneStream) process(data []byte) {
	for offset := 0; offset < len(data); offset += 4 {
		mix := 1.0
		if t.transition > 0 {
			mix = 1 - float64(t.transition)/float64(t.totalTransition)
		}
		for ch := 0; ch < 2; ch++ {
			i := offset + 2*ch
			x := float64(int16(binary.LittleEndian.Uint16(data[i:]))) / 32768
			y := t.chain.sample(x, ch)
			if t.transition > 0 {
				y = mix*y + (1-mix)*t.previous.sample(x, ch)
			}
			sample := int16(math.Round(math.Max(-32768, math.Min(32767, y*32768))))
			binary.LittleEndian.PutUint16(data[i:], uint16(sample))
		}
		if t.transition > 0 {
			t.transition--
		}
	}
}

func (t *ToneStream) Seek(offset int64, whence int) (int64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	// The source may be ahead because this wrapper has buffered processed samples.
	if whence == io.SeekCurrent {
		offset -= int64(len(t.pending))
	}
	position, err := t.source.Seek(offset, whence)
	if err != nil {
		return position, err
	}
	t.pending = nil
	t.readErr = nil
	t.transition = 0
	t.chain = makeToneChain(t.config, t.rate)
	return position, nil
}
