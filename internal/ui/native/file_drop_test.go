package native

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestAcceptedDroppedAudioFiltersAndDeduplicates(t *testing.T) {
	paths, err := acceptedDroppedAudio([]string{"first.MP3", "notes.txt", "first.MP3", "second.flac", "third.M4A"})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 || filepath.Base(paths[0]) != "first.MP3" || filepath.Base(paths[1]) != "second.flac" || filepath.Base(paths[2]) != "third.M4A" {
		t.Fatalf("acceptedDroppedAudio() = %v", paths)
	}
}

func TestAcceptedDroppedAudioRejectsUnsupportedDrop(t *testing.T) {
	if _, err := acceptedDroppedAudio([]string{"cover.png", "notes.txt"}); err == nil {
		t.Fatal("expected a drop without audio files to be rejected")
	}
}

func TestAcceptedDroppedAudioLimitsBatchSize(t *testing.T) {
	paths := make([]string, maxDroppedFiles+1)
	for index := range paths {
		paths[index] = fmt.Sprintf("track-%d.mp3", index)
	}
	_, err := acceptedDroppedAudio(paths)
	if err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("error = %v, want batch size limit", err)
	}
}
