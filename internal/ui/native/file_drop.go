package native

import (
	"fmt"
	"path/filepath"
	"strings"

	"playerdoit/internal/player"
)

const maxDroppedFiles = 256

func acceptedDroppedAudio(paths []string) ([]string, error) {
	if len(paths) > maxDroppedFiles {
		return nil, fmt.Errorf("a drop can contain at most %d files", maxDroppedFiles)
	}

	accepted := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if !player.IsSupported(filepath.Ext(path)) {
			continue
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve dropped file %q: %w", path, err)
		}
		key := strings.ToLower(filepath.Clean(absolute))
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		accepted = append(accepted, absolute)
	}
	if len(accepted) == 0 {
		return nil, fmt.Errorf("drop one or more supported audio or video files")
	}
	return accepted, nil
}
