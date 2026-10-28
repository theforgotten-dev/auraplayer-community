//go:build windows

package video

import (
	"os"
	"path/filepath"
	"playerdoit/internal/dsp"
	"strings"
	"testing"
	"time"
)

func TestToneLibMPVIntegration(t *testing.T) {
	if os.Getenv("AURA_TONE_INTEGRATION") != "1" {
		t.Skip("set AURA_TONE_INTEGRATION=1 with AURA_MPV_DLL")
	}
	player, err := OpenAudio(filepath.Join("..", "codecs", "testdata", "tone.wav"))
	if err != nil {
		t.Fatal(err)
	}
	defer player.Close()
	p := player.(*mpvPlayer)
	p.SetVolume(0)
	for _, c := range []dsp.ToneConfig{
		{Enabled: true, PreampDB: -3, GainsDB: [10]float64{-12, -10, -8, 3, -2, 2, -1, -8, -10, -12}},
		{},
	} {
		if err := p.ConfigureTone(c); err != nil {
			t.Fatal(err)
		}
		p.mu.Lock()
		active := p.text("af")
		p.mu.Unlock()
		t.Logf("active filter: %s", active)
		if c.Enabled && !strings.Contains(active, "equalizer") {
			t.Fatal("equalizer not installed")
		}
		if !c.Enabled && strings.Contains(active, "equalizer") {
			t.Fatal("bypass left equalizer active")
		}
		if err := p.Seek(0); err != nil {
			t.Fatal(err)
		}
		p.Play()
		deadline := time.Now().Add(4 * time.Second)
		for p.Position() < 100*time.Millisecond {
			if err := p.Err(); err != nil {
				t.Fatal(err)
			}
			if time.Now().After(deadline) {
				t.Fatal("filtered audio did not advance")
			}
			time.Sleep(10 * time.Millisecond)
		}
		p.Pause()
	}
}
