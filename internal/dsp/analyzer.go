// Package dsp contains real-time signal analysis for decoded PCM streams.
package dsp

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"sync"

	"gonum.org/v1/gonum/dsp/fourier"

	"playerdoit/internal/audio"
)

const (
	fftSize   = 2048
	hopSize   = 512
	minDB     = -72.0
	frameSize = 4
	bandCount = 16
)

var bandEdges = [...]float64{
	40, 63, 100, 160, 250, 400, 630, 1000, 1600,
	2500, 4000, 6300, 8000, 10000, 12500, 16000, 20000,
}

// Spectrum is a normalized view of the current stereo signal. Band values
// range from 0 (at or below -72 dBFS) to 1 (0 dBFS).
type Spectrum struct {
	Frequencies [bandCount]float64 `json:"frequencies"`
	Left        [bandCount]float64 `json:"left"`
	Right       [bandCount]float64 `json:"right"`
	LeftLevel   float64            `json:"leftLevel"`
	RightLevel  float64            `json:"rightLevel"`
}

// Analyzer passes PCM through unchanged while calculating an FFT for each
// stereo channel. It is safe to query while the audio engine is reading it.
type Analyzer struct {
	source   io.ReadSeeker
	config   audio.Config
	sourceMu sync.Mutex
	mu       sync.RWMutex

	fft               *fourier.FFT
	window            []float64
	scale             float64
	sampleBufferLeft  []float64
	sampleBufferRight []float64
	leftSamples       []float64
	rightSamples      []float64
	leftCoeffs        []complex128
	rightCoeffs       []complex128
	writeIndex        int
	samplesSinceFFT   int
	totalSamples      int
	carry             []byte
	spectrum          Spectrum
}

// NewAnalyzer wraps stereo signed 16-bit little-endian PCM.
func NewAnalyzer(source io.ReadSeeker, config audio.Config) (*Analyzer, error) {
	if source == nil {
		return nil, errors.New("PCM source cannot be nil")
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if config.ChannelCount != 2 || config.Format != audio.FormatSignedInt16LE {
		return nil, errors.New("spectrum analyzer requires stereo signed 16-bit little-endian PCM")
	}

	analyzer := &Analyzer{
		source:            source,
		config:            config,
		fft:               fourier.NewFFT(fftSize),
		window:            make([]float64, fftSize),
		sampleBufferLeft:  make([]float64, fftSize),
		sampleBufferRight: make([]float64, fftSize),
		leftSamples:       make([]float64, fftSize),
		rightSamples:      make([]float64, fftSize),
		leftCoeffs:        make([]complex128, fftSize/2+1),
		rightCoeffs:       make([]complex128, fftSize/2+1),
		spectrum:          EmptySpectrum(),
		carry:             make([]byte, 0, frameSize-1),
	}
	windowSum := 0.0
	for i := range analyzer.window {
		factor := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(fftSize-1))
		analyzer.window[i] = factor
		windowSum += factor
	}
	analyzer.scale = 2 / windowSum
	return analyzer, nil
}

// EmptySpectrum returns the stable band layout used by the UI.
func EmptySpectrum() Spectrum {
	var spectrum Spectrum
	for index := range spectrum.Frequencies {
		spectrum.Frequencies[index] = math.Sqrt(bandEdges[index] * bandEdges[index+1])
	}
	return spectrum
}

func (a *Analyzer) Read(destination []byte) (int, error) {
	a.sourceMu.Lock()
	read, err := a.source.Read(destination)
	a.sourceMu.Unlock()
	if read > 0 {
		a.mu.Lock()
		a.consume(destination[:read])
		a.mu.Unlock()
	}
	return read, err
}

func (a *Analyzer) Seek(offset int64, whence int) (int64, error) {
	a.sourceMu.Lock()
	position, err := a.source.Seek(offset, whence)
	a.sourceMu.Unlock()
	if err != nil {
		return position, err
	}
	a.mu.Lock()
	a.reset()
	a.mu.Unlock()
	return position, nil
}

// Snapshot returns a copy so callers can serialize it without holding the
// analyzer lock or racing the audio thread.
func (a *Analyzer) Snapshot() Spectrum {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.spectrum
}

