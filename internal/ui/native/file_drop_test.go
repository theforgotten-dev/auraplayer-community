package native

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestAcceptedDroppedMP3FiltersAndDeduplicates(t *testing.T) {
	paths, err := acceptedDroppedMP3([]string{"first.MP3", "notes.txt", "first.MP3", "second.mp3"})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 || filepath.Base(paths[0]) != "first.MP3" || filepath.Base(paths[1]) != "second.mp3" {
		t.Fatalf("acceptedDroppedMP3() = %v", paths)
	}
}

func TestAcceptedDroppedMP3RejectsUnsupportedDrop(t *testing.T) {
	if _, err := acceptedDroppedMP3([]string{"cover.png", "notes.txt"}); err == nil {
		t.Fatal("expected a drop without MP3 files to be rejected")
	}
}

func TestAcceptedDroppedMP3LimitsBatchSize(t *testing.T) {
	paths := make([]string, maxDroppedFiles+1)
	for index := range paths {
		paths[index] = fmt.Sprintf("track-%d.mp3", index)
	}
	_, err := acceptedDroppedMP3(paths)
	if err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("error = %v, want batch size limit", err)
	}
}
