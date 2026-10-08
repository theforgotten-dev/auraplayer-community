package player

import (
	"path/filepath"
	"testing"
)

func TestInspectTrackReadsSupportedFormatMetadata(t *testing.T) {
	tests := []struct {
		file  string
		codec string
	}{
		{file: "tone.wav", codec: "WAV"},
		{file: "tone.flac", codec: "FLAC"},
		{file: "tone.ogg", codec: "OGG/VORBIS"},
	}
	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			track, err := inspectTrack(filepath.Join("..", "codecs", "testdata", test.file))
			if err != nil {
				t.Fatal(err)
			}
			if track.Codec != test.codec || track.SampleRate != 44_100 || track.Channels != 1 || track.Duration <= 0 {
				t.Fatalf("track = %+v", track)
			}
		})
	}
}

func TestInspectTrackRejectsUnsupportedExtension(t *testing.T) {
	if _, err := inspectTrack("cover.png"); err == nil {
		t.Fatal("inspectTrack accepted an unsupported extension")
	}
}
