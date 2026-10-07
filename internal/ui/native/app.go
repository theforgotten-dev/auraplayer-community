// Package native implements Aura's GPU-rendered desktop interface with Gio.
package native

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"

	"playerdoit/internal/dsp"
	"playerdoit/internal/player"
)

const (
	windowWidth  = 1280
	windowHeight = 780
	minWidth     = 900
	minHeight    = 620
)

type fileResult struct {
	paths []string
	skin  *Skin
	err   error
}

type UI struct {
	window     *app.Window
	explorer   *explorer.Explorer
	controller *player.Controller
	theme      *material.Theme
	ops        op.Ops

	skin    Skin
	palette Palette

	state       player.Snapshot
	spectrum    dsp.Spectrum
	nextPoll    time.Time
	lastSeek    time.Time
	message     string
	showSkins   bool
	dialogBusy  bool
	fileResults chan fileResult

	openButton     widget.Clickable
	previousButton widget.Clickable
	stopButton     widget.Clickable
	toggleButton   widget.Clickable
	nextButton     widget.Clickable
	skinsButton    widget.Clickable
	closeSkin      widget.Clickable
	importSkin     widget.Clickable
	skinButtons    [3]widget.Clickable
	queueButtons   map[string]*widget.Clickable
	queueList      widget.List
	seek           widget.Float
	volume         widget.Float
}

// Run creates and owns Aura's native desktop window.
func Run(initialPaths []string) error {
	window := new(app.Window)
	window.Option(
		app.Title("Aura Music Player"),
		app.Size(unit.Dp(windowWidth), unit.Dp(windowHeight)),
		app.MinSize(unit.Dp(minWidth), unit.Dp(minHeight)),
	)
	ui := newUI(window, initialPaths)
	defer ui.controller.Close()
	return ui.loop()
}

func newUI(window *app.Window, initialPaths []string) *UI {
	theme := material.NewTheme()
	theme.TextSize = unit.Sp(12)
	theme.FingerSize = unit.Dp(34)
	skin := persistedSkin()
	ui := &UI{
		window:       window,
		explorer:     explorer.NewExplorer(window),
		controller:   player.NewController(),
		theme:        theme,
		skin:         skin,
		fileResults:  make(chan fileResult, 2),
		queueButtons: make(map[string]*widget.Clickable),
	}
	ui.palette = paletteFor(ui.skin)
	ui.volume.Value = 0.8
	ui.queueList.Axis = layout.Vertical
	if len(initialPaths) > 0 {
		if err := ui.controller.AddFiles(initialPaths); err != nil {
			ui.message = "ERR:: " + err.Error()
		} else if err := ui.controller.Play(0); err != nil {
			ui.message = "ERR:: " + err.Error()
		}
	}
	ui.refreshState()
	return ui
}

func (ui *UI) loop() error {
	for {
		event := ui.window.Event()
		ui.explorer.ListenEvents(event)
		switch event := event.(type) {
		case app.DestroyEvent:
			return event.Err
		case app.FrameEvent:
			ui.receiveFiles()
			gtx := app.NewContext(&ui.ops, event)
			ui.update(gtx)
			ui.layout(gtx)
			event.Frame(gtx.Ops)
		}
	}
}

func (ui *UI) update(gtx layout.Context) {
	if !gtx.Now.Before(ui.nextPoll) {
		ui.refreshState()
		ui.nextPoll = gtx.Now.Add(200 * time.Millisecond)
	}
	if ui.state.State == player.StatePlaying {
		ui.spectrum = ui.controller.Spectrum()
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 30)})
	}

	if ui.openButton.Clicked(gtx) && !ui.dialogBusy {
		ui.chooseAudio()
	}
	if ui.previousButton.Clicked(gtx) {
		ui.runAction(ui.controller.Previous())
	}
	if ui.stopButton.Clicked(gtx) {
		ui.runAction(ui.controller.Stop())
	}
	if ui.toggleButton.Clicked(gtx) {
		ui.runAction(ui.controller.Toggle())
	}
	if ui.nextButton.Clicked(gtx) {
		ui.runAction(ui.controller.Next())
	}
	if ui.skinsButton.Clicked(gtx) {
		ui.showSkins = true
	}
	if ui.closeSkin.Clicked(gtx) {
		ui.showSkins = false
	}
	for index := range ui.skinButtons {
		if ui.skinButtons[index].Clicked(gtx) {
			ui.applySkin(builtinSkins[index])
		}
	}
	if ui.importSkin.Clicked(gtx) && !ui.dialogBusy {
		ui.chooseSkin()
	}
	for id, button := range ui.queueButtons {
		if !button.Clicked(gtx) {
			continue
		}
		for index, track := range ui.state.Queue {
			if track.ID == id {
				ui.runAction(ui.controller.Play(index))
				break
			}
		}
	}
}

