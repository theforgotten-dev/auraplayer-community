// Package native implements Aura's GPU-rendered desktop interface with Gio.
package native

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/system"
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
	windowWidth   = 1280
	windowHeight  = 780
	compactWidth  = 720
	compactHeight = 420
	minWidth      = 600
	minHeight     = 360
	windowBarH    = 38
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

	state         player.Snapshot
	spectrum      dsp.Spectrum
	visualLeft    [16]float64
	visualRight   [16]float64
	peaksLeft     [16]float64
	peaksRight    [16]float64
	peakHoldLeft  [16]int
	peakHoldRight [16]int
	levelLeft     float64
	levelRight    float64
	lastAnimTime  time.Time
	nextPoll      time.Time
	lastSeek      time.Time
	message       string
	showSkins     bool
	dialogBusy    bool
	compactMode   bool
	windowMode    app.WindowMode
	fileResults   chan fileResult
	decorations   widget.Decorations

	openButton     widget.Clickable
	compactButton  widget.Clickable
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
		app.Decorated(false),
		app.Size(unit.Dp(compactWidth), unit.Dp(compactHeight+windowBarH)),
		app.MinSize(unit.Dp(minWidth), unit.Dp(minHeight+windowBarH)),
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
		compactMode:  true,
		fileResults:  make(chan fileResult, 2),
		queueButtons: make(map[string]*widget.Clickable),
	}
	ui.palette = paletteFor(ui.skin)
	ui.spectrum = dsp.EmptySpectrum()
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
		case app.ConfigEvent:
			ui.windowMode = event.Config.Mode
			ui.decorations.Maximized = event.Config.Mode == app.Maximized
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
	if actions := ui.decorations.Update(gtx); actions != 0 {
		ui.window.Perform(actions)
	}
	if !gtx.Now.Before(ui.nextPoll) {
		ui.refreshState()
		ui.nextPoll = gtx.Now.Add(200 * time.Millisecond)
	}
	if ui.state.State == player.StatePlaying {
		ui.animateSpectrum(gtx.Now, true)
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 60)})
	} else if ui.hasActiveSpectrum() {
		ui.animateSpectrum(gtx.Now, false)
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 60)})
	}

	if ui.openButton.Clicked(gtx) && !ui.dialogBusy {
		ui.chooseAudio()
	}
	if ui.compactButton.Clicked(gtx) {
		ui.compactMode = !ui.compactMode
		if ui.compactMode {
			ui.window.Option(
				app.Size(unit.Dp(compactWidth), unit.Dp(compactHeight+windowBarH)),
			)
		} else {
			ui.window.Option(
				app.Size(unit.Dp(windowWidth), unit.Dp(windowHeight+windowBarH)),
			)
		}
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
	playerContent := func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(14)).Layout(gtx, content)
	}
	main := func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(ui.layoutWindowBar),
			layout.Flexed(1, playerContent),
		)
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

func (ui *UI) layoutWindowBar(gtx layout.Context) layout.Dimensions {
	height := gtx.Dp(windowBarH)
	gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, height))
	fillRect(gtx.Ops, image.Rectangle{Max: gtx.Constraints.Max}, ui.palette.PanelRaised)
	fillRect(gtx.Ops, image.Rect(0, height-1, gtx.Constraints.Max.X, height), ui.palette.LineHot)

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return ui.decorations.LayoutMove(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: 12, Right: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(9)}.Layout(gtx,
						layout.Rigid(ui.layoutWindowIcon),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, "AURA // PLAYERDOIT", 9, ui.palette.Text, true)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, "NATIVE_AUDIO", 7, ui.palette.Muted, false)
						}),
					)
				})
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.windowButton(gtx, system.ActionMinimize, windowIconMinimize, false)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.windowButton(gtx, system.ActionMaximize, windowIconMaximize, false)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.windowButton(gtx, system.ActionClose, windowIconClose, true)
				}),
			)
		}),
	)
}

type windowIcon uint8

const (
	windowIconMinimize windowIcon = iota
	windowIconMaximize
	windowIconClose
)

