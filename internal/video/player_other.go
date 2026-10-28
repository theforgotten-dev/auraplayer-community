//go:build !windows

package video

import "errors"

func Open(path string) (Player, error) {
	return nil, errors.New("video playback currently requires Windows and libmpv-2.dll")
}

func OpenAudio(path string) (Player, error) {
	return nil, errors.New("extended audio playback currently requires Windows and libmpv-2.dll")
}
