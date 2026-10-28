package native

import (
	"fmt"
	"image"
	"math"
	"time"

	"gioui.org/f32"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"playerdoit/internal/dsp"
)

type tonePanel struct {
	open                           bool
	enter, back, flat, voice, bass widget.Clickable
	enabled                        widget.Bool
	bands                          [10]widget.Float
	preamp                         widget.Float
	applied                        dsp.ToneConfig
	previewDirty                   bool
	nextPreview                    time.Time
	curveConfig                    dsp.ToneConfig
	curve                          [128]float64
	curveValid                     bool
	effectivePreamp                float64
}

func (ui *UI) editTone(c dsp.ToneConfig) {
	t := &ui.tonePanel
	t.enabled.Value = c.Enabled
	t.preamp.Value = float32((c.PreampDB + 24) / 36)
	for i, g := range c.GainsDB {
		t.bands[i].Value = float32((g + 12) / 24)
	}
}

func (ui *UI) toneConfig() dsp.ToneConfig {
	t := &ui.tonePanel
	c := dsp.ToneConfig{Enabled: t.enabled.Value, PreampDB: math.Round((float64(t.preamp.Value)*36-24)*10) / 10}
	for i := range c.GainsDB {
		c.GainsDB[i] = math.Round((float64(t.bands[i].Value)*24-12)*10) / 10
	}
	return c
}

func (ui *UI) updateTone(gtx layout.Context) {
	t := &ui.tonePanel
	if !ui.settings.open {
		if t.previewDirty && !ui.actionBusy {
			config := ui.preferences.Tone
			ui.runAction(func() error { return ui.controller.ConfigureTone(config) })
			t.previewDirty = false
			t.applied = config
		}
		return
	}
	if ui.settings.tab == 0 {
		if t.enter.Clicked(gtx) {
			t.open = true
		}
		if t.back.Clicked(gtx) {
			t.open = false
		}
		if t.flat.Clicked(gtx) {
			ui.editTone(dsp.ToneConfig{Enabled: true})
		}
		if t.voice.Clicked(gtx) {
			ui.editTone(dsp.ToneConfig{Enabled: true, GainsDB: [10]float64{-10, -8, -5, -2, 0, 2, 3, 2, -3, -6}})
		}
		if t.bass.Clicked(gtx) {
			ui.editTone(dsp.ToneConfig{Enabled: true, GainsDB: [10]float64{4, 5, 3, 1, 0, 0, 0, 0, 0, 0}})
		}
	}
	config := ui.toneConfig()
	if config != t.applied && !ui.actionBusy && !gtx.Now.Before(t.nextPreview) {
		t.applied = config
		t.previewDirty = true
		t.nextPreview = gtx.Now.Add(80 * time.Millisecond)
		ui.runAction(func() error { return ui.controller.ConfigureTone(config) })
	}
	if config != t.applied {
		gtx.Execute(op.InvalidateCmd{At: t.nextPreview})
	}
}

// The native gesture control uses geometry rather than font glyphs.
func (ui *UI) verticalFader(gtx layout.Context, value *widget.Float, label string) layout.Dimensions {
	size := gtx.Constraints.Max
	if size.X <= 0 || size.Y < 20 {
		return layout.Dimensions{Size: size}
	}
	semantic.LabelOp(label).Add(gtx.Ops)
	padding := gtx.Dp(8)
	height := max(1, size.Y-2*padding)
	x := size.X / 2
	fillRect(gtx.Ops, image.Rect(x-2, padding, x+2, padding+height), ui.palette.Background)
	for i := 0; i <= 4; i++ {
		y := padding + height*i/4
		col := ui.palette.Line
		if i == 2 {
			col = ui.palette.Muted
		}
		fillRect(gtx.Ops, image.Rect(x-10, y, x+10, y+1), col)
	}
	offset := op.Offset(image.Pt(0, padding)).Push(gtx.Ops)
	input := gtx
	input.Constraints = layout.Exact(image.Pt(size.X, height))
	value.Layout(input, layout.Vertical, unit.Dp(8))
	offset.Pop()
	y := padding + int((1-value.Value)*float32(height))
	half := min(gtx.Dp(13), max(4, (size.X-6)/2))
	fillRect(gtx.Ops, image.Rect(x-half, y-5, x+half, y+6), ui.palette.Accent)
	fillRect(gtx.Ops, image.Rect(x-half+3, y, x+half-3, y+1), ui.palette.Background)
	return layout.Dimensions{Size: size}
}

