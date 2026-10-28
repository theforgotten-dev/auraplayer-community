package player

import (
	"os"
	"path/filepath"
	"playerdoit/internal/dsp"
	"testing"
	"time"
)

func TestTonePCMPlaybackIntegration(t *testing.T) {
	if os.Getenv("AURA_TONE_INTEGRATION") != "1" {
		t.Skip("set AURA_TONE_INTEGRATION=1 for real audio engine")
	}
	c := NewController()
	defer c.Close()
	c.SetVolume(0)
	config := dsp.ToneConfig{Enabled: true, GainsDB: [10]float64{3, -2, 2, 1, -3}}
	if err := c.ConfigureTone(config); err != nil {
		t.Fatal(err)
	}
	if err := c.AddFiles([]string{filepath.Join("..", "codecs", "testdata", "tone.wav")}); err != nil {
		t.Fatal(err)
	}
	if err := c.Play(0); err != nil {
		t.Fatal(err)
	}
	if c.toneStream == nil {
		t.Fatal("PCM chain missing tone processor")
	}
	deadline := time.Now().Add(4 * time.Second)
	for c.Snapshot().Position < .1 {
		if time.Now().After(deadline) {
			t.Fatal("PCM did not advance")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := c.ConfigureTone(dsp.ToneConfig{}); err != nil {
		t.Fatal(err)
	}
	if err := c.Seek(0); err != nil {
		t.Fatal(err)
	}
	if err := c.ConfigureTone(config); err != nil {
		t.Fatal(err)
	}
	if err := c.Stop(); err != nil {
		t.Fatal(err)
	}
}