func (ui *UI) layout(gtx layout.Context) layout.Dimensions {
	paint.Fill(gtx.Ops, ui.palette.Background)
	content := func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(ui.spacing()))}.Layout(gtx,
			layout.Rigid(ui.layoutHeader),
			layout.Flexed(1, ui.layoutWorkspace),
			layout.Rigid(ui.layoutTransport),
		)
	}
	main := func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(14)).Layout(gtx, content)
	}
	if !ui.showSkins {
		return main(gtx)
	}
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions { return main(gtx.Disabled()) }),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paint.Fill(gtx.Ops, color.NRGBA{A: 185})
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(ui.layoutSkinManager),
	)
}

func (ui *UI) layoutHeader(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.Y = gtx.Dp(68)
	return ui.panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(12)}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(46), gtx.Dp(38)))
						return ui.outlineBox(gtx, ui.palette.Accent, func(gtx layout.Context) layout.Dimensions {
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, ">_", 16, ui.palette.Accent, true) })
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.label(gtx, "PLAYERDOIT::NATIVE_AUDIO_ENGINE", 9, ui.palette.Muted, false)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.label(gtx, "AURA_CONSOLE", 19, ui.palette.Accent, true)
							}),
						)
					}),
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				status := "ENGINE_READY"
				statusColor := ui.palette.Muted
				if ui.state.State == player.StatePlaying {
					status, statusColor = "ENGINE_PLAYING", ui.palette.Accent
				}
				if ui.state.State == player.StatePaused {
					status = "ENGINE_PAUSED"
				}
				if ui.state.State == player.StateError {
					status, statusColor = "ENGINE_ERROR", ui.palette.Danger
				}
				return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, "●  "+status, 9, statusColor, false)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.button(gtx, &ui.openButton, "[ LOAD MP3 ]", true)
			}),
		)
	})
}

func (ui *UI) layoutWorkspace(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(ui.spacing()))}.Layout(gtx,
		layout.Flexed(.82, ui.layoutQueue),
		layout.Flexed(1.55, ui.layoutNowPlaying),
		layout.Flexed(.82, ui.layoutMetadata),
	)
}

func (ui *UI) layoutQueue(gtx layout.Context) layout.Dimensions {
	return ui.panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(8)}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.heading(gtx, "// LOCAL_LIBRARY", fmt.Sprintf("QUEUE  %02d", len(ui.state.Queue)))
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(ui.state.Queue) == 0 {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, "[ ]\n\nQUEUE_EMPTY\nLOAD ONE OR MORE MP3 FILES", 10, ui.palette.Muted, false)
					})
				}
				return material.List(ui.theme, &ui.queueList).Layout(gtx, len(ui.state.Queue), func(gtx layout.Context, index int) layout.Dimensions {
					track := ui.state.Queue[index]
					button := ui.queueButton(track.ID)
					return button.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						active := index == ui.state.CurrentIndex
						background, border, foreground := ui.palette.Panel, ui.palette.Line, ui.palette.Muted
						if active {
							background, border, foreground = ui.palette.PanelRaised, ui.palette.Accent, ui.palette.Accent
						}
						return ui.rowBox(gtx, background, border, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, fmt.Sprintf(">%02d", index+1), 9, foreground, false)
								}),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Left: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return ui.singleLine(gtx, track.Title, 10, foreground) })
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, formatTime(track.Duration), 9, ui.palette.Muted, false)
								}),
							)
						})
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, "J/K SELECT  ·  ENTER PLAY", 8, ui.palette.Muted, false)
				})
			}),
		)
	})
}

