package native

import (
	"context"
	"fmt"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"maps"
	"playerdoit/internal/radio"
	"strings"
)

type radioSettings struct {
	source, styles, country, mirrors widget.Editor
	hideBroken, personalized         widget.Bool
	limit, reset, provider           widget.Clickable
	list                             widget.List
	refreshMirrors                   widget.Clickable
	mirrorResults                    chan mirrorResult
	mirrorCancel                     context.CancelFunc
	mirrorBusy                       bool
	mirrorMessage                    string
}

type mirrorResult struct {
	urls []string
	err  error
}

func (ui *UI) editRadioSettings(c radio.Config) {
	r := &ui.radioSettings
	if r.mirrorCancel != nil {
		r.mirrorCancel()
	}
	r.mirrorResults = nil
	r.mirrorBusy = false
	r.mirrorMessage = "Lista guardada. Actualizar consulta los espejos anunciados actualmente."
	r.source.SetText(c.BaseURL)
	r.mirrors.SetText(strings.Join(c.Mirrors, "\n"))
	r.styles.SetText(strings.Join(c.Styles, ", "))
	r.country.SetText(c.Country)
	r.hideBroken.Value = c.HideBroken
	r.personalized.Value = c.Personalized
}

func (ui *UI) radioSettingsConfig() radio.Config {
	r := &ui.radioSettings
	c := ui.settings.draft.Radio
	c.BaseURL = strings.TrimSpace(r.source.Text())
	c.Mirrors = nil
	for _, mirror := range strings.Fields(r.mirrors.Text()) {
		c.Mirrors = append(c.Mirrors, mirror)
	}
	c.Country = strings.ToUpper(strings.TrimSpace(r.country.Text()))
	c.Styles = nil
	for _, s := range strings.Split(r.styles.Text(), ",") {
		c.Styles = append(c.Styles, strings.ToLower(strings.TrimSpace(s)))
	}
	c.HideBroken = r.hideBroken.Value
	c.Personalized = r.personalized.Value
	return c
}

func (ui *UI) updateRadioSettings(gtx layout.Context) {
	r := &ui.radioSettings
	if r.mirrorResults != nil {
		select {
		case result := <-r.mirrorResults:
			r.mirrorBusy = false
			r.mirrorResults = nil
			if result.err != nil {
				r.mirrorMessage = "No se pudo actualizar: " + result.err.Error() + ". Se conserva la lista."
			} else {
				r.mirrors.SetText(strings.Join(result.urls, "\n"))
				r.mirrorMessage = fmt.Sprintf("DNS anuncia %d espejo(s). No comprueba HTTP. Pulsa Guardar para conservarlos.", len(result.urls))
			}
		default:
		}
	}
	if !ui.settings.open || ui.settings.tab != 5 {
		return
	}
	if ui.settings.draft.Radio.Provider != "icecast" {
		refresh := r.refreshMirrors.Clicked(gtx)
		if refresh && !r.mirrorBusy {
			r.mirrorBusy = true
			r.mirrorMessage = "Consultando espejos anunciados por DNS..."
			r.mirrorResults = make(chan mirrorResult, 1)
			ctx, cancel := context.WithCancel(context.Background())
			r.mirrorCancel = cancel
			results, window := r.mirrorResults, ui.window
			go func() {
				defer cancel()
				urls, err := radio.DiscoverMirrors(ctx)
				results <- mirrorResult{urls, err}
				window.Invalidate()
			}()
		}
	}
	if r.provider.Clicked(gtx) {
		c := ui.radioSettingsConfig()
		if c.Provider == "icecast" {
			c.Provider = "radio-browser"
			c.BaseURL = radio.DefaultBaseURL
		} else {
			c.Provider = "icecast"
			c.BaseURL = radio.DefaultIcecastURL
			c.Country = ""
		}
		ui.settings.draft.Radio = c
		ui.editRadioSettings(c)
	}
	if r.limit.Clicked(gtx) {
		ui.settings.draft.Radio.Limit = ui.settings.draft.Radio.Limit%30 + 10
		if ui.settings.draft.Radio.Limit > 30 {
			ui.settings.draft.Radio.Limit = 10
		}
	}
	if r.reset.Clicked(gtx) {
		ui.settings.draft.Radio = radio.DefaultConfig()
		ui.settings.draft.Styles = nil
		ui.editRadioSettings(ui.settings.draft.Radio)
		ui.settings.message = "Valores de radios restaurados. Pulsa Guardar para aplicarlos."
	}
}