func (ui *UI) layoutWindowIcon(gtx layout.Context) layout.Dimensions {
	size := image.Pt(gtx.Dp(18), gtx.Dp(18))
	fillRect(gtx.Ops, image.Rectangle{Max: size}, ui.palette.Background)
	stroke := max(1, gtx.Dp(1))
	fillRect(gtx.Ops, image.Rect(0, 0, size.X, stroke), ui.palette.Accent)
	fillRect(gtx.Ops, image.Rect(0, size.Y-stroke, size.X, size.Y), ui.palette.Accent)
	fillRect(gtx.Ops, image.Rect(0, 0, stroke, size.Y), ui.palette.Accent)
	fillRect(gtx.Ops, image.Rect(size.X-stroke, 0, size.X, size.Y), ui.palette.Accent)
	barW := max(2, gtx.Dp(2))
	for index, barH := range []int{5, 10, 7, 13} {
		x := gtx.Dp(3) + index*gtx.Dp(3)
		h := gtx.Dp(unit.Dp(barH))
		fillRect(gtx.Ops, image.Rect(x, size.Y-gtx.Dp(3)-h, x+barW, size.Y-gtx.Dp(3)), ui.palette.Accent)
	}
	return layout.Dimensions{Size: size}
}

func (ui *UI) windowButton(gtx layout.Context, action system.Action, icon windowIcon, danger bool) layout.Dimensions {
	button := ui.decorations.Clickable(action)
	width, height := gtx.Dp(44), gtx.Dp(windowBarH)
	gtx.Constraints = layout.Exact(image.Pt(width, height))
	return button.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		system.ActionInputOp(action).Add(gtx.Ops)
		background := ui.palette.PanelRaised
		foreground := ui.palette.Muted
		if button.Hovered() {
			background = ui.palette.Line
			foreground = ui.palette.Text
			if danger {
				background = ui.palette.Danger
				foreground = ui.palette.Background
			}
		}
		fillRect(gtx.Ops, image.Rectangle{Max: gtx.Constraints.Max}, background)
		ui.drawWindowIcon(gtx, icon, foreground)
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
}

func (ui *UI) drawWindowIcon(gtx layout.Context, icon windowIcon, foreground color.NRGBA) {
	centerX, centerY := float32(gtx.Constraints.Max.X)/2, float32(gtx.Constraints.Max.Y)/2
	strokeWidth := float32(max(1, gtx.Dp(1)))
	if icon == windowIconMinimize {
		fillRect(gtx.Ops, image.Rect(int(centerX)-6, int(centerY)+4, int(centerX)+6, int(centerY)+5), foreground)
		return
	}
	if icon == windowIconMaximize {
		if ui.windowMode == app.Maximized {
			drawOutlineRect(gtx.Ops, image.Rect(int(centerX)-4, int(centerY)-5, int(centerX)+7, int(centerY)+4), foreground)
			drawOutlineRect(gtx.Ops, image.Rect(int(centerX)-7, int(centerY)-2, int(centerX)+4, int(centerY)+7), foreground)
		} else {
			drawOutlineRect(gtx.Ops, image.Rect(int(centerX)-6, int(centerY)-6, int(centerX)+6, int(centerY)+6), foreground)
		}
		return
	}

	var path clip.Path
	path.Begin(gtx.Ops)
	path.MoveTo(f32.Pt(centerX-5, centerY-5))
	path.LineTo(f32.Pt(centerX+5, centerY+5))
	path.MoveTo(f32.Pt(centerX+5, centerY-5))
	path.LineTo(f32.Pt(centerX-5, centerY+5))
	outline := clip.Stroke{Path: path.End(), Width: strokeWidth}.Op().Push(gtx.Ops)
	paint.Fill(gtx.Ops, foreground)
	outline.Pop()
}

