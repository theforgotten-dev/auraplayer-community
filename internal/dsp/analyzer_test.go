package dsp

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"sync"
	"testing"

	"playerdoit/internal/audio"
)

func TestAnalyzerPassesPCMThroughAndSeparatesChannels(t *testing.T) {
	pcm := stereoTones(48_000, fftSize*2, 1000, 4000)
	analyzer, err := NewAnalyzer(bytes.NewReader(pcm), testConfig())
	if err != nil {
		t.Fatal(err)
	}
	actual, err := io.ReadAll(analyzer)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, pcm) {
		t.Fatal("analyzer changed PCM bytes")
	}

	spectrum := analyzer.Snapshot()
	leftBand := peakBand(spectrum.Left)
	rightBand := peakBand(spectrum.Right)
	if frequency := spectrum.Frequencies[leftBand]; frequency < 800 || frequency > 1300 {
		t.Fatalf("left peak frequency = %.0f Hz, want about 1 kHz", frequency)
	}
	if frequency := spectrum.Frequencies[rightBand]; frequency < 3200 || frequency > 5100 {
		t.Fatalf("right peak frequency = %.0f Hz, want about 4 kHz", frequency)
	}
	if spectrum.Left[leftBand] <= spectrum.Right[leftBand]+0.25 {
		t.Fatal("1 kHz band is not isolated in the left channel")
	}
	if spectrum.Right[rightBand] <= spectrum.Left[rightBand]+0.25 {
		t.Fatal("4 kHz band is not isolated in the right channel")
	}
}

func TestAnalyzerHandlesUnalignedReadsAndResetsOnSeek(t *testing.T) {
	pcm := stereoTones(48_000, fftSize, 440, 880)
	analyzer, err := NewAnalyzer(bytes.NewReader(pcm), testConfig())
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 7)
	for {
		_, readErr := analyzer.Read(buffer)
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			t.Fatal(readErr)
		}
	}
	if analyzer.Snapshot().LeftLevel == 0 {
		t.Fatal("expected a measured signal")
	}
	position, err := analyzer.Seek(0, io.SeekStart)
	if err != nil || position != 0 {
		t.Fatalf("Seek() = (%d, %v), want (0, nil)", position, err)
	}
	reset := analyzer.Snapshot()
	if reset.LeftLevel != 0 || reset.RightLevel != 0 {
		t.Fatal("seek did not reset channel levels")
	}
	for index := range reset.Left {
		if reset.Left[index] != 0 || reset.Right[index] != 0 {
			t.Fatalf("seek did not reset band %d", index)
		}
	}
}

func stereoTones(sampleRate, frames int, leftFrequency, rightFrequency float64) []byte {
	pcm := make([]byte, frames*4)
	for frame := 0; frame < frames; frame++ {
		time := float64(frame) / float64(sampleRate)
		left := int16(math.Sin(2*math.Pi*leftFrequency*time) * 26_000)
		right := int16(math.Sin(2*math.Pi*rightFrequency*time) * 26_000)
		binary.LittleEndian.PutUint16(pcm[frame*4:frame*4+2], uint16(left))
		binary.LittleEndian.PutUint16(pcm[frame*4+2:frame*4+4], uint16(right))
	}
	return pcm
}

func peakBand(values [bandCount]float64) int {
	peak := 0
	for index := 1; index < len(values); index++ {
		if values[index] > values[peak] {
			peak = index
		}
	}
	return peak
}

func testConfig() audio.Config {
	return audio.Config{SampleRate: 48_000, ChannelCount: 2, Format: audio.FormatSignedInt16LE}
}

func BenchmarkAnalyzerRead(b *testing.B) {
	pcm := stereoTones(48_000, 48_000, 1000, 4000)
	analyzer, err := NewAnalyzer(bytes.NewReader(pcm), testConfig())
	if err != nil {
		b.Fatal(err)
	}
	buffer := make([]byte, 4096)
	b.ReportAllocs()
	b.SetBytes(int64(len(pcm)))
	b.ResetTimer()
	for b.Loop() {
		if _, err := analyzer.Seek(0, io.SeekStart); err != nil {
			b.Fatal(err)
		}
		for {
			if _, err := analyzer.Read(buffer); err == io.EOF {
				break
			} else if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func TestAnalyzerConcurrentReadSeekAndSnapshot(t *testing.T) {
	analyzer, err := NewAnalyzer(bytes.NewReader(stereoTones(48_000, fftSize*4, 1000, 4000)), testConfig())
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	workers.Add(3)
	go func() {
		defer workers.Done()
		buffer := make([]byte, 4093)
		for i := 0; i < 200; i++ {
			if _, err := analyzer.Read(buffer); err != nil && err != io.EOF {
				t.Error(err)
				return
			}
		}
	}()
	go func() {
		defer workers.Done()
		for i := 0; i < 200; i++ {
			if _, err := analyzer.Seek(0, io.SeekStart); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	go func() {
		defer workers.Done()
		for i := 0; i < 200; i++ {
			spectrum := analyzer.Snapshot()
			for band := range spectrum.Left {
				if spectrum.Left[band] < 0 || spectrum.Left[band] > 1 || math.IsNaN(spectrum.Left[band]) {
					t.Errorf("invalid published spectrum: %v", spectrum.Left)
					return
				}
			}
		}
	}()
	workers.Wait()
}
