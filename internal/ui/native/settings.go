package native

import (
	"errors"
	"fmt"
	"image"
	"math"
	"os"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type settingsPanel struct {
	open                                             bool
	tab                                              int
	draft                                            Preferences
	message                                          string
	openButton, closeButton, saveButton, resetButton widget.Clickable
	tabs                                             [6]widget.Clickable
	language, delay, fullscreen, skins, compact      widget.Clickable
	volume                                           widget.Float
	muted, subtitles, autoHide                       widget.Bool
}

func (ui *UI) loadPreferences() {
	ui.preferences = defaultPreferences()
	path, err := preferencesPath()
	if err == nil {
		p, readErr := readPreferences(path)
		if readErr == nil {
			ui.preferences = p
		} else if !errors.Is(readErr, os.ErrNotExist) {
			ui.message = "ERR:: preferencias: " + readErr.Error()
		}
	} else {
		ui.message = "ERR:: preferencias: " + err.Error()
	}
	volume := ui.preferences.Volume
	if ui.preferences.Muted {
		volume = 0
	}
	if err := ui.controller.SetVolume(volume); err != nil {
		ui.message = "ERR:: " + err.Error()
	}
	if err := ui.controller.ConfigureSubtitles(ui.preferences.Subtitles, ui.preferences.Language); err != nil {
		ui.message = "ERR:: " + err.Error()
	}
	if err := ui.controller.ConfigureTone(ui.preferences.Tone); err != nil {
		ui.message = "ERR:: " + err.Error()
	}
}

func (ui *UI) editPreferences(p Preferences) {
	s := &ui.settings
	s.draft = clonePreferences(p)
	ui.editRadioSettings(p.Radio)
	ui.editTone(p.Tone)
	s.volume.Value = float32(p.Volume)
	s.muted.Value = p.Muted
	s.subtitles.Value = p.Subtitles
	s.autoHide.Value = p.AutoHide
}

func (ui *UI) openSettings() {
	if ui.actionBusy || ui.dialogBusy || ui.settings.open {
		return
	}
	p := ui.preferences
	if volume := ui.controller.Snapshot().Volume; volume > 0 {
		p.Volume = volume
		p.Muted = false
	}
	ui.editPreferences(p)
	ui.tonePanel.applied = ui.preferences.Tone
	ui.tonePanel.previewDirty = false
	ui.settings.message = ""
	ui.settings.open = true
	ui.showSkins = false
	ui.fullscreenActivity = time.Now()
}

func (ui *UI) saveSettings() {
	if ui.actionBusy {
		return
	}
	s := &ui.settings
	p := clonePreferences(s.draft)
	p.Radio = ui.radioSettingsConfig()
	p.Tone = ui.toneConfig()
	p.Volume = math.Round(float64(s.volume.Value)*100) / 100
	p.Muted = s.muted.Value
	p.Subtitles = s.subtitles.Value
	p.AutoHide = s.autoHide.Value
	if err := p.validate(); err != nil {
		s.message = err.Error()
		return
	}
	volume := p.Volume
	if p.Muted {
		volume = 0
	}
	if err := ui.controller.ConfigureSubtitles(p.Subtitles, p.Language); err != nil {
		s.message = err.Error()
		return
	}
	if err := ui.controller.SetVolume(volume); err != nil {
		s.message = err.Error()
		return
	}
	if err := ui.controller.ConfigureTone(p.Tone); err != nil {
		s.message = err.Error()
		return
	}
	ui.preferences = p
	path, err := preferencesPath()
	if err == nil {
		err = writePreferences(path, p)
	}
	if err != nil {
		s.message = "Aplicado, pero no se pudo guardar: " + err.Error()
		return
	}
	s.open = false
	ui.fullscreenActivity = time.Now()
	ui.refreshState()
}

func (ui *UI) updateSettings(gtx layout.Context) {
	s := &ui.settings
	if s.openButton.Clicked(gtx) {
		ui.openSettings()
	}
	if !s.open {
		return
	}
	if s.closeButton.Clicked(gtx) {
		s.open = false
		ui.fullscreenActivity = gtx.Now
	}
	if s.saveButton.Clicked(gtx) {
		ui.saveSettings()
	}
	if s.resetButton.Clicked(gtx) {
		ui.editPreferences(defaultPreferences())
		s.message = "Pulsa Guardar para aplicar los valores iniciales."
	}
	for i := range s.tabs {
		if s.tabs[i].Clicked(gtx) {
			s.tab = i
		}
	}
	if s.language.Clicked(gtx) {
		values := []string{"spa", "eng", "fra", "auto"}
		for i, v := range values {
			if s.draft.Language == v {
				s.draft.Language = values[(i+1)%len(values)]
				break
			}
		}
	}
	if s.delay.Clicked(gtx) {
		s.draft.HideDelay = s.draft.HideDelay%5 + 1
	}
	if s.fullscreen.Clicked(gtx) {
		ui.setFullscreen(!ui.fullscreen)
	}
	if s.skins.Clicked(gtx) {
		s.open = false
		ui.showSkins = true
	}
	if s.compact.Clicked(gtx) {
		s.open = false
		ui.compactButton.Click()
	}
}

func (ui *UI) settingsCheck(gtx layout.Context, value *widget.Bool, label string) layout.Dimensions {
	c := material.CheckBox(ui.theme, value, label)
	c.Color = ui.palette.Text
	c.IconColor = ui.palette.Accent
	return c.Layout(gtx)
}

func (ui *UI) layoutSettings(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		width, height := 660, 390
		if ui.settings.tab == 0 && ui.tonePanel.open {
			width, height = 960, 620
		}
		gtx.Constraints = layout.Exact(image.Pt(min(gtx.Dp(unit.Dp(width)), gtx.Constraints.Max.X-24), min(gtx.Dp(unit.Dp(height)), gtx.Constraints.Max.Y-24)))
		return ui.panel(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(8)}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, "Configuración · Preferencias básicas", 16, ui.palette.Text, true)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.button(gtx, &ui.settings.closeButton, "[ Cerrar ]", false)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					var tabs []layout.FlexChild
					for i, name := range []string{"Audio", "Video", "Subtítulos", "Interfaz", "Descubrir", "Radios"} {
						tabs = append(tabs, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return ui.button(gtx, &ui.settings.tabs[i], name, ui.settings.tab == i)
						}))
					}
					return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(6)}.Layout(gtx, tabs...)
				}),
				layout.Flexed(1, ui.layoutSettingsBody),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					text := ui.settings.message
					if text == "" {
						text = "Guardar aplica los ajustes y los conserva para el próximo inicio."
					}
					return ui.label(gtx, text, 10, ui.palette.Muted, false)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.button(gtx, &ui.settings.resetButton, "Restablecer", false)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.button(gtx, &ui.settings.saveButton, "Guardar", true)
						}),
					)
				}),
			)
		})
	})
}

