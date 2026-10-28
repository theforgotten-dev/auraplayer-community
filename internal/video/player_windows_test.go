//go:build windows

package video

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInvalidContextSubtitleDoesNotAbortVideo(t *testing.T) {
	fixture := os.Getenv("AURA_VIDEO_TEST_FILE")
	if fixture == "" {
		t.Skip("set AURA_VIDEO_TEST_FILE")
	}
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "movie.mp4")
	for name, content := range map[string][]byte{"movie.mp4": data, "movie.srt": []byte("invalid subtitle"), "movie.eng.srt": []byte("1\n00:00:00,000 --> 00:00:05,000\nValid subtitle\n")} {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	p, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if len(p.SubtitleTracks()) != 1 {
		t.Fatalf("valid subtitle not preserved: %v", p.SubtitleTracks())
	}
}

// Set AURA_VIDEO_TEST_FILE and AURA_MPV_DLL to exercise the real decoder,
// renderer and audio output. The fixture should be at least six seconds long.
func TestPlaybackIntegration(t *testing.T) {
	path := os.Getenv("AURA_VIDEO_TEST_FILE")
	if path == "" {
		t.Skip("set AURA_VIDEO_TEST_FILE for real video playback")
	}
	p, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if p.Duration() < 6*time.Second {
		t.Fatalf("duration = %v, need >= 6s", p.Duration())
	}
	if err := p.SetVolume(.15); err != nil {
		t.Fatal(err)
	}
	p.Play()
	deadline := time.Now().Add(5 * time.Second)
	for p.Position() < 300*time.Millisecond || p.Frame() == nil {
		if err := p.Err(); err != nil {
			t.Fatal(err)
		}
		if time.Now().After(deadline) {
			t.Fatal("video did not produce a frame and advance")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !p.IsPlaying() {
		t.Fatalf("not playing: %v", p.Err())
	}
	first := p.Frame()
	time.Sleep(150 * time.Millisecond)
	if first == p.Frame() {
		t.Fatal("rendered frames did not update")
	}
	p.Pause()
	paused := p.Position()
	time.Sleep(200 * time.Millisecond)
	if delta := p.Position() - paused; delta > 100*time.Millisecond {
		t.Fatalf("position advanced while paused: %v", delta)
	}
	if p.IsPlaying() {
		t.Fatal("still playing after pause")
	}
	// Compare paused rendered frames with subtitles hidden and visible. This
	// checks the real composited picture, not just a successful API command.
	subs := filepath.Join(t.TempDir(), "subtítulos de prueba.srt")
	if err := os.WriteFile(subs, []byte("1\n00:00:00,000 --> 00:00:08,000\nAURA SUBTITLE TEST\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := p.LoadSubtitles(subs); err != nil {
		t.Fatal(err)
	}
	if !p.SubtitlesVisible() {
		t.Fatal("loaded subtitles are hidden")
	}
	if p.(*mpvPlayer).number("sid") < 1 {
		t.Fatal("external subtitle track was not selected")
	}
	time.Sleep(250 * time.Millisecond)
	withSubs := p.Frame()
	if err := p.SetSubtitlesVisible(false); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(3 * time.Second)
	for p.Frame() == withSubs {
		if time.Now().After(deadline) {
			t.Fatal("subtitle visibility did not refresh paused frame")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if bytes.Equal(withSubs.Pix, p.Frame().Pix) {
		t.Fatal("subtitles did not change rendered pixels")
	}
	if p.SubtitlesVisible() {
		t.Fatal("subtitles still visible")
	}
	if err := p.LoadSubtitles(filepath.Join(t.TempDir(), "missing.srt")); err == nil {
		t.Fatal("accepted missing subtitles")
	}
	if err := p.LoadSubtitles(path); err == nil {
		t.Fatal("accepted video as subtitle file")
	}
	if err := p.Seek(3 * time.Second); err != nil {
		t.Fatal(err)
	}
	p.Play()
	deadline = time.Now().Add(3 * time.Second)
	for p.Position() < 3*time.Second {
		if time.Now().After(deadline) {
			t.Fatalf("seek position = %v", p.Position())
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := p.Stop(); err != nil {
		t.Fatal(err)
	}
	if p.IsPlaying() {
		t.Fatal("still playing after stop")
	}
	p.Play()
	deadline = time.Now().Add(3 * time.Second)
	for p.Position() < 100*time.Millisecond || p.Position() > 2*time.Second {
		if time.Now().After(deadline) {
			t.Fatalf("restart position = %v", p.Position())
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := p.Seek(p.Duration() - 200*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(4 * time.Second)
	for p.IsPlaying() {
		if time.Now().After(deadline) {
			t.Fatal("playback did not end")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := p.Err(); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAutoSubtitlesIntegration(t *testing.T) {
	path := os.Getenv("AURA_VIDEO_TEST_FILE")
	if path == "" {
		t.Skip("set AURA_VIDEO_TEST_FILE for real subtitle playback")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	fixture := filepath.Join(dir, "auto.mp4")
	if err := os.WriteFile(fixture, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "auto.srt"), []byte("1\n00:00:00,000 --> 00:00:08,000\nAUTOMATIC SUBTITLES\n"), 0600); err != nil {
		t.Fatal(err)
	}
	p, err := Open(fixture)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	p.Play()
	deadline := time.Now().Add(3 * time.Second)
	for p.Position() < 300*time.Millisecond || p.Frame() == nil {
		if time.Now().After(deadline) {
			t.Fatal("automatic subtitle fixture did not start")
		}
		time.Sleep(20 * time.Millisecond)
	}
	p.Pause()
	time.Sleep(200 * time.Millisecond)
	if p.(*mpvPlayer).number("sid") < 1 {
		t.Fatal("matching SRT track was not automatically selected")
	}
	shown := p.Frame()
	if err := p.SetSubtitlesVisible(false); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(3 * time.Second)
	for p.Frame() == shown {
		if time.Now().After(deadline) {
			t.Fatal("auto subtitle visibility did not redraw")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if bytes.Equal(shown.Pix, p.Frame().Pix) {
		t.Fatal("matching SRT was not auto-loaded and rendered")
	}
}
