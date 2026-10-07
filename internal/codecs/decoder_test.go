package codecs

import (
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
	if _, err := DecoderFor(".wav"); err == nil {
		t.Fatal("DecoderFor(.wav) unexpectedly succeeded")
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