func drawOutlineRect(ops *op.Ops, rectangle image.Rectangle, foreground color.NRGBA) {
	fillRect(ops, image.Rect(rectangle.Min.X, rectangle.Min.Y, rectangle.Max.X, rectangle.Min.Y+1), foreground)
	fillRect(ops, image.Rect(rectangle.Min.X, rectangle.Max.Y-1, rectangle.Max.X, rectangle.Max.Y), foreground)
	fillRect(ops, image.Rect(rectangle.Min.X, rectangle.Min.Y, rectangle.Min.X+1, rectangle.Max.Y), foreground)
	fillRect(ops, image.Rect(rectangle.Max.X-1, rectangle.Min.Y, rectangle.Max.X, rectangle.Max.Y), foreground)
}

func (ui *UI) isCompact(gtx layout.Context) bool {
	return ui.compactMode || gtx.Constraints.Max.X < 920
}

func (ui *UI) layoutHeader(gtx layout.Context) layout.Dimensions {
	compact := ui.isCompact(gtx)
	headerH := 58
	if compact {
		headerH = 46
	}
	gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(headerH))
	return ui.panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(8)}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						iconW, iconH := gtx.Dp(38), gtx.Dp(32)
						iconText := float32(14)
						if compact {
							iconW, iconH = gtx.Dp(32), gtx.Dp(26)
							iconText = 11
						}
						gtx.Constraints = layout.Exact(image.Pt(iconW, iconH))
						return ui.outlineBox(gtx, ui.palette.Accent, func(gtx layout.Context) layout.Dimensions {
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return ui.label(gtx, ">_", iconText, ui.palette.Accent, true)
							})
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						title := "AURA // PLAYERDOIT"
						eyebrow := "KDE_CLI_EDITION :: OTO_ENGINE"
						titleSize := float32(16)
						if compact {
							title = "AURA // PLAYER"
							eyebrow = "CLI_EDITION"
							titleSize = 13
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.label(gtx, eyebrow, 7, ui.palette.Muted, false)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.label(gtx, title, titleSize, ui.palette.Accent, true)
							}),
						)
					}),
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if compact {
					return layout.Dimensions{}
				}
				status := "STANDBY"
				statusColor := ui.palette.Muted
				if ui.state.State == player.StatePlaying {
					status, statusColor = "STREAMING (48kHz S16LE)", ui.palette.Accent
				} else if ui.state.State == player.StatePaused {
					status, statusColor = "PAUSED", ui.palette.Text
				} else if ui.state.State == player.StateError {
					status, statusColor = "ERROR", ui.palette.Danger
				}
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, "●  ENGINE::"+status, 8, statusColor, true)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				modeText := "[ ◰ COMPACT ]"
				if compact {
					modeText = "[ ◰ FULL ]"
				}
				return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(6)}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.button(gtx, &ui.compactButton, modeText, false)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.button(gtx, &ui.openButton, "[ + LOAD MP3 ]", true)
					}),
				)
			}),
		)
	})
}

func (ui *UI) layoutWorkspace(gtx layout.Context) layout.Dimensions {
	if ui.isCompact(gtx) {
		return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(ui.spacing()))}.Layout(gtx,
			layout.Flexed(0.85, ui.layoutQueue),
			layout.Flexed(1.4, ui.layoutPlayerContext),
		)
	}
	return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(ui.spacing()))}.Layout(gtx,
		layout.Flexed(.85, ui.layoutQueue),
		layout.Flexed(1.45, ui.layoutPlayerContext),
		layout.Flexed(.85, ui.layoutMetadata),
	)
}

func (ui *UI) layoutPlayerContext(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(ui.spacing()))}.Layout(gtx,
		layout.Rigid(ui.layoutNowPlayingCard),
		layout.Flexed(1, ui.layoutSpectrum),
	)
}

