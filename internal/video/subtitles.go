package video

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type SubtitleTrack struct {
	ID       int64
	Title    string
	Language string
	Selected bool
}

// ContextSubtitles finds matching sidecars and subtitle files in the video's
// Subs/Subtitles directories, without scanning recursively or changing files.
const maxSubtitleBytes = 16 << 20
const maxContextSubtitles = 32
const maxDirectoryEntries = 4096

func boundedEntries(path string) ([]os.DirEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(maxDirectoryEntries + 1)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(entries) > maxDirectoryEntries {
		return nil, fmt.Errorf("subtitle directory exceeds %d entries", maxDirectoryEntries)
	}
	return entries, nil
}

func ContextSubtitles(path string) ([]string, error) {
	dir := filepath.Dir(path)
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	entries, err := boundedEntries(dir)
	if err != nil {
		return nil, err
	}
	dedicated := true
	for _, e := range entries {
		if !e.IsDir() && IsSupported(filepath.Ext(e.Name())) && !strings.EqualFold(e.Name(), filepath.Base(path)) {
			dedicated = false
		}
	}
	matches := func(name string) bool {
		base := strings.TrimSuffix(name, filepath.Ext(name))
		return strings.EqualFold(base, stem) || strings.HasPrefix(strings.ToLower(base), strings.ToLower(stem)+".")
	}
	languageOnly := func(name string) bool {
		base := strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
		for _, prefix := range []string{"english", "espa\u00f1ol", "espanol", "fran\u00e7ais", "francais"} {
			if base == prefix || strings.HasPrefix(base, prefix+".") || strings.HasPrefix(base, prefix+" (") {
				return true
			}
		}
		return false
	}
	var result []string
	add := func(parent string, e os.DirEntry) {
		if len(result) >= maxContextSubtitles {
			return
		}
		info, err := e.Info()
		if err == nil && info.Mode().IsRegular() && e.Type()&os.ModeSymlink == 0 && info.Size() <= maxSubtitleBytes {
			result = append(result, filepath.Join(parent, e.Name()))
		}
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		if entry.IsDir() && (strings.EqualFold(name, "subs") || strings.EqualFold(name, "subtitles")) {
			children, err := boundedEntries(filepath.Join(dir, name))
			if err != nil {
				continue
			}
			for _, child := range children {
				if !child.IsDir() && child.Type()&os.ModeSymlink == 0 && IsSubtitle(filepath.Ext(child.Name())) {
					if matches(child.Name()) || (dedicated && languageOnly(child.Name())) {
						add(filepath.Join(dir, name), child)
					}
				}
			}
			continue
		}
		if entry.IsDir() || !IsSubtitle(filepath.Ext(name)) {
			continue
		}
		base := strings.TrimSuffix(name, filepath.Ext(name))
		if strings.EqualFold(base, stem) || strings.HasPrefix(strings.ToLower(base), strings.ToLower(stem)+".") {
			add(dir, entry)
		}
	}
	return result, nil
}

func subtitleLanguage(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	code := strings.ToLower(filepath.Ext(base))
	switch code {
	case ".spa", ".es", ".es-419":
		return "spa"
	case ".eng", ".en":
		return "eng"
	case ".fre", ".fra", ".fr":
		return "fra"
	default:
		return strings.TrimPrefix(code, ".")
	}
}