func (ui *UI) layoutRadioSettings(gtx layout.Context) layout.Dimensions {
	r := &ui.radioSettings
	r.source.SingleLine = true
	r.styles.SingleLine = true
	r.country.SingleLine = true
	r.list.Axis = layout.Vertical
	rows := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions {
			name := "Radio Browser"
			if ui.settings.draft.Radio.Provider == "icecast" {
				name = "Icecast"
			}
			return ui.button(gtx, &r.provider, "Directorio: "+name+" (cambiar)", false)
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, "URL de la API Radio Browser o del catalogo XML Icecast", 11, ui.palette.Text, false)
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.discoveryEditor(gtx, &r.source, "https://de1.api.radio-browser.info")
		},
		func(gtx layout.Context) layout.Dimensions {
			if ui.settings.draft.Radio.Provider == "icecast" {
				return layout.Dimensions{}
			}
			return ui.label(gtx, fmt.Sprintf("Máximo %d intentos totales. API principal primero; después los espejos en orden.", radio.MaxSearchAttempts), 11, ui.palette.Text, false)
		},
		func(gtx layout.Context) layout.Dimensions {
			if ui.settings.draft.Radio.Provider == "icecast" {
				return layout.Dimensions{}
			}
			return ui.label(gtx, fmt.Sprintf("Espejos configurados: %d. Una URL por línea (editable).", len(strings.Fields(r.mirrors.Text()))), 11, ui.palette.Text, false)
		},
		func(gtx layout.Context) layout.Dimensions {
			if ui.settings.draft.Radio.Provider == "icecast" {
				return layout.Dimensions{}
			}
			return ui.discoveryEditor(gtx, &r.mirrors, "Sin espejos adicionales")
		},
		func(gtx layout.Context) layout.Dimensions {
			if ui.settings.draft.Radio.Provider == "icecast" {
				return layout.Dimensions{}
			}
			label := "Actualizar espejos disponibles"
			if r.mirrorBusy {
				label = "Actualizando espejos..."
			}
			return ui.button(gtx, &r.refreshMirrors, label, false)
		},
		func(gtx layout.Context) layout.Dimensions {
			if ui.settings.draft.Radio.Provider == "icecast" {
				return layout.Dimensions{}
			}
			return ui.label(gtx, "Fuente: "+radio.MirrorDiscoveryHost+". "+r.mirrorMessage, 11, ui.palette.Muted, false)
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, "Estilos iniciales separados por comas", 11, ui.palette.Text, false)
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.discoveryEditor(gtx, &r.styles, "rock, pop, jazz")
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, "Pais (Radio Browser): CL, AR, ES; vacio para todos / Icecast", 11, ui.palette.Text, false)
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.discoveryEditor(gtx, &r.country, "Todos los paises")
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.settingsCheck(gtx, &r.hideBroken, "Ocultar emisoras caidas (Radio Browser)")
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.settingsCheck(gtx, &r.personalized, "Personalizar sugerencias con mis valoraciones")
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.button(gtx, &r.limit, fmt.Sprintf("Resultados: %d (cambiar)", ui.settings.draft.Radio.Limit), false)
		},
		func(gtx layout.Context) layout.Dimensions {
			return ui.button(gtx, &r.reset, "Restablecer radios y perfil", false)
		},
	}
	return r.list.Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions { return layout.Inset{Bottom: 6}.Layout(gtx, rows[i]) })
}

func (ui *UI) discoveryEditor(gtx layout.Context, editor *widget.Editor, hint string) layout.Dimensions {
	e := material.Editor(ui.theme, editor, hint)
	e.Color = ui.palette.Text
	e.HintColor = ui.palette.Muted
	return e.Layout(gtx)
}

func clonePreferences(p Preferences) Preferences {
	p.Styles = maps.Clone(p.Styles)
	p.Radio.Styles = append([]string(nil), p.Radio.Styles...)
	p.Radio.Mirrors = append([]string(nil), p.Radio.Mirrors...)
	return p
}