func (ui *UI) layoutNowPlayingCard(gtx layout.Context) layout.Dimensions {
	track := ui.currentTrack()
	title := "NO_TRACK_LOADED"
	codec := "NONE"
	sampleRate := "—"
	channels := "—"
	duration := "—"
	if track != nil {
		title = track.Title
		codec = track.Codec
		sampleRate = fmt.Sprintf("%.1f kHz", float64(track.SampleRate)/1000)
		if track.Channels == 2 {
			channels = "STEREO"
		} else {
			channels = "MONO"
		}
		duration = formatTime(track.Duration)
	}

	return ui.panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(10)}.Layout(gtx,
			// Cybernetic ASCII Cassette/Vinyl Artwork Glyph
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				w, h := gtx.Dp(58), gtx.Dp(48)
				gtx.Constraints = layout.Exact(image.Pt(w, h))
				return ui.outlineBox(gtx, ui.palette.LineHot, func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						glyph := "[●══●]\n|░░░░|\n[AURA]"
						if ui.state.State == player.StatePlaying {
							glyph = "[▶══●]\n|████|\n[PLAY]"
						}
						return ui.label(gtx, glyph, 8, ui.palette.Accent, true)
					})
				})
			}),
			// Track Details and Metadata Badges
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(3)}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						trackNo := "--"
						if ui.state.CurrentIndex >= 0 {
							trackNo = fmt.Sprintf("%02d/%02d", ui.state.CurrentIndex+1, len(ui.state.Queue))
						}
						return ui.label(gtx, "// PLAYER_CONTEXT · NOW_PLAYING ["+trackNo+"]", 8, ui.palette.Muted, true)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.singleLine(gtx, title, 13, ui.palette.Text)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(4)}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.tagBadge(gtx, codec, ui.palette.PanelRaised, ui.palette.Accent)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.tagBadge(gtx, sampleRate, ui.palette.PanelRaised, ui.palette.Text)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.tagBadge(gtx, channels, ui.palette.PanelRaised, ui.palette.ChannelRight)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.tagBadge(gtx, duration, ui.palette.PanelRaised, ui.palette.Muted)
							}),
						)
					}),
				)
			}),
		)
	})
}

func (ui *UI) layoutSpectrum(gtx layout.Context) layout.Dimensions {
	return ui.outlineBox(gtx, ui.palette.Line, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 8, Right: 10, Bottom: 8, Left: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(5)}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, "// LIVE_SPECTRUM_ANALYZER · 60_FPS · FFT_2048", 8, ui.palette.Muted, true)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, "● CH_L   ● CH_R", 8, ui.palette.Accent, true)
						}),
					)
				}),
				layout.Flexed(1, ui.drawSpectrum),
				layout.Rigid(ui.layoutFrequencyScale),
				layout.Rigid(ui.layoutChannelLevels),
			)
		})
	})
}

func (ui *UI) layoutFrequencyScale(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, "40Hz", 7, ui.palette.Muted, false) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, "250Hz", 7, ui.palette.Muted, false) })
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, "1kHz", 7, ui.palette.Muted, false) })
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, "4kHz", 7, ui.palette.Muted, false) })
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, "20kHz", 7, ui.palette.Muted, false) }),
	)
}

func (ui *UI) tagBadge(gtx layout.Context, value string, background, foreground color.NRGBA) layout.Dimensions {
	return ui.rowBox(gtx, background, ui.palette.Line, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 2, Right: 6, Bottom: 2, Left: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, value, 8, foreground, true)
		})
	})
}

func (ui *UI) layoutQueue(gtx layout.Context) layout.Dimensions {
	totalDuration := 0.0
	for _, track := range ui.state.Queue {
		totalDuration += track.Duration
	}
	return ui.panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(6)}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return ui.heading(gtx, "// MEDIA_COLLECTION", fmt.Sprintf("QUEUE [%02d]", len(ui.state.Queue)))
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if len(ui.state.Queue) == 0 {
							return layout.Dimensions{}
						}
						return ui.tagBadge(gtx, formatTime(totalDuration), ui.palette.PanelRaised, ui.palette.Muted)
					}),
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(ui.state.Queue) == 0 {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, "[ ]\n\nQUEUE_EMPTY\nUSE [ + LOAD MP3 ]\nOR DRAG FILES", 9, ui.palette.Muted, false)
					})
				}
				return material.List(ui.theme, &ui.queueList).Layout(gtx, len(ui.state.Queue), func(gtx layout.Context, index int) layout.Dimensions {
					track := ui.state.Queue[index]
					button := ui.queueButton(track.ID)
					return button.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						active := index == ui.state.CurrentIndex
						background, border, foreground := ui.palette.Panel, ui.palette.Line, ui.palette.Muted
						bullet := fmt.Sprintf("░%02d", index+1)
						if active {
							background, border, foreground = ui.palette.PanelRaised, ui.palette.Accent, ui.palette.Accent
							bullet = fmt.Sprintf("▶%02d", index+1)
						}
						return ui.rowBox(gtx, background, border, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, bullet, 9, foreground, true)
								}),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Left: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return ui.singleLine(gtx, track.Title, 10, foreground)
									})
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, formatTime(track.Duration), 8, ui.palette.Muted, false)
								}),
							)
						})
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, "▲/▼ J/K SELECT · [ENTER] PLAY", 7, ui.palette.Muted, false)
				})
			}),
		)
	})
}

