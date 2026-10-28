package dsp

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
)

// UnmarshalJSON migrates the five-band demo without assigning old gains to the
// wrong frequencies. Old manual cutoffs are approximated by the new band gains.
func (c *ToneConfig) UnmarshalJSON(data []byte) error {
	var raw struct {
		Enabled  bool      `json:"enabled"`
		PreampDB float64   `json:"preampDB"`
		GainsDB  []float64 `json:"gainsDB"`
		High     float64   `json:"highPassHz"`
		Low      float64   `json:"lowPassHz"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return err
	}
	for _, gain := range raw.GainsDB {
		if math.IsNaN(gain) || math.IsInf(gain, 0) || gain < -12 || gain > 12 {
			return errors.New("ganancia anterior fuera de rango")
		}
	}
	next := ToneConfig{Enabled: raw.Enabled, PreampDB: raw.PreampDB}
	switch len(raw.GainsDB) {
	case 0:
	case 10:
		copy(next.GainsDB[:], raw.GainsDB)
	case 5:
		old := [5]float64{60, 250, 1000, 4000, 12000}
		for i, hz := range ToneFrequencies() {
			gain := raw.GainsDB[0]
			if hz >= old[4] {
				gain = raw.GainsDB[4]
			} else {
				for j := 0; j < 4; j++ {
					if hz >= old[j] && hz < old[j+1] {
						t := math.Log(hz/old[j]) / math.Log(old[j+1]/old[j])
						gain = raw.GainsDB[j]*(1-t) + raw.GainsDB[j+1]*t
						break
					}
				}
			}
			next.GainsDB[i] = gain
		}
	default:
		return errors.New("el ecualizador requiere diez bandas (o cinco para migrar el demo)")
	}
	if raw.High < 0 || raw.High > 300 || raw.Low < 0 || raw.Low > 20000 {
		return errors.New("cortes anteriores fuera de rango")
	}
	if (raw.High > 0 || raw.Low > 0) && len(raw.GainsDB) != 10 {
		chain := toneChain{}
		if raw.High > 0 {
			chain.filters[chain.count] = makeBiquad("high", raw.High, 0, 48000)
			chain.count++
		}
		if raw.Low > 0 {
			chain.filters[chain.count] = makeBiquad("low", raw.Low, 0, 48000)
			chain.count++
		}
		for i, hz := range ToneFrequencies() {
			next.GainsDB[i] = math.Max(-12, math.Min(12, next.GainsDB[i]+chain.responseDB(hz, 48000)))
		}
	}
	if err := next.Validate(); err != nil {
		return err
	}
	*c = next
	return nil
}