func (ui *UI) layoutNowPlaying(gtx layout.Context) layout.Dimensions {
	track := ui.currentTrack()
	title := "NO_TRACK_SELECTED"
	if track != nil {
		title = track.Title
	}
	return ui.panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(9)}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.heading(gtx, "// NOW_PLAYING", title) }),
			layout.Flexed(1, ui.layoutSpectrum),
			layout.Rigid(ui.layoutTimeline),
		)
	})
}

func (ui *UI) layoutSpectrum(gtx layout.Context) layout.Dimensions {
	return ui.outlineBox(gtx, ui.palette.Line, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 10, Right: 10, Bottom: 8, Left: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(6)}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, "FFT_2048 :: 40HZ—20KHZ", 8, ui.palette.Muted, false)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, "● CH_L   ● CH_R", 8, ui.palette.Text, false)
						}),
					)
				}),
				layout.Flexed(1, ui.drawSpectrum),
				layout.Rigid(ui.layoutChannelLevels),
			)
		})
	})
}

func (ui *UI) drawSpectrum(gtx layout.Context) layout.Dimensions {
	size := gtx.Constraints.Max
	if size.X <= 0 || size.Y <= 0 {
		return layout.Dimensions{}
	}
	visualHeight := min(gtx.Dp(180), size.Y)
	visualTop := (size.Y - visualHeight) / 2
	for row := 1; row < 4; row++ {
		y := visualTop + visualHeight*row/4
		fillRect(gtx.Ops, image.Rect(0, y, size.X, y+1), withAlpha(ui.palette.Line, 100))
	}
	gap := max(2, size.X/170)
	bandWidth := max(4, (size.X-gap*15)/16)
	barGap := max(1, bandWidth/9)
	barWidth := max(1, (bandWidth-barGap)/2)
	graphHeight := max(1, visualHeight)
	graphBottom := visualTop + graphHeight
	for index := 0; index < 16; index++ {
		x := index * (bandWidth + gap)
		leftHeight := int(float64(graphHeight) * ui.spectrum.Left[index])
		rightHeight := int(float64(graphHeight) * ui.spectrum.Right[index])
		fillSegmentedBar(gtx.Ops, image.Rect(x, graphBottom-leftHeight, x+barWidth, graphBottom), ui.palette.ChannelLeft, ui.palette.Background)
		fillSegmentedBar(gtx.Ops, image.Rect(x+barWidth+barGap, graphBottom-rightHeight, x+barWidth*2+barGap, graphBottom), ui.palette.ChannelRight, ui.palette.Background)
	}
	return layout.Dimensions{Size: size}
}

func (ui *UI) layoutChannelLevels(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(5)}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, "L", 8, ui.palette.Muted, false) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return ui.levelBar(gtx, ui.spectrum.LeftLevel, ui.palette.ChannelLeft)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, "R", 8, ui.palette.Muted, false) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return ui.levelBar(gtx, ui.spectrum.RightLevel, ui.palette.ChannelRight)
		}),
	)
}

func (ui *UI) layoutTimeline(gtx layout.Context) layout.Dimensions {
	duration := 0.0
	if track := ui.currentTrack(); track != nil {
		duration = track.Duration
	}
	if !ui.seek.Dragging() && duration > 0 {
		ui.seek.Value = float32(ui.state.Position / duration)
	}
	before := ui.seek.Value
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			slider := material.Slider(ui.theme, &ui.seek)
			slider.Color = ui.palette.Accent
			slider.FingerSize = 24
			dimensions := slider.Layout(gtx)
			if ui.seek.Value != before && duration > 0 && (gtx.Now.Sub(ui.lastSeek) >= 80*time.Millisecond || !ui.seek.Dragging()) {
				ui.lastSeek = gtx.Now
				ui.runAction(ui.controller.Seek(float64(ui.seek.Value) * duration))
			}
			return dimensions
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, "T+"+formatTime(ui.state.Position), 8, ui.palette.Muted, false)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 0)}
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, "T+"+formatTime(duration), 8, ui.palette.Muted, false)
				}),
			)
		}),
	)
}

