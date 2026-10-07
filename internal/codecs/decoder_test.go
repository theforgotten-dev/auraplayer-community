package codecs

import (
	"strings"
	"testing"
)

func TestDecoderForMP3IsCaseInsensitive(t *testing.T) {
	if _, err := DecoderFor(".MP3"); err != nil {
		t.Fatalf("DecoderFor(.MP3): %v", err)
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
