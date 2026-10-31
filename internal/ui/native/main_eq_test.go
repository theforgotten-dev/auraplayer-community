package native

import (
	"image"
	"testing"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget/material"
	"playerdoit/internal/player"
)

func TestMainEqualizerAppliesAndSavesOutsideSettings(t *testing.T) {
	profile := t.TempDir()
	t.Setenv("APPDATA", profile)
	t.Setenv("XDG_CONFIG_HOME", profile)
	ui := &UI{window: new(app.Window), controller: player.NewController(), actionResults: make(chan error, 1), preferences: defaultPreferences()}
	defer ui.controller.Close()
	ui.editMainEQ(ui.preferences.Tone)
	gtx := layout.Context{Ops: new(op.Ops), Now: time.Now()}
	ui.mainEQ.bass.Click()
	ui.updateMainEQ(gtx)
	finishTonePreview(t, ui)
	ui.updateMainEQ(gtx)
	if ui.settings.open || !ui.state.Tone.Enabled || ui.state.Tone.GainsDB[1] != 5 || ui.preferences.Tone != ui.state.Tone {
		t.Fatal("main equalizer did not apply outside the dialog")
	}
	ui.mainEQ.save.Click()
	ui.updateMainEQ(gtx)
	finishTonePreview(t, ui)
	ui.updateMainEQ(gtx)
	path, err := preferencesPath()
	if err != nil {
		t.Fatal(err)
	}
	saved, err := readPreferences(path)
	if err != nil || saved.Tone != ui.state.Tone {
		t.Fatal("equalizer not persisted", err)
	}
}

func TestMainEqualizerAndSearchColumnsFitTheirBounds(t *testing.T) {
	ui := &UI{theme: material.NewTheme(), skin: builtinSkins[0], palette: paletteFor(builtinSkins[0]), preferences: defaultPreferences()}
	ui.editMainEQ(ui.preferences.Tone)
	ui.editAudiusSettings(ui.preferences.Audius)
	for _, size := range []image.Point{image.Pt(900, 440), image.Pt(550, 400)} {
		gtx := layout.Context{Ops: new(op.Ops), Now: time.Now(), Constraints: layout.Exact(size)}
		dims := ui.layoutAudiusAndEQ(gtx)
		if dims.Size.X > size.X || dims.Size.Y > size.Y {
			t.Fatalf("columns exceed %v: %v", size, dims.Size)
		}
	}
}