func (ui *UI) layoutMetadata(gtx layout.Context) layout.Dimensions {
	track := ui.currentTrack()
	return ui.panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(7)}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.heading(gtx, "// SIGNAL_INSPECTOR", "METADATA") }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.metadataRow(gtx, "TÍTULO", trackValue(track, func(t player.Track) string { return t.Title }))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.metadataRow(gtx, "CODEC", trackValue(track, func(t player.Track) string { return t.Codec }))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.metadataRow(gtx, "FRECUENCIA", trackValue(track, func(t player.Track) string { return fmt.Sprintf("%.1f kHz", float64(t.SampleRate)/1000) }))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.metadataRow(gtx, "CANALES", trackValue(track, func(t player.Track) string {
					if t.Channels == 2 {
						return "ESTÉREO"
					}
					return "MONO"
				}))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.metadataRow(gtx, "RUTA", trackValue(track, func(t player.Track) string { return t.Path }))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.rowBox(gtx, ui.palette.PanelRaised, ui.palette.Line, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, "OUTPUT_PIPELINE", 8, ui.palette.ChannelRight, true)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, "MP3 → PCM S16LE → SRC 48 kHz → FFT → Oto v3", 8, ui.palette.Accent, false)
						}),
					)
				})
			}),
		)
	})
}

func (ui *UI) layoutTransport(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.Y = gtx.Dp(64)
	return ui.panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(6)}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &ui.skinsButton, "[ SKINS ]", false) }),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if ui.message == "" {
							return layout.Dimensions{}
						}
						return ui.label(gtx, "  "+ui.message, 8, ui.palette.Danger, false)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				toggle := "PLAY"
				if ui.state.State == player.StatePlaying {
					toggle = "PAUSE"
				}
				return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(6)}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &ui.previousButton, "|<", false) }),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &ui.stopButton, "[■]", false) }),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &ui.toggleButton, toggle, true) }),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &ui.nextButton, ">|", false) }),
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, "VOL", 8, ui.palette.Muted, true) }),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(150), gtx.Dp(28)))
							before := ui.volume.Value
							slider := material.Slider(ui.theme, &ui.volume)
							slider.Color = ui.palette.Accent
							slider.FingerSize = 24
							dimensions := slider.Layout(gtx)
							if before != ui.volume.Value {
								ui.runAction(ui.controller.SetVolume(float64(ui.volume.Value)))
							}
							return dimensions
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, fmt.Sprintf("%3.0f%%", ui.volume.Value*100), 8, ui.palette.Accent, true)
						}),
					)
				})
			}),
		)
	})
}

func (ui *UI) layoutSkinManager(gtx layout.Context) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		width, height := gtx.Dp(620), gtx.Dp(330)
		width = min(width, gtx.Constraints.Max.X-40)
		height = min(height, gtx.Constraints.Max.Y-40)
		gtx.Constraints = layout.Exact(image.Pt(width, height))
		return ui.outlineBox(gtx, ui.palette.LineHot, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(18).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(14)}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return ui.heading(gtx, "// NATIVE_VISUAL_MOD_LOADER", "SKIN_MANAGER")
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &ui.closeSkin, "[ X ]", false) }),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, "Selecciona una skin nativa o importa un manifiesto JSON propio.", 10, ui.palette.Muted, false)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(8)}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return ui.skinCard(gtx, 0) }),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return ui.skinCard(gtx, 1) }),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return ui.skinCard(gtx, 2) }),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return ui.label(gtx, "ACTIVE:: "+ui.skin.Name, 9, ui.palette.Accent, true)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.button(gtx, &ui.importSkin, "[ IMPORT JSON ]", true)
							}),
						)
					}),
				)
			})
		})
	})
}

