package native

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"playerdoit/internal/audius"
	"playerdoit/internal/dsp"
	"playerdoit/internal/radio"
)

type Preferences struct {
	Audius               audius.Config  `json:"audius"`
	SpectrumRGB          bool           `json:"spectrumRGB"`
	SpectrumCycleSeconds int            `json:"spectrumCycleSeconds"`
	Tone                 dsp.ToneConfig `json:"tone"`
	Radio                radio.Config   `json:"radio"`
	Styles               map[string]int `json:"styles,omitempty"`
	Version              int            `json:"version"`
	Volume               float64        `json:"volume"`
	Muted                bool           `json:"muted"`
	Subtitles            bool           `json:"subtitles"`
	Language             string         `json:"subtitleLanguage"`
	AutoHide             bool           `json:"autoHideControls"`
	HideDelay            int            `json:"hideDelaySeconds"`
}

const maxPreferencesBytes = 16 * 1024

func defaultPreferences() Preferences {
	return Preferences{Audius: audius.DefaultConfig(), SpectrumCycleSeconds: 24, Radio: radio.DefaultConfig(), Version: 1, Volume: .8, Subtitles: true, Language: "spa", AutoHide: true, HideDelay: 2}
}
func (p Preferences) validate() error {
	if err := p.Audius.Validate(); err != nil {
		return err
	}
	if p.SpectrumCycleSeconds != 12 && p.SpectrumCycleSeconds != 24 && p.SpectrumCycleSeconds != 48 {
		return errors.New("ciclo RGB fuera de rango")
	}
	if err := p.Tone.Validate(); err != nil {
		return err
	}
	if err := p.Radio.Validate(); err != nil {
		return err
	}
	if len(p.Styles) > 100 {
		return errors.New("el perfil admite hasta 100 estilos")
	}
	for tag, weight := range p.Styles {
		if len(tag) == 0 || len(tag) > 80 || weight < -10 || weight > 10 {
			return errors.New("estilo o afinidad fuera de rango")
		}
	}
	if p.Version != 1 || math.IsNaN(p.Volume) || math.IsInf(p.Volume, 0) || p.Volume < 0 || p.Volume > 1 || p.HideDelay < 1 || p.HideDelay > 10 {
		return errors.New("preferencias fuera de rango")
	}
	switch p.Language {
	case "auto", "spa", "eng", "fra":
	default:
		return errors.New("idioma de subtítulos no válido")
	}
	return nil
}
func preferencesPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "auraplayer-community", "preferences.json"), nil
}

// Read the legacy profile only on the first Community launch. Future saves use
// Community's own profile so new fields cannot break older Aura editions.
func readCommunityPreferences(path string) (Preferences, error) {
	p, err := readPreferences(path)
	if !errors.Is(err, os.ErrNotExist) {
		return p, err
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return Preferences{}, err
	}
	return readPreferences(filepath.Join(dir, "playerdoit", "preferences.json"))
}
func readPreferences(path string) (Preferences, error) {
	f, err := os.Open(path)
	if err != nil {
		return Preferences{}, err
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, maxPreferencesBytes+1))
	d.DisallowUnknownFields()
	p := defaultPreferences()
	if err := d.Decode(&p); err != nil {
		return p, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return p, errors.New("contenido extra en preferencias")
	}
	return p, p.validate()
}
func writePreferences(path string, p Preferences) error {
	if err := p.validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > maxPreferencesBytes {
		return errors.New("las preferencias superan el límite de 16 KiB; reduce estilos o espejos")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), "preferences-*.tmp")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("guardar preferencias: %w", err)
	}
	return nil
}