func (ui *UI) animateSpectrum(now time.Time, playing bool) {
	var raw dsp.Spectrum
	if playing {
		raw = ui.controller.Spectrum()
	}

	dt := 1.0 / 60.0
	if !ui.lastAnimTime.IsZero() {
		elapsed := now.Sub(ui.lastAnimTime).Seconds()
		if elapsed > 0 && elapsed < 0.2 {
			dt = elapsed
		}
	}
	ui.lastAnimTime = now

	// Winamp-style visual ballistics:
	// Bars fall quickly and smoothly: ~1.8 units/s (100% to 0% in ~550ms)
	barDecay := 1.8 * dt
	// Peak caps hold then fall slower: ~0.8 units/s (100% to 0% in ~1250ms)
	peakDecay := 0.8 * dt
	// VU level meters decay: ~2.2 units/s
	levelDecay := 2.2 * dt

	for i := 0; i < 16; i++ {
		// Left channel
		targetL := raw.Left[i]
		if targetL > ui.visualLeft[i] {
			ui.visualLeft[i] = targetL // Instant attack
		} else {
			ui.visualLeft[i] = math.Max(0, ui.visualLeft[i]-barDecay)
		}
		if ui.visualLeft[i] >= ui.peaksLeft[i] {
			ui.peaksLeft[i] = ui.visualLeft[i]
			ui.peakHoldLeft[i] = 16 // hold for ~260ms at 60 FPS
		} else if ui.peakHoldLeft[i] > 0 {
			ui.peakHoldLeft[i]--
		} else {
			ui.peaksLeft[i] = math.Max(0, ui.peaksLeft[i]-peakDecay)
		}

		// Right channel
		targetR := raw.Right[i]
		if targetR > ui.visualRight[i] {
			ui.visualRight[i] = targetR // Instant attack
		} else {
			ui.visualRight[i] = math.Max(0, ui.visualRight[i]-barDecay)
		}
		if ui.visualRight[i] >= ui.peaksRight[i] {
			ui.peaksRight[i] = ui.visualRight[i]
			ui.peakHoldRight[i] = 16
		} else if ui.peakHoldRight[i] > 0 {
			ui.peakHoldRight[i]--
		} else {
			ui.peaksRight[i] = math.Max(0, ui.peaksRight[i]-peakDecay)
		}
	}

	// Channel VU levels
	if raw.LeftLevel > ui.levelLeft {
		ui.levelLeft = raw.LeftLevel
	} else {
		ui.levelLeft = math.Max(0, ui.levelLeft-levelDecay)
	}

	if raw.RightLevel > ui.levelRight {
		ui.levelRight = raw.RightLevel
	} else {
		ui.levelRight = math.Max(0, ui.levelRight-levelDecay)
	}

	ui.spectrum.Left = ui.visualLeft
	ui.spectrum.Right = ui.visualRight
	ui.spectrum.LeftLevel = ui.levelLeft
	ui.spectrum.RightLevel = ui.levelRight
}

func (ui *UI) hasActiveSpectrum() bool {
	if ui.levelLeft > 0.002 || ui.levelRight > 0.002 {
		return true
	}
	for i := 0; i < 16; i++ {
		if ui.visualLeft[i] > 0.002 || ui.visualRight[i] > 0.002 ||
			ui.peaksLeft[i] > 0.002 || ui.peaksRight[i] > 0.002 {
			return true
		}
	}
	return false
}