func (ui *UI) skinCard(gtx layout.Context, index int) layout.Dimensions {
	skin := builtinSkins[index]
	palette := paletteFor(skin)
	return ui.skinButtons[index].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return ui.rowBox(gtx, palette.Background, palette.Line, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					fillRect(gtx.Ops, image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Dp(64)), palette.PanelRaised)
					return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(64))}
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.labelWithPalette(gtx, skin.Name, 8, palette.Accent, true)
				}),
			)
		})
	})
}

func (ui *UI) chooseAudio() {
	ui.dialogBusy = true
	go func() {
		readers, err := ui.explorer.ChooseFiles(".mp3")
		if errors.Is(err, explorer.ErrUserDecline) {
			err = nil
		}
		paths := make([]string, 0, len(readers))
		for _, reader := range readers {
			if file, ok := reader.(*os.File); ok {
				paths = append(paths, file.Name())
			}
			_ = reader.Close()
		}
		ui.fileResults <- fileResult{paths: paths, err: err}
		ui.window.Invalidate()
	}()
}

func (ui *UI) chooseSkin() {
	ui.dialogBusy = true
	go func() {
		reader, err := ui.explorer.ChooseFile(".json")
		if errors.Is(err, explorer.ErrUserDecline) {
			err = nil
		}
		var skin *Skin
		if err == nil && reader != nil {
			loaded, loadErr := loadSkin(reader)
			err = loadErr
			skin = &loaded
			_ = reader.Close()
		}
		ui.fileResults <- fileResult{skin: skin, err: err}
		ui.window.Invalidate()
	}()
}

func (ui *UI) receiveFiles() {
	for {
		select {
		case result := <-ui.fileResults:
			ui.dialogBusy = false
			if result.err != nil {
				ui.message = "ERR:: " + result.err.Error()
				continue
			}
			if len(result.paths) > 0 {
				ui.runAction(ui.controller.AddFiles(result.paths))
			}
			if result.skin != nil {
				ui.applySkin(*result.skin)
				ui.showSkins = false
			}
		default:
			return
		}
	}
}

func (ui *UI) applySkin(skin Skin) {
	ui.skin = skin
	ui.palette = paletteFor(skin)
	ui.message = "SKIN:: " + skin.Name
	if err := persistSkin(skin); err != nil {
		ui.message = "ERR:: save skin: " + err.Error()
	}
	ui.window.Invalidate()
}

func (ui *UI) runAction(err error) {
	if err != nil {
		ui.message = "ERR:: " + err.Error()
	} else {
		ui.message = ""
	}
	ui.refreshState()
	ui.window.Invalidate()
}

func (ui *UI) refreshState() {
	ui.state = ui.controller.Snapshot()
	ui.volume.Value = float32(ui.state.Volume)
	if ui.state.State != player.StatePlaying {
		ui.spectrum = dsp.EmptySpectrum()
	}
}

func (ui *UI) currentTrack() *player.Track {
	if ui.state.CurrentIndex < 0 || ui.state.CurrentIndex >= len(ui.state.Queue) {
		return nil
	}
	return &ui.state.Queue[ui.state.CurrentIndex]
}

func (ui *UI) queueButton(id string) *widget.Clickable {
	button := ui.queueButtons[id]
	if button == nil {
		button = new(widget.Clickable)
		ui.queueButtons[id] = button
	}
	return button
}

func (ui *UI) panel(gtx layout.Context, body layout.Widget) layout.Dimensions {
	return ui.outlineBox(gtx, ui.palette.Line, func(gtx layout.Context) layout.Dimensions { return layout.UniformInset(14).Layout(gtx, body) })
}

func (ui *UI) outlineBox(gtx layout.Context, border color.NRGBA, body layout.Widget) layout.Dimensions {
	size := gtx.Constraints.Max
	fillRect(gtx.Ops, image.Rectangle{Max: size}, ui.palette.Panel)
	stroke := max(1, int(ui.skin.Metrics.StrokeWidth))
	fillRect(gtx.Ops, image.Rect(0, 0, size.X, stroke), border)
	fillRect(gtx.Ops, image.Rect(0, size.Y-stroke, size.X, size.Y), border)
	fillRect(gtx.Ops, image.Rect(0, 0, stroke, size.Y), border)
	fillRect(gtx.Ops, image.Rect(size.X-stroke, 0, size.X, size.Y), border)
	return body(gtx)
}

