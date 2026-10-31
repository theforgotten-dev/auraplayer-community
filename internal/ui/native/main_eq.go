package native

import (
	"fmt"
	"math"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"playerdoit/internal/dsp"
)

// Separate widgets keep the main equalizer independent of the modal editor.
type mainEQPanel struct {
	enabled                 widget.Bool
	bands                   [10]widget.Float
	preamp                  widget.Float
	flat, voice, bass, save widget.Clickable
	applied, target         dsp.ToneConfig
	pending, saving         bool
	nextPreview             time.Time
	message                 string
}

func (ui *UI) editMainEQ(c dsp.ToneConfig) {
	t := &ui.mainEQ
	t.enabled.Value = c.Enabled
	t.preamp.Value = float32((c.PreampDB + 24) / 36)
	for i, g := range c.GainsDB {
		t.bands[i].Value = float32((g + 12) / 24)
	}
	t.applied = c
}

func (ui *UI) mainEQConfig() dsp.ToneConfig {
	t := &ui.mainEQ
	c := dsp.ToneConfig{Enabled: t.enabled.Value, PreampDB: math.Round((float64(t.preamp.Value)*36-24)*10) / 10}
	for i := range c.GainsDB {
		c.GainsDB[i] = math.Round((float64(t.bands[i].Value)*24-12)*10) / 10
	}
	return c
}

func (ui *UI) updateMainEQ(gtx layout.Context) {
	t := &ui.mainEQ
	if t.pending && !ui.actionBusy {
		if ui.state.Tone == t.target {
			ui.preferences.Tone = t.target
			t.applied = t.target
			if t.saving && ui.message == "" {
				t.message = "Ecualizador guardado."
			}
		} else {
			ui.editMainEQ(ui.state.Tone)
		}
		if ui.message != "" {
			t.message = ui.message
		}
		t.pending = false
		t.saving = false
	}
	if ui.settings.open {
		return
	}
	if t.flat.Clicked(gtx) {
		ui.editMainEQ(dsp.ToneConfig{Enabled: true})
		t.applied = ui.preferences.Tone
	}
	if t.voice.Clicked(gtx) {
		ui.editMainEQ(dsp.ToneConfig{Enabled: true, GainsDB: [10]float64{-10, -8, -5, -2, 0, 2, 3, 2, -3, -6}})
		t.applied = ui.preferences.Tone
	}
	if t.bass.Clicked(gtx) {
		ui.editMainEQ(dsp.ToneConfig{Enabled: true, GainsDB: [10]float64{4, 5, 3, 1, 0, 0, 0, 0, 0, 0}})
		t.applied = ui.preferences.Tone
	}
	config := ui.mainEQConfig()
	if t.save.Clicked(gtx) && !ui.actionBusy {
		p := clonePreferences(ui.preferences)
		p.Tone = config
		path, err := preferencesPath()
		if err != nil {
			t.message = err.Error()
			return
		}
		t.target = config
		t.pending = true
		t.saving = true
		ui.runAction(func() error {
			if err := ui.controller.ConfigureTone(config); err != nil {
				return err
			}
			return writePreferences(path, p)
		})
		return
	}
	if config != t.applied && !ui.actionBusy && !gtx.Now.Before(t.nextPreview) {
		t.target = config
		t.pending = true
		t.nextPreview = gtx.Now.Add(80 * time.Millisecond)
		t.message = "Cambios en vivo. Guardar conserva el ajuste."
		ui.runAction(func() error { return ui.controller.ConfigureTone(config) })
	}
	if config != t.applied {
		gtx.Execute(op.InvalidateCmd{At: t.nextPreview})
	}
}

func (ui *UI) layoutMainEQ(gtx layout.Context) layout.Dimensions {
	t := &ui.mainEQ
	if ui.settings.open {
		return ui.panel(gtx, func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, "Ecualizador\nCierra Configuración para ajustar el sonido aquí.", 12, ui.palette.Muted, false)
		})
	}
	c := ui.mainEQConfig()
	return ui.panel(gtx, func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, "Ecualizador", 14, ui.palette.Text, true)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.settingsCheck(gtx, &t.enabled, "Activo") }))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Gap: gtx.Dp(4)}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &t.flat, "Plano", false) }),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &t.voice, "Voz", false) }),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &t.bass, "Graves", false) }))
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				labels := [10]string{"31", "63", "125", "250", "500", "1k", "2k", "4k", "8k", "16k"}
				bands := make([]layout.FlexChild, 0, 10)
				for i := range t.bands {
					bands = append(bands, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return ui.faderColumn(gtx, &t.bands[i], labels[i], c.GainsDB[i])
					}))
				}
				return layout.Flex{}.Layout(gtx, bands...)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, fmt.Sprintf("Previo: %+.1f dB", c.PreampDB), 11, ui.palette.Text, false)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				s := material.Slider(ui.theme, &t.preamp)
				s.Color = ui.palette.Accent
				return s.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &t.save, "Guardar EQ", false) }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, t.message, 10, ui.palette.Muted, false)
			}),
		}
		if track := ui.currentTrack(); track == nil || !track.Video {
			children = append(children, layout.Flexed(.8, ui.layoutSpectrum))
		}
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(5)}.Layout(gtx, children...)
	})
}
