package native

import (
	"errors"
	"fmt"
	"image/color"
	"os"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"gioui.org/x/explorer"
	"playerdoit/internal/video"
)

func (ui *UI) layoutVideo(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(6)}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.button(gtx, &ui.subtitleButton, "[ + SUBS ]", false)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.button(gtx, &ui.fullscreenButton, "[ F11 ]", false)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					label := "[ CC OFF ]"
					if ui.state.SubtitlesVisible {
						label = "[ CC ON ]"
					}
					return ui.button(gtx, &ui.subtitleToggle, label, ui.state.SubtitlesVisible)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					label := fmt.Sprintf("[ SUBS: %d ]", len(ui.state.SubtitleTracks))
					for _, track := range ui.state.SubtitleTracks {
						if track.Selected {
							label = "[ " + track.Title + " ]"
							break
						}
					}
					return ui.button(gtx, &ui.subtitleTrack, label, false)
				}),
			)
		}),
		layout.Flexed(1, ui.layoutVideoFrame),
	)
}

func (ui *UI) layoutVideoFrame(gtx layout.Context) layout.Dimensions {
	return ui.videoClick.Layout(gtx, ui.drawVideoFrame)
}

func (ui *UI) drawVideoFrame(gtx layout.Context) layout.Dimensions {
	return ui.outlineBox(gtx, ui.palette.Line, func(gtx layout.Context) layout.Dimensions {
		defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, color.NRGBA{A: 255})
		if !ui.actionBusy {
			if frame := ui.controller.VideoFrame(); frame != nil {
				return (widget.Image{Src: paint.NewImageOp(frame), Fit: widget.Contain, Position: layout.Center}).Layout(gtx)
			}
		}
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, "VIDEO · PLAY TO LOAD", 12, ui.palette.Text, false)
		})
	})
}

func (ui *UI) chooseSubtitles() {
	track := ui.currentTrack()
	if track == nil || !track.Video {
		return
	}
	id := track.ID
	ui.dialogBusy = true
	go func() {
		reader, err := ui.explorer.ChooseFile(video.SubtitleExtensions()...)
		if errors.Is(err, explorer.ErrUserDecline) {
			err = nil
		}
		path := ""
		if reader != nil {
			if file, ok := reader.(*os.File); ok {
				path = file.Name()
			} else if err == nil {
				err = errors.New("subtitle selection did not return a local file")
			}
			_ = reader.Close()
		}
		ui.fileResults <- fileResult{subtitle: path, subtitleFor: id, err: err}
		ui.window.Invalidate()
	}()
}
