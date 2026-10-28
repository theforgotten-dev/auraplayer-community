package native

import (
	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"time"
)

func (ui *UI) setFullscreen(enabled bool) {
	if ui.fullscreen == enabled {
		return
	}
	if enabled {
		ui.restoreMode = ui.windowMode
		if ui.restoreMode != app.Maximized {
			ui.restoreMode = app.Windowed
		}
		ui.fullscreen = true
		ui.fullscreenActivity = time.Now()
		ui.fullscreenHover = false
		ui.showSkins = false
		ui.settings.open = false
		ui.window.Option(app.Fullscreen.Option())
	} else {
		ui.fullscreen = false
		ui.window.Option(ui.restoreMode.Option())
	}
	ui.window.Invalidate()
}

func (ui *UI) updateWindowShortcuts(gtx layout.Context) {
	for {
		e, ok := gtx.Event(pointer.Filter{Target: &ui.fullscreenPointer, Kinds: pointer.Move | pointer.Press | pointer.Drag})
		if !ok {
			break
		}
		if e, ok := e.(pointer.Event); ok && ui.fullscreen {
			ui.fullscreenActivity = gtx.Now
			ui.fullscreenHover = e.Position.Y > float32(gtx.Constraints.Max.Y-gtx.Dp(170))
		}
	}
	for {
		e, ok := gtx.Event(
			key.Filter{Name: key.NameF11},
			key.Filter{Name: key.NameReturn, Required: key.ModAlt},
			key.Filter{Name: key.NameEscape},
			key.Filter{Name: "P", Required: key.ModCtrl},
		)
		if !ok {
			break
		}
		if e, ok := e.(key.Event); ok && e.State == key.Press {
			switch e.Name {
			case "P":
				ui.openSettings()
			case key.NameF11, key.NameReturn:
				ui.setFullscreen(!ui.fullscreen)
			case key.NameEscape:
				if ui.settings.open {
					ui.settings.open = false
				} else if ui.fullscreen {
					ui.setFullscreen(false)
				} else {
					ui.showSkins = false
				}
			}
		}
	}
	if ui.fullscreenButton.Clicked(gtx) {
		ui.setFullscreen(!ui.fullscreen)
	}
	for {
		click, ok := ui.videoClick.Update(gtx)
		if !ok {
			break
		}
		if click.NumClicks == 2 {
			ui.setFullscreen(!ui.fullscreen)
		}
	}
}