func (ui *UI) layoutSettingsBody(gtx layout.Context) layout.Dimensions {
	s := &ui.settings
	var rows []layout.FlexChild
	label := func(text string) layout.FlexChild {
		return layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, text, 11, ui.palette.Text, false) })
	}
	switch s.tab {
	case 5:
		return ui.layoutRadioSettings(gtx)
	case 4:
		return ui.layoutDiscovery(gtx)
	case 0:
		if ui.tonePanel.open {
			return ui.layoutTone(gtx)
		}
		rows = []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.button(gtx, &ui.tonePanel.enter, "Ecualizador y filtros", true)
			}),
			label(fmt.Sprintf("Volumen inicial: %.0f %%", s.volume.Value*100)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				slider := material.Slider(ui.theme, &s.volume)
				slider.Color = ui.palette.Accent
				return slider.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.settingsCheck(gtx, &s.muted, "Silenciar el audio")
			}),
			label("Se aplica al archivo actual y a los siguientes."),
		}
	case 1:
		rows = []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.button(gtx, &s.fullscreen, "Alternar pantalla completa · F11", false)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.settingsCheck(gtx, &s.autoHide, "Ocultar controles automáticamente en pantalla completa")
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.button(gtx, &s.delay, fmt.Sprintf("Ocultar después de %d segundos", s.draft.HideDelay), false)
			}),
			label("Esc vuelve al modo de ventana. La imagen conserva su proporción."),
		}
	case 2:
		languages := map[string]string{"spa": "Español", "eng": "Inglés", "fra": "Francés", "auto": "Automático"}
		rows = []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.settingsCheck(gtx, &s.subtitles, "Mostrar subtítulos al reproducir videos")
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.button(gtx, &s.language, "Idioma preferido: "+languages[s.draft.Language], false)
			}),
			label("La preferencia se aplica cuando existe una pista de ese idioma."),
			label("Usa + SUBS para agregar un archivo SRT, ASS, SSA o VTT."),
		}
	case 3:
		rows = []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.button(gtx, &s.skins, "Cambiar apariencia / skin", false)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.button(gtx, &s.compact, "Alternar ventana compacta", false)
			}),
			label("Ctrl+P abre estas preferencias. Esc cierra el panel."),
			label("F11 o Alt+Enter alternan pantalla completa."),
		}
	}
	return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(10)}.Layout(gtx, rows...)
	})
}
