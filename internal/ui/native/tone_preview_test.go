package native

import (
	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"playerdoit/internal/dsp"
	"playerdoit/internal/player"
	"testing"
	"time"
)

func finishTonePreview(t *testing.T, ui *UI) {
	t.Helper()
	select {
	case err := <-ui.actionResults:
		if err != nil {
			t.Fatal(err)
		}
		ui.actionResults <- err
	case <-time.After(time.Second):
		t.Fatal("preview blocked")
	}
	ui.receiveAction()
}

func TestTonePreviewCancelRestoresSavedSound(t *testing.T) {
	saved := dsp.ToneConfig{Enabled: true, GainsDB: [10]float64{1, 2, 3}}
	draft := dsp.ToneConfig{Enabled: true, GainsDB: [10]float64{-10, -8, -5, 0, 2, 3}}
	ui := &UI{window: new(app.Window), controller: player.NewController(), actionResults: make(chan error, 1), preferences: defaultPreferences()}
	defer ui.controller.Close()
	ui.preferences.Tone = saved
	ui.controller.ConfigureTone(saved)
	ui.editPreferences(ui.preferences)
	ui.settings.open = true
	ui.tonePanel.applied = saved
	ui.editTone(draft)
	gtx := layout.Context{Ops: new(op.Ops), Now: time.Now()}
	ui.updateTone(gtx)
	finishTonePreview(t, ui)
	if got := ui.controller.Snapshot().Tone; got != draft {
		t.Fatalf("preview=%+v", got)
	}
	ui.settings.open = false
	ui.updateTone(gtx)
	finishTonePreview(t, ui)
	if got := ui.controller.Snapshot().Tone; got != saved {
		t.Fatalf("cancel kept preview=%+v", got)
	}
	// After saving, closing must keep the new settings instead of the old baseline.
	ui.settings.open = true
	ui.editTone(draft)
	ui.updateTone(layout.Context{Ops: new(op.Ops), Now: gtx.Now.Add(time.Second)})
	finishTonePreview(t, ui)
	ui.preferences.Tone = draft
	ui.settings.open = false
	ui.updateTone(gtx)
	finishTonePreview(t, ui)
	if got := ui.controller.Snapshot().Tone; got != draft {
		t.Fatalf("saved tone reverted: %+v", got)
	}
}
