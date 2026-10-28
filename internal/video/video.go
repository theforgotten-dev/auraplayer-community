// Package video provides synchronized video and audio playback for Aura.
package video

import (
	"image"
	"slices"
	"strings"
	"time"

	"playerdoit/internal/audio"
)

var extensions = []string{".mp4", ".mkv", ".webm", ".avi", ".mov", ".m4v", ".mpg", ".mpeg", ".wmv"}

// These containers use libmpv's audio decoder rather than the PCM adapters.
var audioExtensions = []string{
	".aac", ".m4a", ".m4b", ".m4r", ".wma", ".opus", ".oga", ".aif", ".aiff", ".aifc",
	".ape", ".wv", ".tta", ".alac", ".ac3", ".eac3", ".dts", ".dtshd",
	".mka", ".mp1", ".mp2", ".mpa", ".mpga", ".amr", ".awb", ".3ga", ".caf", ".au", ".snd",
	".w64", ".rf64", ".wave", ".bwf", ".adts", ".weba", ".spx", ".dsf", ".dff", ".tak", ".mpc", ".mpp", ".mp+",
	".ra", ".8svx", ".voc", ".mod", ".xm", ".it", ".s3m", ".mtm", ".umx",
}

func AudioExtensions() []string { return slices.Clone(audioExtensions) }
func IsAudioSupported(extension string) bool {
	return slices.Contains(audioExtensions, strings.ToLower(extension))
}

func SupportedExtensions() []string { return slices.Clone(extensions) }
func IsSupported(extension string) bool {
	return slices.Contains(extensions, strings.ToLower(extension))
}

// Player owns both media streams so seeking and pausing preserve synchronization.
type Player interface {
	audio.Player
	Position() time.Duration
	Duration() time.Duration
	Frame() *image.RGBA
	LoadSubtitles(path string) error
	SetSubtitlesVisible(visible bool) error
	SubtitlesVisible() bool
	SubtitleTracks() []SubtitleTrack
	SelectSubtitle(id int64) error
	AudioFormat() (sampleRate int, channels int)
}

var subtitleExtensions = []string{".srt", ".ass", ".ssa", ".vtt"}

func SubtitleExtensions() []string { return slices.Clone(subtitleExtensions) }
func IsSubtitle(extension string) bool {
	return slices.Contains(subtitleExtensions, strings.ToLower(extension))
}
