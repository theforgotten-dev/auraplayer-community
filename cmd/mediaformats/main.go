// Command mediaformats prints the desktop file extensions for packaging.
package main

import (
	"encoding/json"
	"os"
	"slices"

	"playerdoit/internal/codecs"
	"playerdoit/internal/video"
)

func main() {
	audio := codecs.SupportedExtensions()
	for _, ext := range video.AudioExtensions() {
		if !slices.Contains(audio, ext) {
			audio = append(audio, ext)
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(struct{ Audio, Video []string }{audio, video.SupportedExtensions()}); err != nil {
		panic(err)
	}
}
