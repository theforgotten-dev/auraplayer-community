//go:build windows

package video

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"playerdoit/internal/dsp"
)

func TestLibMPVSpectrumIntegration(t *testing.T) {
	if os.Getenv("AURA_TONE_INTEGRATION") != "1" {
		t.Skip("set AURA_TONE_INTEGRATION=1 with AURA_MPV_DLL")
	}
	server := httptest.NewServer(http.FileServer(http.Dir(filepath.Join("..", "codecs", "testdata"))))
	defer server.Close()
	source := server.URL + "/tone.wav"
	if configured := os.Getenv("AURA_SPECTRUM_TEST_URL"); configured != "" {
		source = configured
	}
	player, err := OpenAudio(source)
	if err != nil {
		t.Fatal(err)
	}
	defer player.Close()
	p := player.(*mpvPlayer)
	if err := p.ConfigureTone(dsp.ToneConfig{}); err != nil {
		t.Fatal(err)
	}
	if err := p.SetVolume(0); err != nil {
		t.Fatal(err)
	}
	p.Play()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s := p.Spectrum()
		for _, band := range s.Left {
			if band > .1 {
				t.Logf("real band energy detected: left=%v right=%v", s.Left, s.Right)
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	p.mu.Lock()
	metadata := p.text("af-metadata/aura-spectrum")
	p.mu.Unlock()
	t.Fatalf("no real spectrum detected; metadata=%s", metadata)
}
