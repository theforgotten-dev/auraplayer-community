package native

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"gioui.org/app"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget/material"
	"playerdoit/internal/dsp"
	"playerdoit/internal/player"
)

func TestPreferencesRoundTripAndReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings", "preferences.json")
	p := defaultPreferences()
	if err := writePreferences(path, p); err != nil {
		t.Fatal(err)
	}
	p.Styles = map[string]int{"jazz": 3}
	p.Tone = dsp.ToneConfig{Enabled: true, PreampDB: -3, GainsDB: [10]float64{2, -1, 3, 0, 2}}
	p.Radio.BaseURL = "https://example.org/radio"
	p.Radio.Styles = []string{"jazz", "news"}
	p.Radio.Country = "CL"
	p.Radio.Limit = 10
	p.Radio.HideBroken = false
	p.Radio.Personalized = false
	p.Volume = .35
	p.Muted = true
	p.Language = "eng"
	p.HideDelay = 5
	if err := writePreferences(path, p); err != nil {
		t.Fatal(err)
	}
	got, err := readPreferences(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, p) {
		t.Fatalf("preferences=%+v, want %+v", got, p)
	}
	for _, data := range []string{`{"version":1,"volume":2}`, `{"version":1} {}`, `{"version":1,"unknown":true}`} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := readPreferences(path); err == nil {
			t.Fatalf("accepted invalid preferences: %s", data)
		}
	}
}

func TestLegacyPreferencesReceiveRadioDefaults(t *testing.T) {
	p := defaultPreferences()
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var old map[string]any
	json.Unmarshal(data, &old)
	delete(old, "radio")
	data, err = json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "preferences.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readPreferences(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Radio, p.Radio) {
		t.Fatalf("missing defaults: %+v", got.Radio)
	}
}

func TestRadioDraftDoesNotMutateSavedPreferences(t *testing.T) {
	p := defaultPreferences()
	p.Styles = map[string]int{"jazz": 2}
	ui := &UI{preferences: p}
	ui.editPreferences(p)
	ui.settings.draft.Styles["jazz"] = 9
	ui.settings.draft.Radio.Styles[0] = "news"
	ui.radioSettings.source.SetText("https://example.org/directory/")
	ui.radioSettings.country.SetText("cl")
	ui.radioSettings.styles.SetText("Jazz, News")
	c := ui.radioSettingsConfig()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.Country != "CL" || c.Styles[0] != "jazz" {
		t.Fatal(c)
	}
	if p.Styles["jazz"] != 2 || p.Radio.Styles[0] != "rock" || p.Radio.BaseURL == c.BaseURL {
		t.Fatal("draft changed saved preferences")
	}
	ui.editPreferences(defaultPreferences())
	if ui.radioSettingsConfig().BaseURL != defaultPreferences().Radio.BaseURL {
		t.Fatal("reset did not restore editors")
	}
}

func TestRadioSettingsLayoutWithSingleSuggestedStyle(t *testing.T) {
	ui := &UI{theme: material.NewTheme(), palette: paletteFor(builtinSkins[0])}
	p := defaultPreferences()
	p.Radio.Styles = []string{"jazz"}
	ui.editPreferences(p)
	for _, tab := range []int{4, 5} {
		ui.settings.tab = tab
		gtx := layout.Context{Ops: new(op.Ops), Now: time.Now(), Constraints: layout.Exact(image.Pt(540, 180))}
		dims := ui.layoutSettingsBody(gtx)
		if dims.Size.X > 540 || dims.Size.Y > 180 {
			t.Fatalf("tab %d exceeds body: %v", tab, dims.Size)
		}
	}
}

func TestToneEditorPresetsAndLayout(t *testing.T) {
	ui := &UI{theme: material.NewTheme(), palette: paletteFor(builtinSkins[0])}
	p := defaultPreferences()
	ui.editPreferences(p)
	if c := ui.toneConfig(); c != p.Tone {
		t.Fatalf("default editor changed tone: %+v", c)
	}
	wanted := dsp.ToneConfig{Enabled: true, PreampDB: -6, GainsDB: [10]float64{3, -2, 0, 2, 1}}
	ui.editTone(wanted)
	if c := ui.toneConfig(); c != wanted {
		t.Fatalf("editor roundtrip=%+v, expected %+v", c, wanted)
	}
	for _, size := range []image.Point{image.Pt(540, 180), image.Pt(540, 120)} {
		gtx := layout.Context{Ops: new(op.Ops), Now: time.Now(), Constraints: layout.Exact(size)}
		dims := ui.layoutTone(gtx)
		if dims.Size.X > size.X || dims.Size.Y > size.Y {
			t.Fatalf("tone panel exceeds viewport: %v", dims)
		}
	}
}

func TestPreferencesShortcutAndEscapePriority(t *testing.T) {
	ui := &UI{window: new(app.Window), controller: player.NewController(), preferences: defaultPreferences(), fullscreen: true}
	defer ui.controller.Close()
	var router input.Router
	gtx := layout.Context{Ops: new(op.Ops), Source: router.Source(), Now: time.Now()}
	ui.updateWindowShortcuts(gtx)
	router.Queue(key.Event{Name: "P", Modifiers: key.ModCtrl, State: key.Press})
	ui.updateWindowShortcuts(gtx)
	if !ui.settings.open {
		t.Fatal("Ctrl+P did not open preferences")
	}
	ui.settings.draft.Language = "fra"
	ui.openSettings()
	if ui.settings.draft.Language != "fra" {
		t.Fatal("reopening discarded unsaved edits")
	}
	router.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	ui.updateWindowShortcuts(gtx)
	if ui.settings.open || !ui.fullscreen {
		t.Fatal("Escape must close preferences before exiting fullscreen")
	}
	router.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	ui.updateWindowShortcuts(gtx)
	if ui.fullscreen {
		t.Fatal("second Escape did not exit fullscreen")
	}
}
