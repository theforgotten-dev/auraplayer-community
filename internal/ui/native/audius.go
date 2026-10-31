package native

import (
	"context"
	"fmt"
	"image"
	"strings"

	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"playerdoit/internal/audius"
)

type audiusResult struct {
	tracks []audius.Track
	id     string
	cover  image.Image
	err    error
	search bool
}

type audiusPanel struct {
	browseTab, playerTab                                      widget.Clickable
	showPlayer                                                bool
	lastTrackID                                               string
	query, api, appName, apiKey                               widget.Editor
	search, clear, settings, terms, privacy, license, nowPage widget.Clickable
	play, page                                                [20]widget.Clickable
	list                                                      widget.List
	tracks                                                    []audius.Track
	covers                                                    map[string]image.Image
	results                                                   chan audiusResult
	coverResults                                              chan audiusResult
	cancel, coverCancel                                       context.CancelFunc
	currentCoverID, selectedID                                string
	busy, showSettings                                        bool
	message                                                   string
}

func (ui *UI) editAudiusSettings(c audius.Config) {
	a := &ui.audiusPanel
	a.api.SetText(c.APIURL)
	a.appName.SetText(c.AppName)
	a.apiKey.SetText(c.APIKey)
}

func (ui *UI) audiusSettingsConfig() audius.Config {
	a := &ui.audiusPanel
	return audius.Config{APIURL: strings.TrimSpace(a.api.Text()), AppName: strings.TrimSpace(a.appName.Text()), APIKey: strings.TrimSpace(a.apiKey.Text())}
}

func (ui *UI) storeAudiusCover(id string, img image.Image) {
	if img == nil {
		return
	}
	a := &ui.audiusPanel
	if a.covers == nil {
		a.covers = map[string]image.Image{}
	}
	if len(a.covers) >= 32 {
		for key := range a.covers {
			if key != a.currentCoverID {
				delete(a.covers, key)
				break
			}
		}
	}
	a.covers[id] = img
}

func (ui *UI) startAudiusSearch() {
	a := &ui.audiusPanel
	config := ui.audiusSettingsConfig()
	if err := config.Validate(); err != nil {
		a.message = err.Error()
		return
	}
	query := strings.TrimSpace(a.query.Text())
	if query == "" || len(query) > 200 {
		a.message = "Escribe un artista o canción (hasta 200 caracteres)."
		return
	}
	if a.cancel != nil {
		a.cancel()
	}
	a.busy = true
	a.showPlayer = false
	a.play = [20]widget.Clickable{}
	a.page = [20]widget.Clickable{}
	a.list.Position = layout.Position{}
	a.tracks = nil
	a.message = "Buscando en Audius..."
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.results = make(chan audiusResult, 24)
	results, window := a.results, ui.window
	go func() {
		defer cancel()
		client := audius.Client{Config: config}
		tracks, err := client.Search(ctx, query)
		select {
		case results <- audiusResult{tracks: tracks, err: err, search: true}:
		case <-ctx.Done():
			return
		}
		window.Invalidate()
		for _, track := range tracks {
			if ctx.Err() != nil {
				return
			}
			if track.ArtworkURL() == "" {
				continue
			}
			cover, err := client.Artwork(ctx, track.ArtworkURL())
			if err != nil {
				continue
			}
			select {
			case results <- audiusResult{id: track.ID, cover: cover}:
			case <-ctx.Done():
				return
			}
			window.Invalidate()
		}
	}()
}

func (ui *UI) receiveAudiusResults() {
	a := &ui.audiusPanel
	for i := 0; i < 24; i++ {
		select {
		case result := <-a.results:
			if result.search {
				a.busy = false
				a.tracks = result.tracks
				if result.err != nil {
					a.message = result.err.Error()
				} else {
					a.message = fmt.Sprintf("%d pistas con acceso público. Reproducir agrega la canción a la cola.", len(result.tracks))
				}
			} else {
				ui.storeAudiusCover(result.id, result.cover)
			}
		default:
			return
		}
	}
}