func (ui *UI) drawSpectrum(gtx layout.Context) layout.Dimensions {
	size := gtx.Constraints.Max
	if size.X <= 0 || size.Y <= 0 {
		return layout.Dimensions{}
	}
	maxH := 180
	if ui.isCompact(gtx) {
		maxH = 120
	}
	visualHeight := min(gtx.Dp(unit.Dp(maxH)), size.Y)
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
	peakCapHeight := max(1, gtx.Dp(2))

	for index := 0; index < 16; index++ {
		x := index * (bandWidth + gap)
		leftHeight := int(float64(graphHeight) * ui.visualLeft[index])
		rightHeight := int(float64(graphHeight) * ui.visualRight[index])

		// Left bar
		fillSegmentedBar(gtx.Ops, image.Rect(x, graphBottom-leftHeight, x+barWidth, graphBottom), ui.palette.ChannelLeft, ui.palette.Background, graphBottom)
		// Left peak cap
		if peakL := ui.peaksLeft[index]; peakL > 0.01 {
			peakY := graphBottom - int(float64(graphHeight)*peakL)
			fillRect(gtx.Ops, image.Rect(x, max(visualTop, peakY-peakCapHeight), x+barWidth, max(visualTop+peakCapHeight, peakY)), ui.palette.LineHot)
		}

		// Right bar
		xRight := x + barWidth + barGap
		fillSegmentedBar(gtx.Ops, image.Rect(xRight, graphBottom-rightHeight, xRight+barWidth, graphBottom), ui.palette.ChannelRight, ui.palette.Background, graphBottom)
		// Right peak cap
		if peakR := ui.peaksRight[index]; peakR > 0.01 {
			peakY := graphBottom - int(float64(graphHeight)*peakR)
			fillRect(gtx.Ops, image.Rect(xRight, max(visualTop, peakY-peakCapHeight), xRight+barWidth, max(visualTop+peakCapHeight, peakY)), ui.palette.LineHot)
		}
	}
	return layout.Dimensions{Size: size}
}

func (ui *UI) layoutChannelLevels(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(5)}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, "L", 8, ui.palette.Muted, false) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return ui.levelBar(gtx, ui.levelLeft, ui.palette.ChannelLeft)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, "R", 8, ui.palette.Muted, false) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return ui.levelBar(gtx, ui.levelRight, ui.palette.ChannelRight)
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
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(2)}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			slider := material.Slider(ui.theme, &ui.seek)
			slider.Color = ui.palette.Accent
			slider.FingerSize = 18
			dimensions := slider.Layout(gtx)
			if ui.seek.Value != before && duration > 0 && (gtx.Now.Sub(ui.lastSeek) >= 80*time.Millisecond || !ui.seek.Dragging()) {
				ui.lastSeek = gtx.Now
				ui.runAction(ui.controller.Seek(float64(ui.seek.Value) * duration))
			}
			return dimensions
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			remaining := formatRemaining(ui.state.Position, duration)
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, "T+ "+formatTime(ui.state.Position), 8, ui.palette.Accent, true)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 0)}
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, "T "+remaining+" / "+formatTime(duration), 8, ui.palette.Muted, false)
				}),
			)
		}),
	)
}

func (ui *UI) layoutMetadata(gtx layout.Context) layout.Dimensions {
	track := ui.currentTrack()
	return ui.panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(6)}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.heading(gtx, "// CONTEXT_INSPECTOR", "ENGINE_SPECS")
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.metadataRow(gtx, "CODEC_DECODER", trackValue(track, func(t player.Track) string { return t.Codec + " (go-mp3 stream)" }))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.metadataRow(gtx, "SRC_SAMPLE_RATE", trackValue(track, func(t player.Track) string { return fmt.Sprintf("%.1f kHz → 48.0 kHz", float64(t.SampleRate)/1000) }))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.metadataRow(gtx, "CHANNELS_OUTPUT", trackValue(track, func(t player.Track) string {
					if t.Channels == 2 {
						return "2 (STEREO S16LE)"
					}
					return "1 (MONO S16LE)"
				}))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.metadataRow(gtx, "DEVICE_BACKEND", "OTO v3 (WASAPI)")
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.metadataRow(gtx, "FILE_SOURCE", trackValue(track, func(t player.Track) string { return filepath.Base(t.Path) }))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.rowBox(gtx, ui.palette.PanelRaised, ui.palette.LineHot, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(6).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(2)}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.label(gtx, "SIGNAL_PIPELINE", 8, ui.palette.Accent, true)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.label(gtx, "MP3 → PCM → SRC 48k → FFT → Oto", 7, ui.palette.Text, false)
							}),
						)
					})
				})
			}),
		)
	})
}

