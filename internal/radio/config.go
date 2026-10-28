package radio

import (
	"errors"
	"net/url"
	"strings"
)

const DefaultBaseURL = "https://de1.api.radio-browser.info"
const DefaultIcecastURL = "https://dir.xiph.org/yp.xml"

// Config contains the user-editable directory and recommendation defaults.
type Config struct {
	Provider     string   `json:"provider"`
	BaseURL      string   `json:"baseURL"`
	Styles       []string `json:"suggestedStyles"`
	Limit        int      `json:"limit"`
	Country      string   `json:"countryCode,omitempty"`
	HideBroken   bool     `json:"hideBroken"`
	Personalized bool     `json:"personalized"`
}

func DefaultConfig() Config {
	return Config{Provider: "radio-browser", BaseURL: DefaultBaseURL, Styles: []string{"rock", "pop", "jazz", "classical", "electronic", "ambient", "latin", "blues"}, Limit: 30, HideBroken: true, Personalized: true}
}

func (c Config) Validate() error {
	if c.Provider != "" && c.Provider != "radio-browser" && c.Provider != "icecast" {
		return errors.New("directorio de radios desconocido")
	}
	if c.Provider == "icecast" && c.Country != "" {
		return errors.New("Icecast no ofrece filtro de pais; deja el pais vacio")
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil || !ValidURL(c.BaseURL) || u.RawQuery != "" || u.Fragment != "" || len(c.BaseURL) > 2048 {
		return errors.New("la fuente debe ser una URL HTTP/HTTPS sin consulta ni fragmento")
	}
	if c.Limit < 1 || c.Limit > 30 {
		return errors.New("el limite de radios debe estar entre 1 y 30")
	}
	if len(c.Country) != 0 && (len(c.Country) != 2 || c.Country[0] < 'A' || c.Country[0] > 'Z' || c.Country[1] < 'A' || c.Country[1] > 'Z') {
		return errors.New("usa un codigo de pais de dos letras, como CL, o dejalo vacio")
	}
	if len(c.Styles) == 0 || len(c.Styles) > 32 {
		return errors.New("configura entre 1 y 32 estilos sugeridos")
	}
	seen := map[string]bool{}
	for _, style := range c.Styles {
		if style == "" || len(style) > 80 || strings.TrimSpace(style) != style || seen[style] {
			return errors.New("los estilos deben ser unicos, no vacios y de hasta 80 caracteres")
		}
		seen[style] = true
	}
	return nil
}
