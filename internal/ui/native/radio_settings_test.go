package native

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestRadioAPIAndMirrorsPersistAndEditIndependently(t *testing.T) {
	p := defaultPreferences()
	p.Radio.BaseURL = "https://custom.example"
	p.Radio.Mirrors = nil
	for i := 0; i < 12; i++ {
		p.Radio.Mirrors = append(p.Radio.Mirrors, fmt.Sprintf("https://mirror%d.example", i))
	}
	path := filepath.Join(t.TempDir(), "preferences.json")
	if err := writePreferences(path, p); err != nil {
		t.Fatal(err)
	}
	loaded, err := readPreferences(path)
	if err != nil {
		t.Fatal(err)
	}
	ui := &UI{}
	ui.editPreferences(loaded)
	ui.radioSettings.mirrors.SetText("https://changed.example\nhttps://second.example")
	edited := ui.radioSettingsConfig()
	if edited.BaseURL != p.Radio.BaseURL || len(loaded.Radio.Mirrors) != 12 || len(edited.Mirrors) != 2 {
		t.Fatalf("loaded=%v edited=%v", loaded.Radio, edited)
	}
	cloned := clonePreferences(loaded)
	cloned.Radio.Mirrors[0] = "https://changed.example"
	if loaded.Radio.Mirrors[0] != p.Radio.Mirrors[0] {
		t.Fatal("editing mutated saved mirrors")
	}
}
