package native

import (
	"errors"
	"net/url"
)

func validateAudiusLink(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || (u.Host != "audius.co" && u.Host != "audius.org") {
		return errors.New("enlace Audius inválido")
	}
	return nil
}

func (ui *UI) openAudiusLink(raw string) {
	if err := validateAudiusLink(raw); err != nil {
		ui.audiusPanel.message = err.Error()
		return
	}
	if err := openExternalURL(raw); err != nil {
		ui.audiusPanel.message = "No se pudo abrir el enlace: " + err.Error()
	}
}