func (ui *UI) updateAudius(gtx layout.Context) {
	a := &ui.audiusPanel
	ui.receiveAudiusResults()
	select {
	case result := <-a.coverResults:
		ui.storeAudiusCover(result.id, result.cover)
	default:
	}
	if track := ui.currentTrack(); !ui.actionBusy && track != nil && track.Provider == "audius" && track.ProviderID != a.currentCoverID {
		if a.coverCancel != nil {
			a.coverCancel()
		}
		a.currentCoverID = track.ProviderID
		if a.covers[track.ProviderID] == nil && track.ArtworkURL != "" {
			ctx, cancel := context.WithCancel(context.Background())
			a.coverCancel = cancel
			a.coverResults = make(chan audiusResult, 1)
			results, window, id, raw := a.coverResults, ui.window, track.ProviderID, track.ArtworkURL
			go func() {
				defer cancel()
				img, err := (audius.Client{}).Artwork(ctx, raw)
				if err == nil {
					results <- audiusResult{id: id, cover: img}
					window.Invalidate()
				}
			}()
		}
	}
	if a.nowPage.Clicked(gtx) {
		if track := ui.currentTrack(); track != nil && track.Provider == "audius" {
			ui.openAudiusLink(track.Permalink)
		}
	}
	if track := ui.currentTrack(); track != nil && track.ID != a.lastTrackID {
		a.lastTrackID = track.ID
		if track.Provider != "audius" {
			a.showPlayer = true
		}
	}
	if a.browseTab.Clicked(gtx) {
		a.showPlayer = false
	}
	if a.playerTab.Clicked(gtx) {
		a.showPlayer = true
	}
	submit := a.search.Clicked(gtx)
	for {
		event, ok := a.query.Update(gtx)
		if !ok {
			break
		}
		if _, ok := event.(widget.SubmitEvent); ok {
			submit = true
		}
	}
	if submit && !a.busy && !ui.settings.open {
		ui.startAudiusSearch()
	}
	if a.settings.Clicked(gtx) {
		ui.openSettings()
		ui.settings.tab = 6
	}
	if a.terms.Clicked(gtx) {
		ui.openAudiusLink(audius.TermsURL)
	}
	if a.privacy.Clicked(gtx) {
		ui.openAudiusLink(audius.PrivacyURL)
	}
	if a.license.Clicked(gtx) {
		ui.openAudiusLink(audius.MusicLicenseURL)
	}
	if a.clear.Clicked(gtx) && !ui.actionBusy {
		if a.cancel != nil {
			a.cancel()
		}
		if a.coverCancel != nil {
			a.coverCancel()
		}
		a.results = nil
		a.coverResults = nil
		a.tracks = nil
		a.covers = nil
		a.currentCoverID = ""
		a.selectedID = ""
		a.busy = false
		a.message = "Sesión Audius vaciada. Las canciones y carátulas no se guardan en disco."
		ui.runAction(ui.controller.ClearAudius)
	}
	for i, t := range a.tracks {
		if a.page[i].Clicked(gtx) {
			ui.openAudiusLink(t.PageURL())
		}
		if a.play[i].Clicked(gtx) && !ui.actionBusy {
			config, source := ui.audiusSettingsConfig(), t
			a.selectedID = t.ID
			ui.runAction(func() error { return ui.controller.PlayAudius(config, source) })
		}
	}
}

func (ui *UI) layoutAudiusArtwork(gtx layout.Context, id string) layout.Dimensions {
	if img := ui.audiusPanel.covers[id]; img != nil {
		return (widget.Image{Src: paint.NewImageOp(img), Fit: widget.Contain}).Layout(gtx)
	}
	return ui.outlineBox(gtx, ui.palette.Line, func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return ui.label(gtx, "♫", 18, ui.palette.Muted, false) })
	})
}

func (ui *UI) closeAudiusSession() {
	a := &ui.audiusPanel
	if a.cancel != nil {
		a.cancel()
	}
	if a.coverCancel != nil {
		a.coverCancel()
	}
	a.tracks = nil
	a.covers = nil
	a.results = nil
	a.coverResults = nil
}

func (ui *UI) layoutAudiusOptions(gtx layout.Context) layout.Dimensions {
	a := &ui.audiusPanel
	a.api.SingleLine = true
	a.appName.SingleLine = true
	a.apiKey.SingleLine = true
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(8)}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, "Opciones avanzadas de Audius. La búsqueda está en la pantalla principal.", 12, ui.palette.Text, false)
		}),
		layout.Rigid(
			func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, "API HTTPS (incluye /v1). Guardar conserva solo esta configuración.", 11, ui.palette.Text, false)
			}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.discoveryEditor(gtx, &a.api, audius.DefaultAPI) }),
		layout.Rigid(
			func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, "Nombre de aplicación / API Key pública opcional. Nunca pegues un Bearer Token.", 11, ui.palette.Text, false)
			}),
		layout.Rigid(
			func(gtx layout.Context) layout.Dimensions {
				return ui.discoveryEditor(gtx, &a.appName, "AuraPlayerCommunity")
			}),
		layout.Rigid(
			func(gtx layout.Context) layout.Dimensions {
				return ui.discoveryEditor(gtx, &a.apiKey, "API Key pública (opcional)")
			}),
	)
}