func (a *Analyzer) consume(data []byte) {
	if len(a.carry) > 0 {
		needed := frameSize - len(a.carry)
		if needed > len(data) {
			a.carry = append(a.carry, data...)
			return
		}
		a.carry = append(a.carry, data[:needed]...)
		a.consumeFrame(a.carry)
		a.carry = a.carry[:0]
		data = data[needed:]
	}

	complete := len(data) - len(data)%frameSize
	for offset := 0; offset < complete; offset += frameSize {
		a.consumeFrame(data[offset : offset+frameSize])
	}
	a.carry = append(a.carry, data[complete:]...)
}

func (a *Analyzer) consumeFrame(frame []byte) {
	// #nosec G115 -- signed PCM uses the same 16 bits in two's-complement form.
	left := float64(int16(binary.LittleEndian.Uint16(frame[0:2]))) / 32768
	// #nosec G115 -- signed PCM uses the same 16 bits in two's-complement form.
	right := float64(int16(binary.LittleEndian.Uint16(frame[2:4]))) / 32768

	a.sampleBufferLeft[a.writeIndex] = left
	a.sampleBufferRight[a.writeIndex] = right
	a.writeIndex = (a.writeIndex + 1) % fftSize
	a.samplesSinceFFT++
	a.totalSamples++

	if a.totalSamples >= fftSize && a.samplesSinceFFT >= hopSize {
		a.calculate()
		a.samplesSinceFFT = 0
	}
}

func (a *Analyzer) calculate() {
	leftRMS, rightRMS := 0.0, 0.0
	for index := 0; index < fftSize; index++ {
		bufferIdx := (a.writeIndex + index) % fftSize
		left := a.sampleBufferLeft[bufferIdx]
		right := a.sampleBufferRight[bufferIdx]
		leftRMS += left * left
		rightRMS += right * right
		factor := a.window[index]
		a.leftSamples[index] = left * factor
		a.rightSamples[index] = right * factor
	}
	a.leftCoeffs = a.fft.Coefficients(a.leftCoeffs, a.leftSamples)
	a.rightCoeffs = a.fft.Coefficients(a.rightCoeffs, a.rightSamples)

	binWidth := float64(a.config.SampleRate) / fftSize
	scale := a.scale
	for band := range a.spectrum.Left {
		first := max(1, int(math.Ceil(bandEdges[band]/binWidth)))
		last := min(len(a.leftCoeffs)-1, int(math.Floor(bandEdges[band+1]/binWidth)))
		if last < first {
			last = first
		}
		leftPeak, rightPeak := 0.0, 0.0
		for bin := first; bin <= last; bin++ {
			leftPeak = math.Max(leftPeak, cmplxAbs(a.leftCoeffs[bin])*scale)
			rightPeak = math.Max(rightPeak, cmplxAbs(a.rightCoeffs[bin])*scale)
		}
		a.spectrum.Left[band] = smooth(a.spectrum.Left[band], normalizeDB(leftPeak))
		a.spectrum.Right[band] = smooth(a.spectrum.Right[band], normalizeDB(rightPeak))
	}
	a.spectrum.LeftLevel = smooth(a.spectrum.LeftLevel, normalizeDB(math.Sqrt(leftRMS/fftSize)))
	a.spectrum.RightLevel = smooth(a.spectrum.RightLevel, normalizeDB(math.Sqrt(rightRMS/fftSize)))
}

func (a *Analyzer) reset() {
	clear(a.sampleBufferLeft)
	clear(a.sampleBufferRight)
	clear(a.leftSamples)
	clear(a.rightSamples)
	a.writeIndex = 0
	a.samplesSinceFFT = 0
	a.totalSamples = 0
	a.carry = a.carry[:0]
	a.spectrum = EmptySpectrum()
}

func normalizeDB(amplitude float64) float64 {
	if amplitude <= 0 {
		return 0
	}
	value := (20*math.Log10(amplitude) - minDB) / -minDB
	return math.Max(0, math.Min(1, value))
}

func smooth(previous, current float64) float64 {
	if current > previous {
		return previous*0.10 + current*0.90
	}
	return previous*0.80 + current*0.20
}

func cmplxAbs(value complex128) float64 {
	return math.Hypot(real(value), imag(value))
}
