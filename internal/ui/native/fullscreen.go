package native

import (
	"image"
	"time"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"playerdoit/internal/player"
)

func controlsOpacity(elapsed time.Duration) float32 {
	const delay = 2 * time.Second
	const fade = 300 * time.Millisecond
	if elapsed <= delay {
		return 1
	}
	if elapsed >= delay+fade {
		return 0
	}
	return 1 - float32(elapsed-delay)/float32(fade)
}

func (ui *UI) layoutFullscreenVideo(gtx layout.Context) layout.Dimensions {
	alpha := float32(1)
	if ui.state.State == player.StatePlaying && !ui.fullscreenHover && !ui.settings.open && ui.preferences.AutoHide {
		alpha = controlsOpacity(gtx.Now.Sub(ui.fullscreenActivity) - time.Duration(ui.preferences.HideDelay-2)*time.Second)
	}
	if alpha > 0 && ui.state.State == player.StatePlaying {
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 60)})
	}
	dims := layout.Stack{}.Layout(gtx,
		layout.Expanded(ui.layoutVideoFrame),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if alpha == 0 {
				return layout.Dimensions{}
			}
			defer paint.PushOpacity(gtx.Ops, alpha).Pop()
			gtx.Constraints.Min = gtx.Constraints.Max
			return layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: 20, Right: 20, Bottom: 20}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					width := min(gtx.Constraints.Max.X, gtx.Dp(1100))
					gtx.Constraints = layout.Exact(image.Pt(width, gtx.Dp(124)))
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.button(gtx, &ui.settings.openButton, "[ CONFIG ]", false)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.button(gtx, &ui.fullscreenButton, "[ EXIT · ESC ]", false)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.button(gtx, &ui.subtitleButton, "[ + SUBS ]", false)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									text := "[ CC OFF ]"
									if ui.state.SubtitlesVisible {
										text = "[ CC ON ]"
									}
									return ui.button(gtx, &ui.subtitleToggle, text, ui.state.SubtitlesVisible)
								}),
							)
						}),
						layout.Flexed(1, ui.layoutTransport),
					)
				})
			})
		}),
	)
	// This transparent input layer observes motion without blocking the
	// floating controls or the video's double-click gesture underneath it.
	defer clip.Rect{Max: dims.Size}.Push(gtx.Ops).Pop()
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, &ui.fullscreenPointer)
	if alpha == 0 {
		pointer.CursorNone.Add(gtx.Ops)
	}
	return dims
}
