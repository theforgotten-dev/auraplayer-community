package native

import (
	"gioui.org/app"
	"playerdoit/internal/player"
	"testing"
	"time"
)

func TestSlowActionDoesNotBlockUIState(t *testing.T) {
	ui := &UI{window: new(app.Window), controller: player.NewController(), actionResults: make(chan error, 1)}
	defer ui.controller.Close()
	release := make(chan struct{})
	ui.runAction(func() error { <-release; return nil })
	if !ui.actionBusy {
		t.Fatal("action was not dispatched")
	}
	done := make(chan struct{})
	go func() { ui.refreshState(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("UI blocked during loading")
	}
	close(release)
	select {
	case err := <-ui.actionResults:
		if err != nil {
			t.Fatal(err)
		}
		ui.actionResults <- err
	case <-time.After(time.Second):
		t.Fatal("action did not finish")
	}
	ui.receiveAction()
	if ui.actionBusy {
		t.Fatal("completion was not received")
	}
}
