package native

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type transportIcon uint8

const (
	iconPrevious transportIcon = iota
	iconStop
	iconPlay
	iconPause
	iconNext
)

// Transport symbols are geometry, independent of text encoding and font glyphs.
func drawTransportIcon(gtx layout.Context, icon transportIcon, foreground color.NRGBA) layout.Dimensions {
	size := gtx.Dp(18)
	scale := float32(size) / 24
	rect := func(x0, y0, x1, y1 float32) {
		paint.FillShape(gtx.Ops, foreground, clip.Rect(image.Rect(int(x0*scale), int(y0*scale), int(x1*scale), int(y1*scale))).Op())
	}
	triangle := func(a, b, c f32.Point) {
		var path clip.Path
		path.Begin(gtx.Ops)
		path.MoveTo(a.Mul(scale))
		path.LineTo(b.Mul(scale))
		path.LineTo(c.Mul(scale))
		path.Close()
		paint.FillShape(gtx.Ops, foreground, clip.Outline{Path: path.End()}.Op())
	}
	switch icon {
	case iconPlay:
		triangle(f32.Pt(6, 3), f32.Pt(21, 12), f32.Pt(6, 21))
	case iconPause:
		rect(5, 3, 10, 21)
		rect(14, 3, 19, 21)
	case iconStop:
		rect(4, 4, 20, 20)
	case iconPrevious:
		rect(3, 4, 6, 20)
		triangle(f32.Pt(20, 4), f32.Pt(7, 12), f32.Pt(20, 20))
	case iconNext:
		rect(18, 4, 21, 20)
		triangle(f32.Pt(4, 4), f32.Pt(17, 12), f32.Pt(4, 20))
	}
	return layout.Dimensions{Size: image.Pt(size, size)}
}

func (ui *UI) transportButton(gtx layout.Context, button *widget.Clickable, icon transportIcon, caption, label string, primary bool) layout.Dimensions {
	style := material.ButtonLayout(ui.theme, button)
	style.CornerRadius = unit.Dp(ui.skin.Metrics.CornerRadius)
	style.Background = ui.palette.PanelRaised
	foreground := ui.palette.Text
	if primary {
		style.Background = ui.palette.Accent
		foreground = ui.palette.Background
	}
	return style.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		semantic.LabelOp(label).Add(gtx.Ops)
		return layout.Inset{Top: 9, Bottom: 9, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle, Gap: gtx.Dp(6)}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return drawTransportIcon(gtx, icon, foreground) }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if caption == "" {
						return layout.Dimensions{}
					}
					return ui.label(gtx, caption, 10, foreground, true)
				}),
			)
		})
	})
}
