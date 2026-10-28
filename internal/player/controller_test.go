package player

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectVideoAndMixedQueue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.MP4")
	if err := os.WriteFile(path, []byte("container validated when opened"), 0600); err != nil {
		t.Fatal(err)
	}
	c := NewController()
	defer c.Close()
	wav := filepath.Join("..", "codecs", "testdata", "tone.wav")
	if err := c.AddFiles([]string{wav, path, path}); err != nil {
		t.Fatal(err)
	}
	s := c.Snapshot()
	if len(s.Queue) != 2 || s.Queue[0].Video || !s.Queue[1].Video || s.Queue[1].Codec != "MP4" {
		t.Fatalf("queue = %+v", s.Queue)
	}
	if _, err := inspectTrack(filepath.Join(t.TempDir(), "missing.mkv")); err == nil {
		t.Fatal("accepted missing video")
	}
}

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

func TestExtendedAudioQueueAndPlaybackIntegration(t *testing.T) {
	dir := os.Getenv("AURA_AUDIO_TEST_DIR")
	if dir == "" {
		t.Skip("set AURA_AUDIO_TEST_DIR for mixed audio queue")
	}
	c := NewController()
	defer c.Close()
	if err := c.SetVolume(0); err != nil {
		t.Fatal(err)
	}
	paths := []string{filepath.Join(dir, "test.opus"), filepath.Join(dir, "test.ogg"), filepath.Join("..", "codecs", "testdata", "tone.wav"), filepath.Join(dir, "surround.wav"), filepath.Join(dir, "test.m4a")}
	if err := c.AddFiles(paths); err != nil {
		t.Fatal(err)
	}
	for i := range paths {
		if err := c.Play(i); err != nil {
			t.Fatalf("track %d: %v", i, err)
		}
		s := c.Snapshot()
		if s.State != StatePlaying || s.CurrentIndex != i {
			t.Fatalf("snapshot=%+v", s)
		}
		if s.Queue[i].Video {
			t.Fatal("audio appeared as video")
		}
		if i == 3 && s.Queue[i].Channels != 6 {
			t.Fatalf("surround channels=%d", s.Queue[i].Channels)
		}
		if err := c.Toggle(); err != nil {
			t.Fatal(err)
		}
		if err := c.Seek(.1); err != nil {
			t.Fatal(err)
		}
	}
}