func (ui *UI) layoutAudiusSearchBar(gtx layout.Context) layout.Dimensions {
	a := &ui.audiusPanel
	a.query.SingleLine = true
	a.query.Submit = true
	return ui.panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(5)}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle, Gap: gtx.Dp(6)}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, "Buscar música en Audius", 13, ui.palette.Text, true)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.button(gtx, &a.browseTab, "Audius", !a.showPlayer)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.button(gtx, &a.playerTab, "Reproductor", a.showPlayer)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &a.settings, "Opciones", false) }),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle, Gap: gtx.Dp(8)}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return ui.outlineBox(gtx, ui.palette.LineHot, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return ui.discoveryEditor(gtx, &a.query, "Escribe una canción o artista y pulsa Enter")
							})
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						label := "Buscar"
						if a.busy {
							label = "Buscando..."
						}
						return ui.button(gtx, &a.search, label, true)
					}),
				)
			}),
		)
	})
}

func (ui *UI) layoutAudiusResults(gtx layout.Context) layout.Dimensions {
	a := &ui.audiusPanel
	a.list.Axis = layout.Vertical
	rows := []layout.Widget{}
	if len(a.tracks) == 0 {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, "Encuentra tu próxima canción\n\nBusca un artista o un tema en la barra superior. Aquí aparecerán sus canciones con carátulas y el botón Reproducir.", 16, ui.palette.Text, false)
			})
		})
	}
	for i, source := range a.tracks {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle, Gap: gtx.Dp(6)}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(44), gtx.Dp(44)))
					return ui.layoutAudiusArtwork(gtx, source.ID)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.singleLine(gtx, source.Title, 12, ui.palette.Text)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.singleLine(gtx, source.Artist()+" · "+formatTime(float64(source.Duration))+" · "+source.License, 10, ui.palette.Muted)
						}))
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &a.play[i], "Reproducir", false) }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &a.page[i], "Audius ↗", false) }))
		})
	}
	return ui.panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(8)}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, a.message, 11, ui.palette.Muted, false)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return a.list.Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions { return layout.Inset{Bottom: 8}.Layout(gtx, rows[i]) })
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, "Música de sus artistas · Audius recibe la búsqueda y datos de conexión.", 10, ui.palette.Muted, false)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Gap: gtx.Dp(4)}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &a.terms, "Términos", false) }),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &a.privacy, "Privacidad", false) }),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.button(gtx, &a.license, "Licencia musical", false)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return ui.button(gtx, &a.clear, "Vaciar", false) })
					}),
				)
			}),
		)
	})
}

func (ui *UI) layoutListeningColumn(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(8)}.Layout(gtx,
		layout.Rigid(ui.layoutNowListening),
		layout.Flexed(1, ui.layoutQueue),
	)
}

func (ui *UI) layoutNowListening(gtx layout.Context) layout.Dimensions {
	track := ui.currentTrack()
	return ui.panel(gtx, func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, "EN REPRODUCCIÓN", 11, ui.palette.Accent, true)
		})}
		if track == nil {
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, "Selecciona una canción o abre tus archivos.", 12, ui.palette.Muted, false)
			}))
		} else {
			children = append(children,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						size := gtx.Dp(96)
						gtx.Constraints = layout.Exact(image.Pt(size, size))
						return ui.layoutAudiusArtwork(gtx, track.ProviderID)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.singleLine(gtx, track.Title, 14, ui.palette.Text)
				}),
			)
			if track.Provider == "audius" {
				children = append(children,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.singleLine(gtx, track.Artist, 12, ui.palette.Muted)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.button(gtx, &ui.audiusPanel.nowPage, "Ver en Audius ↗", false)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.singleLine(gtx, "Open Music License · "+track.License, 9, ui.palette.Muted)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.singleLine(gtx, track.Copyright, 9, ui.palette.Muted)
					}),
				)
			}
		}
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(6)}.Layout(gtx, children...)
	})
}
