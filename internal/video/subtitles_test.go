package video

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestContextSubtitles(t *testing.T) {
	dir := t.TempDir()
	for _, subdir := range []string{"Subs", "other"} {
		if err := os.Mkdir(filepath.Join(dir, subdir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"movie.srt", "movie.es.vtt", "unrelated.srt", "movie2.srt", "Subs/Espa\u00f1ol.spa.srt", "Subs/English.eng.srt", "Subs/cover.jpg", "other/hidden.srt"} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(path)), []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	paths, err := ContextSubtitles(filepath.Join(dir, "movie.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 4 {
		t.Fatalf("contextual subtitles = %v", paths)
	}
	for _, path := range paths {
		if filepath.Base(path) == "unrelated.srt" || filepath.Base(path) == "movie2.srt" {
			t.Fatal("associated subtitles with another video")
		}
	}
}

func TestSharedSubtitleFolder(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "Subs"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"movie.mp4", "other.mkv", "Subs/movie.spa.srt", "Subs/different.spa.srt", "Subs/English.eng.srt"} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	paths, err := ContextSubtitles(filepath.Join(dir, "movie.mp4"))
	if err != nil || len(paths) != 1 || filepath.Base(paths[0]) != "movie.spa.srt" {
		t.Fatalf("unexpected subtitles %v: %v", paths, err)
	}
}

func TestSubtitleSizeAndCountLimits(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < maxContextSubtitles+5; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("movie.%02d.srt", i)), []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	f, err := os.Create(filepath.Join(dir, "movie.big.srt"))
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate(maxSubtitleBytes + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	paths, err := ContextSubtitles(filepath.Join(dir, "movie.mp4"))
	if err != nil || len(paths) != maxContextSubtitles {
		t.Fatalf("limits: %d %v", len(paths), err)
	}
	for _, path := range paths {
		if filepath.Base(path) == "movie.big.srt" {
			t.Fatal("oversized subtitle accepted")
		}
	}
}
