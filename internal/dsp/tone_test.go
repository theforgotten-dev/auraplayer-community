package dsp

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"playerdoit/internal/audio/pipeline"
	"testing"
)

func tonePCM(hz float64, frames int) []byte {
	data := make([]byte, frames*4)
	for i := 0; i < frames; i++ {
		v := int16(4000 * math.Sin(2*math.Pi*hz*float64(i)/48000))
		binary.LittleEndian.PutUint16(data[i*4:], uint16(v))
	}
	return data
}
func filteredPCM(t *testing.T, c ToneConfig, hz float64) []byte {
	t.Helper()
	stream, err := NewToneStream(bytes.NewReader(tonePCM(hz, 48000)), pipeline.OutputConfig(), c)
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(stream)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func level(data []byte) float64 {
	sum := 0.0
	count := 0
	for i := 4800 * 4; i < len(data); i += 4 {
		x := float64(int16(binary.LittleEndian.Uint16(data[i:])))
		sum += x * x
		count++
	}
	return math.Sqrt(sum / float64(count))
}

func TestToneBypassAndFlatAreByteExact(t *testing.T) {
	input := tonePCM(1000, 2307)
	for _, c := range []ToneConfig{{}, {Enabled: true}, {GainsDB: [10]float64{12, 12, 12, 12, 12}, PreampDB: 12}} {
		stream, err := NewToneStream(bytes.NewReader(input), pipeline.OutputConfig(), c)
		if err != nil {
			t.Fatal(err)
		}
		var output []byte
		buffer := make([]byte, 7)
		for {
			n, err := stream.Read(buffer)
			output = append(output, buffer[:n]...)
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		if !bytes.Equal(output, input) {
			t.Fatal("flat or bypass modified samples")
		}
	}
}

func TestToneMeasuredFrequencyResponseAndChannelIsolation(t *testing.T) {
	tests := []struct {
		name         string
		c            ToneConfig
		hz, min, max float64
	}{
		{"peak at center", ToneConfig{Enabled: true, GainsDB: [10]float64{5: 6}}, 1000, .93, .96},
		{"peak outside band", ToneConfig{Enabled: true, GainsDB: [10]float64{5: 6}}, 60, .46, .49},
		{"highpass rejects low", ToneConfig{Enabled: true, GainsDB: [10]float64{-12, -12, -12}}, 20, 0, .025},
		{"highpass preserves high", ToneConfig{Enabled: true, GainsDB: [10]float64{-12, -12, -12}}, 1000, .85, 1.02},
		{"lowpass rejects high", ToneConfig{Enabled: true, GainsDB: [10]float64{7: -12, 8: -12, 9: -12}}, 16000, 0, .035},
		{"lowpass preserves low", ToneConfig{Enabled: true, GainsDB: [10]float64{7: -12, 8: -12, 9: -12}}, 1000, .85, 1.02},
		{"preamp minus six", ToneConfig{Enabled: true, PreampDB: -6}, 1000, .49, .52},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out := filteredPCM(t, test.c, test.hz)
			ratio := level(out) / level(tonePCM(test.hz, 48000))
			if ratio < test.min || ratio > test.max {
				t.Fatalf("amplitude ratio %.5f, expected %.3f..%.3f", ratio, test.min, test.max)
			}
			for i := 2; i < len(out); i += 4 {
				if binary.LittleEndian.Uint16(out[i:]) != 0 {
					t.Fatal("signal leaked into right channel")
				}
			}
		})
	}
}

func TestToneSeekResetsHistoryAndBufferedOutput(t *testing.T) {
	source := bytes.NewReader(tonePCM(60, 48000))
	c := ToneConfig{Enabled: true, GainsDB: [10]float64{-12, -12, -12}}
	stream, _ := NewToneStream(source, pipeline.OutputConfig(), c)
	first := make([]byte, 300)
	io.ReadFull(stream, first)
	if position, err := stream.Seek(0, io.SeekCurrent); err != nil || position != 300 {
		t.Fatalf("position=%d error=%v", position, err)
	}
	if _, err := stream.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	again := make([]byte, 300)
	io.ReadFull(stream, again)
	if !bytes.Equal(first, again) {
		t.Fatal("seek retained filter history")
	}
}

func TestToneValidationAndConfigure(t *testing.T) {
	for _, c := range []ToneConfig{{PreampDB: math.NaN()}, {GainsDB: [10]float64{math.Inf(1)}}, {GainsDB: [10]float64{13}}, {GainsDB: [10]float64{-13}}} {
		if c.Validate() == nil {
			t.Fatal("invalid config accepted")
		}
	}
	s, _ := NewToneStream(bytes.NewReader(tonePCM(1000, 48000)), pipeline.OutputConfig(), ToneConfig{})
	io.CopyN(io.Discard, s, 2048)
	if err := s.Configure(ToneConfig{Enabled: true, PreampDB: -6}); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(s)
	if err != nil {
		t.Fatal(err)
	}
	ratio := level(out) / level(tonePCM(1000, len(out)/4))
	if ratio < .49 || ratio > .52 {
		t.Fatal(ratio)
	}
	if err := s.Configure(ToneConfig{PreampDB: 99}); err == nil {
		t.Fatal("invalid update accepted")
	}
}

func BenchmarkToneRead(b *testing.B) {
	for _, enabled := range []bool{false, true} {
		name := "bypass"
		if enabled {
			name = "ten-bands"
		}
		b.Run(name, func(b *testing.B) {
			source := bytes.NewReader(tonePCM(1000, 48000))
			s, _ := NewToneStream(source, pipeline.OutputConfig(), ToneConfig{Enabled: enabled, GainsDB: [10]float64{-12, -10, -8, 3, -2, 2, -1, -8, -10, -12}})
			buffer := make([]byte, 4096)
			b.ReportAllocs()
			b.SetBytes(48000 * 4)
			b.ResetTimer()
			for b.Loop() {
				s.Seek(0, io.SeekStart)
				for {
					_, err := s.Read(buffer)
					if err == io.EOF {
						break
					}
					if err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}
