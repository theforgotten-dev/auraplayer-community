//go:build windows

package video

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExtendedAudioIntegration(t *testing.T) {
	dir := os.Getenv("AURA_AUDIO_TEST_DIR")
	if dir == "" {
		t.Skip("set AURA_AUDIO_TEST_DIR for real audio decoder tests")
	}
	paths, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no audio fixtures found")
	}
	for _, path := range paths {
		t.Run(filepath.Ext(path)+"-"+filepath.Base(path), func(t *testing.T) {
			p, err := OpenAudio(path)
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()
			rate, channels := p.AudioFormat()
			t.Logf("audio format before play: %d Hz, %d channels", rate, channels)
			if p.Duration() < time.Second {
				t.Fatalf("duration=%v", p.Duration())
			}
			if err := p.SetVolume(0); err != nil {
				t.Fatal(err)
			}
			p.Play()
			deadline := time.Now().Add(4 * time.Second)
			for p.Position() < 100*time.Millisecond {
				if err := p.Err(); err != nil {
					t.Fatal(err)
				}
				if time.Now().After(deadline) {
					t.Fatal("audio did not advance")
				}
				time.Sleep(10 * time.Millisecond)
			}
			p.Pause()
			paused := p.Position()
			time.Sleep(50 * time.Millisecond)
			if p.Position()-paused > 80*time.Millisecond {
				t.Fatal("audio advanced while paused")
			}
			if p.Frame() != nil {
				t.Fatal("audio-only playback allocated a video frame")
			}
			if err := p.Seek(time.Second); err != nil {
				t.Fatal(err)
			}
			p.Play()
			deadline = time.Now().Add(3 * time.Second)
			for p.Position() < time.Second {
				if time.Now().After(deadline) {
					t.Fatal("audio seek failed")
				}
				time.Sleep(10 * time.Millisecond)
			}
			if err := p.Stop(); err != nil {
				t.Fatal(err)
			}
			if p.IsPlaying() {
				t.Fatal("audio kept playing after Stop")
			}
		})
	}
}
