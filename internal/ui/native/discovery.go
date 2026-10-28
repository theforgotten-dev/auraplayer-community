package native

import (
	"context"
	"gioui.org/layout"
	"gioui.org/widget"
	"maps"
	"playerdoit/internal/radio"
	"strings"
)

type discoveryResult struct {
	stations []radio.Station
	err      error
}
type discoveryPanel struct {
	query                 widget.Editor
	search, like, dislike widget.Clickable
	styles                [4]widget.Clickable
	tune                  [30]widget.Clickable
	list                  widget.List
	results               chan discoveryResult
	stations              []radio.Station
	busy                  bool
	message               string
}

func (ui *UI) updateDiscovery(gtx layout.Context) {
	d := &ui.discovery
	if d.results != nil {
		select {
		case result := <-d.results:
			d.busy = false
			d.stations = result.stations
			if result.err != nil {
				d.message = result.err.Error()
			} else {
				d.message = "Selecciona Sintonizar. Cada emisora reemplaza la radio anterior."
			}
		default:
		}
	}
	if !ui.settings.open || ui.settings.tab != 4 {
		return
	}
	styles := radio.SuggestedStyles(ui.radioSettingsConfig(), ui.settings.draft.Styles)
	for i := range d.styles {
		if d.styles[i].Clicked(gtx) && i < len(styles) {
			d.query.SetText(styles[i])
		}
	}
	delta := 0
	if d.like.Clicked(gtx) {
		delta = 1
	}
	if d.dislike.Clicked(gtx) {
		delta = -1
	}
	if delta != 0 {
		tag := strings.ToLower(strings.TrimSpace(d.query.Text()))
		if tag != "" && len(tag) <= 80 {
			if _, exists := ui.settings.draft.Styles[tag]; !exists && len(ui.settings.draft.Styles) >= 100 {
				d.message = "El perfil admite hasta 100 estilos"
				return
			}
			if ui.settings.draft.Styles == nil {
				ui.settings.draft.Styles = map[string]int{}
			}
			value := ui.settings.draft.Styles[tag] + delta
			value = max(-10, min(10, value))
			ui.settings.draft.Styles[tag] = value
			d.message = "Perfil actualizado. Pulsa Guardar para conservarlo."
			if ui.radioSettings.personalized.Value {
				radio.Rank(d.stations, ui.settings.draft.Styles)
			}
		}
	}
	if d.search.Clicked(gtx) && !d.busy {
		config := ui.radioSettingsConfig()
		if err := config.Validate(); err != nil {
			d.message = err.Error()
			return
		}
		d.busy = true
		d.message = "Buscando radios..."
		if d.results == nil {
			d.results = make(chan discoveryResult, 1)
		}
		query := d.query.Text()
		weights := maps.Clone(ui.settings.draft.Styles)
		results := d.results
		window := ui.window
		go func() {
			stations, err := (radio.Client{Config: &config}).Search(context.Background(), query)
			if config.Personalized {
				radio.Rank(stations, weights)
			}
			results <- discoveryResult{stations, err}
			window.Invalidate()
		}()
	}
	for i := range d.tune {
		if d.tune[i].Clicked(gtx) && i < len(d.stations) {
			station := d.stations[i]
			ui.runAction(func() error { return ui.controller.Tune(station) })
		}
	}
}

func (ui *UI) layoutDiscovery(gtx layout.Context) layout.Dimensions {
	d := &ui.discovery
	d.query.SingleLine = true
	d.list.Axis = layout.Vertical
	styles := radio.SuggestedStyles(ui.radioSettingsConfig(), ui.settings.draft.Styles)
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(5)}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, "Estilo musical o tema (ej.: jazz, rock, news). Perfil guardado en este equipo.", 10, ui.palette.Text, false)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.discoveryEditor(gtx, &d.query, "Estilo o tema")
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			children := []layout.FlexChild{}
			for i := range d.styles {
				if i >= len(styles) {
					break
				}
				children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &d.styles[i], styles[i], false) }))
			}
			return layout.Flex{}.Layout(gtx, children...)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &d.search, "Buscar radios", true) }),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &d.like, "Me interesa +", false) }),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &d.dislike, "Menos de esto", false) }))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			message := d.message
			if ui.message != "" {
				message = ui.message
			}
			return ui.label(gtx, message, 10, ui.palette.Muted, false)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return d.list.Layout(gtx, len(d.stations), func(gtx layout.Context, i int) layout.Dimensions {
				s := d.stations[i]
				return ui.button(gtx, &d.tune[i], "Sintonizar · "+s.Name+" · "+s.Country, false)
			})
		}),
	)
}
