package player

import (
	"context"
	"os"
	"playerdoit/internal/dsp"
	"playerdoit/internal/radio"
	"testing"
	"time"
)

func TestTuneRejectsNonHTTP(t *testing.T) {
	c := NewController()
	defer c.Close()
	if err := c.Tune(radio.Station{URL: "file:///tmp/music"}); err == nil {
		t.Fatal("accepted local URL")
	}
	if len(c.Snapshot().Queue) != 0 {
		t.Fatal("modified queue")
	}
}

func TestRadioLiveIntegration(t *testing.T) {
	if os.Getenv("AURA_RADIO_LIVE_TEST") != "1" {
		t.Skip("set AURA_RADIO_LIVE_TEST=1 to query and tune live stations")
	}
	config := radio.DefaultConfig()
	style := "jazz"
	if os.Getenv("AURA_RADIO_PROVIDER") == "icecast" {
		config.Provider = "icecast"
		config.BaseURL = radio.DefaultIcecastURL
		style = ""
	}
	stations, err := (radio.Client{Config: &config}).Search(context.Background(), style)
	if err != nil {
		t.Fatal(err)
	}
	if len(stations) == 0 {
		t.Fatal("empty directory")
	}
	c := NewController()
	defer c.Close()
	c.SetVolume(0)
	if os.Getenv("AURA_TONE_INTEGRATION") == "1" {
		if err := c.ConfigureTone(dsp.ToneConfig{Enabled: true, GainsDB: [10]float64{3, -2, 2, 0, -2}}); err != nil {
			t.Fatal(err)
		}
	}
	played := 0
	for _, s := range stations {
		if err := c.Tune(s); err != nil {
			t.Logf("%s: %v", s.Name, err)
			continue
		}
		deadline := time.Now().Add(6 * time.Second)
		for time.Now().Before(deadline) {
			snap := c.Snapshot()
			if snap.State == StateError {
				break
			}
			if snap.Position > .2 && snap.State == StatePlaying {
				t.Logf("stream advancing: %s (%s), position=%.2f", s.Name, s.Codec, snap.Position)
				played++
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if played == 2 {
			break
		}
	}
	if played < 2 {
		t.Fatalf("only %d stations advanced", played)
	}
	live := 0
	for _, track := range c.Snapshot().Queue {
		if track.Live {
			live++
		}
	}
	if live != 1 {
		t.Fatalf("live slots=%d", live)
	}
}