func (ui *UI) rowBox(gtx layout.Context, background, border color.NRGBA, body layout.Widget) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			fillRect(gtx.Ops, image.Rectangle{Max: gtx.Constraints.Max}, background)
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions { return layout.UniformInset(9).Layout(gtx, body) }),
	)
}

func (ui *UI) heading(gtx layout.Context, eyebrow, title string) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, eyebrow, 8, ui.palette.Muted, true) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.singleLine(gtx, title, 13, ui.palette.Text) }),
	)
}

func (ui *UI) metadataRow(gtx layout.Context, key, value string) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, ":: "+key, 8, ui.palette.Muted, true) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.singleLine(gtx, value, 9, ui.palette.Text) }),
	)
}

func (ui *UI) button(gtx layout.Context, clickable *widget.Clickable, text string, primary bool) layout.Dimensions {
	style := material.Button(ui.theme, clickable, text)
	style.TextSize = 10
	style.CornerRadius = unit.Dp(ui.skin.Metrics.CornerRadius)
	style.Inset = layout.Inset{Top: 9, Right: 12, Bottom: 9, Left: 12}
	style.Background, style.Color = ui.palette.PanelRaised, ui.palette.Muted
	if primary {
		style.Background, style.Color = ui.palette.Accent, ui.palette.Background
	}
	return style.Layout(gtx)
}

func (ui *UI) label(gtx layout.Context, value string, size float32, foreground color.NRGBA, bold bool) layout.Dimensions {
	return ui.labelWithPalette(gtx, value, size, foreground, bold)
}

func (ui *UI) labelWithPalette(gtx layout.Context, value string, size float32, foreground color.NRGBA, bold bool) layout.Dimensions {
	style := material.Label(ui.theme, unit.Sp(size), value)
	style.Color = foreground
	if bold {
		style.Font.Weight = font.Bold
	}
	return style.Layout(gtx)
}

func (ui *UI) singleLine(gtx layout.Context, value string, size float32, foreground color.NRGBA) layout.Dimensions {
	style := material.Label(ui.theme, unit.Sp(size), value)
	style.Color = foreground
	style.MaxLines = 1
	style.Truncator = "…"
	style.WrapPolicy = text.WrapGraphemes
	return style.Layout(gtx)
}

func (ui *UI) levelBar(gtx layout.Context, value float64, foreground color.NRGBA) layout.Dimensions {
	height := gtx.Dp(4)
	width := gtx.Constraints.Max.X
	fillRect(gtx.Ops, image.Rect(0, 0, width, height), withAlpha(ui.palette.Line, 100))
	fillRect(gtx.Ops, image.Rect(0, 0, int(float64(width)*value), height), foreground)
	return layout.Dimensions{Size: image.Pt(width, height)}
}

func (ui *UI) spacing() float32 {
	if ui.skin.Metrics.Spacing > 0 {
		return ui.skin.Metrics.Spacing
	}
	return 8
}

func fillRect(ops *op.Ops, rectangle image.Rectangle, fill color.NRGBA) {
	if rectangle.Empty() {
		return
	}
	paint.FillShape(ops, fill, clip.Rect(rectangle).Op())
}

func fillSegmentedBar(ops *op.Ops, rectangle image.Rectangle, fill, mask color.NRGBA) {
	if rectangle.Empty() {
		return
	}
	fillRect(ops, rectangle, fill)
	for y := rectangle.Min.Y + 7; y < rectangle.Max.Y; y += 9 {
		fillRect(ops, image.Rect(rectangle.Min.X, y, rectangle.Max.X, min(y+2, rectangle.Max.Y)), mask)
	}
}

func withAlpha(value color.NRGBA, alpha uint8) color.NRGBA { value.A = alpha; return value }
func formatTime(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	total := int(seconds)
	return fmt.Sprintf("%d:%02d", total/60, total%60)
}
func trackValue(track *player.Track, getter func(player.Track) string) string {
	if track == nil {
		return "—"
	}
	return getter(*track)
}
