package pipeline

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"testing"

	"playerdoit/internal/audio"
)

func TestNormalizePassesThroughMatchingPCM(t *testing.T) {
	source := bytes.NewReader(makePCM(48_000))
	config := OutputConfig()
	normalized, err := Normalize(source, config, config)
	if err != nil {
		t.Fatal(err)
	}
	if normalized != source {
		t.Fatal("matching PCM was wrapped instead of passed through")
	}
}

func TestNormalizeResamplesToTargetRate(t *testing.T) {
	const sourceRate = 44_100
	source := bytes.NewReader(makePCM(sourceRate))
	sourceConfig := audio.Config{SampleRate: sourceRate, ChannelCount: 2, Format: audio.FormatSignedInt16LE}
	normalized, err := Normalize(source, sourceConfig, OutputConfig())
	if err != nil {
		t.Fatal(err)
	}

	output, err := io.ReadAll(normalized)
	if err != nil {
		t.Fatal(err)
	}
	gotFrames := len(output) / 4
	if difference := abs(gotFrames - 48_000); difference > 2 {
		t.Fatalf("resampled frame count = %d, want approximately 48000", gotFrames)
	}
}

func TestResampledSeekMapsOutputTimeToSource(t *testing.T) {
	const sourceRate = 44_100
	source := bytes.NewReader(makePCM(sourceRate * 2))
	sourceConfig := audio.Config{SampleRate: sourceRate, ChannelCount: 2, Format: audio.FormatSignedInt16LE}
	normalized, err := Normalize(source, sourceConfig, OutputConfig())
	if err != nil {
		t.Fatal(err)
	}

	const oneSecondAt48K = int64(48_000 * 4)
	position, err := normalized.Seek(oneSecondAt48K, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}
	if position != oneSecondAt48K {
		t.Fatalf("Seek() = %d, want %d", position, oneSecondAt48K)
	}
	sourcePosition, err := source.Seek(0, io.SeekCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(sourceRate * 4); sourcePosition != want {
		t.Fatalf("source position = %d, want %d", sourcePosition, want)
	}
}

func TestNormalizeDuplicatesMonoIntoStereo(t *testing.T) {
	mono := make([]byte, 200)
	for offset := 0; offset < len(mono); offset += 2 {
		binary.LittleEndian.PutUint16(mono[offset:offset+2], uint16(int16(12_345)))
	}
	config := audio.Config{SampleRate: 48_000, ChannelCount: 1, Format: audio.FormatSignedInt16LE}
	normalized, err := Normalize(bytes.NewReader(mono), config, OutputConfig())
	if err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(normalized)
	if err != nil {
		t.Fatal(err)
	}
	if len(output) == 0 || len(output)%4 != 0 {
		t.Fatalf("normalized mono produced %d bytes", len(output))
	}
	for offset := 0; offset < len(output); offset += 4 {
		if !bytes.Equal(output[offset:offset+2], output[offset+2:offset+4]) {
			t.Fatalf("frame %d differs between channels", offset/4)
		}
	}
}

func TestNormalizeRejectsMultichannelPCM(t *testing.T) {
	config := audio.Config{SampleRate: 44_100, ChannelCount: 3, Format: audio.FormatSignedInt16LE}
	if _, err := Normalize(bytes.NewReader(nil), config, OutputConfig()); err == nil {
		t.Fatal("Normalize() accepted three-channel PCM")
	}
}

func makePCM(frames int) []byte {
	data := make([]byte, frames*4)
	for frame := 0; frame < frames; frame++ {
		sample := int16(math.Sin(2*math.Pi*440*float64(frame)/44_100) * 12_000)
		offset := frame * 4
		binary.LittleEndian.PutUint16(data[offset:offset+2], uint16(sample))
		binary.LittleEndian.PutUint16(data[offset+2:offset+4], uint16(sample))
	}
	return data
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