func (ui *UI) faderColumn(gtx layout.Context, value *widget.Float, title string, db float64) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, title, 10, ui.palette.Text, true) })
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return ui.verticalFader(gtx, value, title) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, fmt.Sprintf("%+.1f", db), 10, ui.palette.Accent, true)
			})
		}),
	)
}

func (ui *UI) layoutTone(gtx layout.Context) layout.Dimensions {
	t := &ui.tonePanel
	c := ui.toneConfig()
	if !t.curveValid || t.curveConfig != c {
		t.curveConfig = c
		t.curveValid = true
		t.curve = dsp.ToneCurve(c)
		t.effectivePreamp = c.EffectivePreamp()
	}
	high, low := c.AutomaticCutoffs()
	cutoff := func(hz float64) string {
		if hz == 0 {
			return "OFF"
		}
		return fmt.Sprintf("%.0f Hz", hz)
	}
	graphHeight := min(gtx.Dp(78), max(24, gtx.Constraints.Max.Y/4))
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(5)}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle, Gap: gtx.Dp(6)}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &t.back, "Volumen", false) }),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return ui.settingsCheck(gtx, &t.enabled, "EQ activo") }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &t.flat, "Plano", false) }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &t.voice, "Voz", false) }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &t.bass, "Graves", false) }),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, graphHeight))
			return ui.drawToneCurve(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			children := []layout.FlexChild{layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(28)
				gtx.Constraints.Max.X = gtx.Dp(28)
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, "dB", 9, ui.palette.Muted, false) }),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceBetween}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, "+12", 9, ui.palette.Muted, false) }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, "0", 9, ui.palette.Text, true) }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, "-12", 9, ui.palette.Muted, false) }))
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, " ", 10, ui.palette.Muted, false) }))
			})}
			labels := [10]string{"31", "63", "125", "250", "500", "1k", "2k", "4k", "8k", "16k"}
			for i := range t.bands {
				children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return ui.faderColumn(gtx, &t.bands[i], labels[i], c.GainsDB[i])
				}))
			}
			children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return ui.faderColumn(gtx, &t.preamp, "PRE", c.PreampDB) }))
			return layout.Flex{Gap: gtx.Dp(3)}.Layout(gtx, children...)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, fmt.Sprintf("Cortes auto: PA %s · PB %s | Previo efectivo %+.1f dB", cutoff(high), cutoff(low), t.effectivePreamp), 10, ui.palette.Muted, false)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, "Preescucha en vivo. Guardar conserva; Cerrar restaura el sonido anterior.", 10, ui.palette.Muted, false)
		}),
	)
}

func (ui *UI) drawToneCurve(gtx layout.Context) layout.Dimensions {
	size := gtx.Constraints.Max
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	fillRect(gtx.Ops, image.Rectangle{Max: size}, ui.palette.Background)
	const top = 12
	height := max(1, size.Y-top-4)
	yFor := func(db float64) float32 {
		return float32(top) + float32((12-math.Max(-24, math.Min(12, db)))/36)*float32(height)
	}
	for _, db := range []float64{12, 0, -12, -24} {
		y := int(yFor(db))
		fillRect(gtx.Ops, image.Rect(0, y, size.X, y+1), ui.palette.Line)
	}
	for _, hz := range dsp.ToneFrequencies() {
		x := int(math.Log(hz/20) / math.Log(1000) * float64(size.X-1))
		fillRect(gtx.Ops, image.Rect(x, top, x+1, size.Y), ui.palette.Line)
	}
	var path clip.Path
	path.Begin(gtx.Ops)
	for i, db := range ui.tonePanel.curve {
		point := f32.Pt(float32(i)*float32(size.X-1)/127, yFor(db))
		if i == 0 {
			path.MoveTo(point)
		} else {
			path.LineTo(point)
		}
	}
	paint.FillShape(gtx.Ops, ui.palette.Accent, clip.Stroke{Path: path.End(), Width: 1.6}.Op())
	label := "Respuesta calculada · 20 Hz – 20 kHz"
	if !ui.tonePanel.enabled.Value {
		label = "Bypass · respuesta plana"
	}
	ui.label(gtx, label, 9, ui.palette.Muted, false)
	return layout.Dimensions{Size: size}
}
