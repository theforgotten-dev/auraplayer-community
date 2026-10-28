//go:build windows

package video

import (
	"bytes"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"
)

// This test opens the actual user-selected movie in place and mutes playback.
func TestMovieContextIntegration(t *testing.T) {
	path := os.Getenv("AURA_CONTEXT_TEST_FILE")
	if path == "" {
		t.Skip("set AURA_CONTEXT_TEST_FILE for contextual movie subtitles")
	}
	paths, err := ContextSubtitles(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 4 {
		t.Fatalf("found %d subtitle files, want 4", len(paths))
	}
	p, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	tracks := p.SubtitleTracks()
	selected := SubtitleTrack{}
	for _, track := range tracks {
		if track.Selected {
			selected = track
		}
	}
	t.Logf("detected files=%d; loaded tracks=%d; selected=%s (%s)", len(paths), len(tracks), selected.Title, selected.Language)
	if len(tracks) != 4 {
		t.Fatalf("loaded %d subtitle tracks, want 4", len(tracks))
	}
	if selected.Language != "spa" {
		t.Fatalf("selected language=%q, want spa", selected.Language)
	}
	if err := p.SetVolume(0); err != nil {
		t.Fatal(err)
	}
	var subtitlePath string
	for _, path := range paths {
		if subtitleLanguage(path) == "spa" {
			subtitlePath = path
			break
		}
	}
	data, err := os.ReadFile(subtitlePath)
	if err != nil {
		t.Fatal(err)
	}
	matches := regexp.MustCompile(`(?m)(\d{2}):(\d{2}):(\d{2}),(\d{3}) --> (\d{2}):(\d{2}):(\d{2}),(\d{3})`).FindSubmatch(data)
	if matches == nil {
		t.Fatal("Spanish SRT contains no valid cue")
	}
	stamp := func(offset int) time.Duration {
		var n [4]int
		for i := range n {
			n[i], _ = strconv.Atoi(string(matches[offset+i]))
		}
		return time.Duration(n[0])*time.Hour + time.Duration(n[1])*time.Minute + time.Duration(n[2])*time.Second + time.Duration(n[3])*time.Millisecond
	}
	target := (stamp(1) + stamp(5)) / 2
	if err := p.Seek(target); err != nil {
		t.Fatal(err)
	}
	p.Play()
	deadline := time.Now().Add(5 * time.Second)
	for p.Position() < target || p.Frame() == nil {
		if time.Now().After(deadline) {
			t.Fatal("movie did not reach subtitle cue")
		}
		time.Sleep(10 * time.Millisecond)
	}
	p.Pause()
	time.Sleep(200 * time.Millisecond)
	shown := p.Frame()
	if err := p.SetSubtitlesVisible(false); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(3 * time.Second)
	for p.Frame() == shown {
		if time.Now().After(deadline) {
			t.Fatal("subtitle toggle did not redraw movie")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if bytes.Equal(shown.Pix, p.Frame().Pix) {
		t.Fatal("Spanish subtitles were selected but did not render")
	}
	for _, track := range tracks {
		if track.Language == "fra" {
			if err := p.SelectSubtitle(track.ID); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	selected = SubtitleTrack{}
	for _, track := range p.SubtitleTracks() {
		if track.Selected {
			selected = track
		}
	}
	if selected.Language != "fra" {
		t.Fatal("could not switch contextual subtitle track")
	}
	t.Log("Spanish subtitles rendered over the actual movie; switching to French also passed")
}
