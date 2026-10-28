package native

import (
	"image"
	"testing"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget/material"
	"playerdoit/internal/player"
)

func TestFullscreenKeyboardAndRestore(t *testing.T) {
	ui := &UI{window: new(app.Window), windowMode: app.Maximized, actionBusy: true}
	var router input.Router
	gtx := layout.Context{Ops: new(op.Ops), Source: router.Source(), Now: time.Now()}
	ui.updateWindowShortcuts(gtx)
	press := func(name key.Name, mod key.Modifiers) {
		router.Queue(key.Event{Name: name, Modifiers: mod, State: key.Press})
		ui.updateWindowShortcuts(gtx)
	}
	press(key.NameF11, 0)
	if !ui.fullscreen || ui.restoreMode != app.Maximized {
		t.Fatal("F11 did not enter fullscreen preserving maximized mode")
	}
	press(key.NameEscape, 0)
	if ui.fullscreen {
		t.Fatal("Escape did not exit fullscreen")
	}
	press(key.NameReturn, key.ModAlt)
	if !ui.fullscreen {
		t.Fatal("Alt+Enter did not enter fullscreen")
	}
	press(key.NameReturn, key.ModAlt)
	if ui.fullscreen {
		t.Fatal("Alt+Enter did not exit fullscreen")
	}
	ui.videoClick.Click()
	ui.videoClick.Click()
	ui.updateWindowShortcuts(gtx)
	if !ui.fullscreen {
		t.Fatal("video double click did not enter fullscreen")
	}
	ui.fullscreenButton.Click()
	ui.updateWindowShortcuts(gtx)
	if ui.fullscreen {
		t.Fatal("fullscreen exit button did not restore window")
	}
}

func TestFullscreenFloatingControlsAndMouseWake(t *testing.T) {
	now := time.Now()
	ui := &UI{window: new(app.Window), fullscreen: true, fullscreenActivity: now.Add(-3 * time.Second), controller: player.NewController(), theme: material.NewTheme()}
	defer ui.controller.Close()
	ui.state.State = player.StatePlaying
	var router input.Router
	gtx := layout.Context{Ops: new(op.Ops), Source: router.Source(), Now: now, Constraints: layout.Exact(image.Pt(1920, 1080))}
	ui.updateWindowShortcuts(gtx)
	dims := ui.layoutFullscreenVideo(gtx)
	if dims.Size != gtx.Constraints.Max {
		t.Fatalf("fullscreen area=%v", dims.Size)
	}
	if controlsOpacity(now.Sub(ui.fullscreenActivity)) != 0 {
		t.Fatal("inactive controls did not disappear")
	}
	router.Frame(gtx.Ops)
	router.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(400, 100)})
	ui.updateWindowShortcuts(gtx)
	if !ui.fullscreenActivity.Equal(now) {
		t.Fatal("moving mouse did not wake controls")
	}
	if controlsOpacity(now.Sub(ui.fullscreenActivity)) != 1 {
		t.Fatal("controls not fully visible after mouse move")
	}
	if alpha := controlsOpacity(2150 * time.Millisecond); alpha < .49 || alpha > .51 {
		t.Fatalf("fade midpoint=%v", alpha)
	}
}
