package native

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const maxSkinSize = 256 * 1024

// Skin is the portable visual manifest used by the native UI.
type Skin struct {
	Version int               `json:"version"`
	Name    string            `json:"name"`
	Author  string            `json:"author,omitempty"`
	Colors  map[string]string `json:"colors,omitempty"`
	Metrics SkinMetrics       `json:"metrics,omitempty"`
}

type SkinMetrics struct {
	Spacing      float32 `json:"spacing,omitempty"`
	CornerRadius float32 `json:"cornerRadius,omitempty"`
	StrokeWidth  float32 `json:"strokeWidth,omitempty"`
}

type Palette struct {
	Background   color.NRGBA
	Panel        color.NRGBA
	PanelRaised  color.NRGBA
	Line         color.NRGBA
	LineHot      color.NRGBA
	Text         color.NRGBA
	Muted        color.NRGBA
	Accent       color.NRGBA
	ChannelLeft  color.NRGBA
	ChannelRight color.NRGBA
	Danger       color.NRGBA
}

var builtinSkins = []Skin{
	{
		Version: 1, Name: "PHOSPHOR_GREEN", Author: "playerdoit",
		Colors: map[string]string{
			"background": "#070a09", "panel": "#080e0b", "panelRaised": "#0b1510",
			"line": "#1d4b38", "lineHot": "#39d98a", "text": "#c9f7dd",
			"muted": "#668c77", "accent": "#53f29c", "channelLeft": "#53f29c",
			"channelRight": "#5ee8e2", "danger": "#ff6b81",
		},
		Metrics: SkinMetrics{Spacing: 8, StrokeWidth: 1},
	},
	{
		Version: 1, Name: "AMBER_CRT", Author: "playerdoit",
		Colors: map[string]string{
			"background": "#0c0903", "panel": "#140d04", "panelRaised": "#1b1206",
			"line": "#624617", "lineHot": "#ffb52e", "text": "#ffe5a8",
			"muted": "#9d7740", "accent": "#ffb52e", "channelLeft": "#ff9d24",
			"channelRight": "#ffe17a", "danger": "#ff664d",
		},
		Metrics: SkinMetrics{Spacing: 8, StrokeWidth: 1},
	},
	{
		Version: 1, Name: "CYBER_VIOLET", Author: "playerdoit",
		Colors: map[string]string{
			"background": "#070512", "panel": "#0d081c", "panelRaised": "#130b28",
			"line": "#483080", "lineHot": "#bb74ff", "text": "#eee4ff",
			"muted": "#8973aa", "accent": "#bb74ff", "channelLeft": "#bb74ff",
			"channelRight": "#48e8ff", "danger": "#ff4f91",
		},
		Metrics: SkinMetrics{Spacing: 8, StrokeWidth: 1},
	},
}

func loadSkin(reader io.Reader) (Skin, error) {
	if reader == nil {
		return Skin{}, fmt.Errorf("skin source cannot be nil")
	}
	contents, err := io.ReadAll(io.LimitReader(reader, maxSkinSize+1))
	if err != nil {
		return Skin{}, fmt.Errorf("read skin: %w", err)
	}
	if len(contents) > maxSkinSize {
		return Skin{}, fmt.Errorf("skin exceeds the %d KiB limit", maxSkinSize/1024)
	}

	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var skin Skin
	if err := decoder.Decode(&skin); err != nil {
		return Skin{}, fmt.Errorf("decode skin: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Skin{}, fmt.Errorf("skin must contain exactly one JSON object")
		}
		return Skin{}, fmt.Errorf("decode trailing skin data: %w", err)
	}
	if skin.Version != 1 {
		return Skin{}, fmt.Errorf("unsupported skin version %d", skin.Version)
	}
	if strings.TrimSpace(skin.Name) == "" || len(skin.Name) > 64 || strings.IndexFunc(skin.Name, unicode.IsControl) >= 0 {
		return Skin{}, fmt.Errorf("skin name must contain 1 to 64 characters")
	}
	if len(skin.Author) > 128 || strings.IndexFunc(skin.Author, unicode.IsControl) >= 0 {
		return Skin{}, fmt.Errorf("skin author must contain at most 128 characters without control characters")
	}
	if skin.Metrics.Spacing < 0 || skin.Metrics.Spacing > 32 || skin.Metrics.CornerRadius < 0 || skin.Metrics.CornerRadius > 32 || skin.Metrics.StrokeWidth < 0 || skin.Metrics.StrokeWidth > 4 {
		return Skin{}, fmt.Errorf("skin metrics are outside their supported range")
	}
	for name, value := range skin.Colors {
		if !knownColor(name) {
			return Skin{}, fmt.Errorf("unknown skin color %q", name)
		}
		if _, err := parseColor(value); err != nil {
			return Skin{}, fmt.Errorf("skin color %q: %w", name, err)
		}
	}
	return skin, nil
}

func paletteFor(skin Skin) Palette {
	base := builtinSkins[0]
	merged := make(map[string]string, len(base.Colors)+len(skin.Colors))
	for name, value := range base.Colors {
		merged[name] = value
	}
	for name, value := range skin.Colors {
		merged[name] = value
	}
	get := func(name string) color.NRGBA {
		parsed, _ := parseColor(merged[name])
		return parsed
	}
	return Palette{
		Background: get("background"), Panel: get("panel"), PanelRaised: get("panelRaised"),
		Line: get("line"), LineHot: get("lineHot"), Text: get("text"), Muted: get("muted"),
		Accent: get("accent"), ChannelLeft: get("channelLeft"), ChannelRight: get("channelRight"), Danger: get("danger"),
	}
}

func parseColor(value string) (color.NRGBA, error) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(value) != 6 && len(value) != 8 {
		return color.NRGBA{}, fmt.Errorf("expected #RRGGBB or #RRGGBBAA")
	}
	bytes, err := hex.DecodeString(value)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("invalid hexadecimal color")
	}
	result := color.NRGBA{R: bytes[0], G: bytes[1], B: bytes[2], A: 255}
	if len(bytes) == 4 {
		result.A = bytes[3]
	}
	return result, nil
}

func knownColor(name string) bool {
	switch name {
	case "background", "panel", "panelRaised", "line", "lineHot", "text", "muted", "accent", "channelLeft", "channelRight", "danger":
		return true
	default:
		return false
	}
}

func persistedSkin() Skin {
	path, err := skinConfigPath()
	if err != nil {
		return builtinSkins[0]
	}
	// #nosec G304 -- path is derived solely from os.UserConfigDir and a fixed suffix.
	file, err := os.Open(path)
	if err != nil {
		return builtinSkins[0]
	}
	defer file.Close()
	skin, err := loadSkin(file)
	if err != nil {
		return builtinSkins[0]
	}
	return skin
}

func persistSkin(skin Skin) error {
	path, err := skinConfigPath()
	if err != nil {
		return err
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	// #nosec G302 -- directories require the execute bit; 0700 keeps access private.
	if err := os.Chmod(directory, 0o700); err != nil {
		return err
	}
	contents, err := json.MarshalIndent(skin, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, contents, 0o600)
}

func skinConfigPath() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "playerdoit", "skin.json"), nil
}