func (ui *UI) layoutTransport(gtx layout.Context) layout.Dimensions {
	compact := ui.isCompact(gtx)
	volW := 130
	if compact {
		volW = 85
	}
	return ui.panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(4)}.Layout(gtx,
			layout.Rigid(ui.layoutTimeline),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(6)}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.button(gtx, &ui.skinsButton, "[ 🎨 SKINS ]", false)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if ui.message == "" {
									status := "READY"
									if ui.state.State == player.StatePlaying {
										status = "PLAYING"
									} else if ui.state.State == player.StatePaused {
										status = "PAUSED"
									}
									return ui.label(gtx, ":: "+status, 8, ui.palette.Muted, false)
								}
								messageColor := ui.palette.Accent
								if strings.HasPrefix(ui.message, "ERR::") {
									messageColor = ui.palette.Danger
								}
								return ui.label(gtx, ui.message, 8, messageColor, false)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						toggle := "[ ▶ PLAY ]"
						if ui.state.State == player.StatePlaying {
							toggle = "[ ⏸ PAUSE ]"
						}
						return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(6)}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &ui.previousButton, "|<<", false) }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &ui.stopButton, "[■]", false) }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &ui.toggleButton, toggle, true) }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &ui.nextButton, ">>|", false) }),
						)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(4)}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, "VOL", 8, ui.palette.Muted, true) }),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(unit.Dp(volW)), gtx.Dp(24)))
									before := ui.volume.Value
									slider := material.Slider(ui.theme, &ui.volume)
									slider.Color = ui.palette.Accent
									slider.FingerSize = 18
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
			}),
		)
	})
}

func formatRemaining(current, duration float64) string {
	rem := duration - current
	if rem < 0 {
		rem = 0
	}
	total := int(rem)
	return fmt.Sprintf("-%d:%02d", total/60, total%60)
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
	content := op.Record(gtx.Ops)
	dimensions := layout.UniformInset(9).Layout(gtx, body)
	contentCall := content.Stop()

	rectangle := image.Rectangle{Max: dimensions.Size}
	fillRect(gtx.Ops, rectangle, background)
	stroke := max(1, int(ui.skin.Metrics.StrokeWidth))
	fillRect(gtx.Ops, image.Rect(0, 0, dimensions.Size.X, stroke), border)
	fillRect(gtx.Ops, image.Rect(0, dimensions.Size.Y-stroke, dimensions.Size.X, dimensions.Size.Y), border)
	fillRect(gtx.Ops, image.Rect(0, 0, stroke, dimensions.Size.Y), border)
	fillRect(gtx.Ops, image.Rect(dimensions.Size.X-stroke, 0, dimensions.Size.X, dimensions.Size.Y), border)
	contentCall.Add(gtx.Ops)
	return dimensions
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

func fillSegmentedBar(ops *op.Ops, rectangle image.Rectangle, fill, mask color.NRGBA, anchorY int) {
	if rectangle.Empty() {
		return
	}
	fillRect(ops, rectangle, fill)
	segStep := 8
	gapHeight := 2
	for y := anchorY - segStep; y > rectangle.Min.Y; y -= segStep {
		gapTop := max(rectangle.Min.Y, y-gapHeight)
		fillRect(ops, image.Rect(rectangle.Min.X, gapTop, rectangle.Max.X, y), mask)
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
