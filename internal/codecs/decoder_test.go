package codecs

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDecoderForMP3IsCaseInsensitive(t *testing.T) {
	if _, err := DecoderFor(".MP3"); err != nil {
		t.Fatalf("DecoderFor(.MP3): %v", err)
	}
}

func TestPCMDuration(t *testing.T) {
	if got, want := pcmDuration(264_600, 44_100, 4), 1500*time.Millisecond; got != want {
		t.Fatalf("pcmDuration() = %v, want %v", got, want)
	}
}

func TestDecoderForRejectsUnsupportedFormat(t *testing.T) {
	if _, err := DecoderFor(".opus"); err == nil {
		t.Fatal("DecoderFor(.opus) unexpectedly succeeded")
	}
}

func TestSupportedExtensions(t *testing.T) {
	want := []string{".mp3", ".wav", ".flac", ".ogg", ".aac", ".m4a", ".wma"}
	if got := SupportedExtensions(); !reflect.DeepEqual(got, want) {
		t.Fatalf("SupportedExtensions() = %v, want %v", got, want)
	}
	for _, extension := range want {
		if !IsSupported(strings.ToUpper(extension)) {
			t.Fatalf("IsSupported(%q) = false", extension)
		}
		if _, err := DecoderFor(extension); err != nil {
			t.Fatalf("DecoderFor(%q): %v", extension, err)
		}
	}
}

func TestBeepFormatsDecodeToSeekableStereoPCM(t *testing.T) {
	tests := []struct {
		extension string
		file      string
	}{
		{extension: ".wav", file: "tone.wav"},
		{extension: ".flac", file: "tone.flac"},
		{extension: ".ogg", file: "tone.ogg"},
	}
	for _, test := range tests {
		t.Run(test.extension, func(t *testing.T) {
			file, err := os.Open(filepath.Join("testdata", test.file))
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			stream, err := Decode(test.extension, file)
			if err != nil {
				t.Fatal(err)
			}
			defer stream.PCM.(io.Closer).Close()
			if stream.Config.SampleRate != 44_100 || stream.Config.ChannelCount != 2 {
				t.Fatalf("stream = %+v", stream)
			}
			if test.extension != ".ogg" && stream.Duration != 500*time.Millisecond {
				t.Fatalf("duration = %v, want 500ms", stream.Duration)
			}
			buffer := make([]byte, 64)
			if count, err := stream.PCM.Read(buffer); err != nil || count == 0 {
				t.Fatalf("Read() = (%d, %v)", count, err)
			}
			if position, err := stream.PCM.Seek(0, io.SeekStart); err != nil || position != 0 {
				t.Fatalf("Seek() = (%d, %v)", position, err)
			}
		})
	}
}

func TestMP3DecoderRejectsInvalidStream(t *testing.T) {
	decoder, err := DecoderFor(".mp3")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decoder.Decode(strings.NewReader("not an mp3")); err == nil {
		t.Fatal("Decode() unexpectedly accepted invalid MP3 data")
	}
}

func TestMP3DecoderContainsInitializationPanic(t *testing.T) {
	decoder, err := DecoderFor(".mp3")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decoder.Decode(panicReader{}); err == nil || !strings.Contains(err.Error(), "panic") {
		t.Fatalf("Decode() error = %v, want recovered initialization panic", err)
	}
}

type panicReader struct{}

func (panicReader) Read([]byte) (int, error) { panic("malformed input") }
