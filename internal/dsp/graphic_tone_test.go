package dsp

import (
	"encoding/json"
	"math"
	"math/cmplx"
	"testing"
)

func TestAutomaticCutoffsFollowEdgeGroups(t *testing.T) {
	flat := ToneConfig{Enabled: true}
	h, l := flat.AutomaticCutoffs()
	if h != 0 || l != 0 {
		t.Fatal("flat enabled extra filters")
	}
	middle := flat
	middle.GainsDB[5] = -12
	h, l = middle.AutomaticCutoffs()
	if h != 0 || l != 0 {
		t.Fatal("mid cut changed edge cutoffs")
	}
	previous := 0.0
	for _, gain := range []float64{0, -3, -6, -9, -12} {
		c := flat
		c.GainsDB[0] = gain
		c.GainsDB[1] = gain
		c.GainsDB[2] = gain
		h, l = c.AutomaticCutoffs()
		if h < previous || h > 125 || l != 0 {
			t.Fatal(h, l)
		}
		previous = h
	}
	c := flat
	c.GainsDB[7] = -12
	c.GainsDB[8] = -12
	c.GainsDB[9] = -12
	h, l = c.AutomaticCutoffs()
	if h != 0 || l != 5000 {
		t.Fatal(h, l)
	}
}

func TestGraphicMeasuredResponseMatchesDesignedCurve(t *testing.T) {
	c := ToneConfig{Enabled: true, GainsDB: [10]float64{-10, -8, -5, -2, 0, 2, 3, 2, -3, -6}}
	for _, hz := range []float64{31, 63, 125, 250, 500, 1000, 1414, 2000, 4000, 8000, 16000} {
		expected := ToneResponseDB(c, hz)
		if expected < -38 {
			continue
		}
		actual := 20 * math.Log10(level(filteredPCM(t, c, hz))/level(tonePCM(hz, 48000)))
		if math.Abs(actual-expected) > .25 {
			t.Errorf("%.0f Hz: measured %.2f dB, designed %.2f dB", hz, actual, expected)
		}
	}
}

func TestGraphicFilterPolesRemainStableAtExtremes(t *testing.T) {
	for _, gain := range []float64{-12, 12} {
		c := ToneConfig{Enabled: true}
		for i := range c.GainsDB {
			c.GainsDB[i] = gain
		}
		chain := designTone(c, 48000)
		for i := 0; i < chain.count; i++ {
			b := chain.filters[i]
			root := cmplx.Sqrt(complex(b.a1*b.a1-4*b.a2, 0))
			for _, pole := range []complex128{(-complex(b.a1, 0) + root) / 2, (-complex(b.a1, 0) - root) / 2} {
				if cmplx.Abs(pole) >= 1 {
					t.Fatalf("unstable filter %d: %v", i, pole)
				}
			}
		}
	}
}

func TestToneJSONMigratesFrequenciesAndRejectsMalformedProfiles(t *testing.T) {
	var c ToneConfig
	if err := json.Unmarshal([]byte(`{"enabled":true,"gainsDB":[1,2,3,4,5],"preampDB":-2}`), &c); err != nil {
		t.Fatal(err)
	}
	if c.GainsDB[0] != 1 || c.GainsDB[3] != 2 || c.GainsDB[5] != 3 || c.GainsDB[7] != 4 || c.GainsDB[9] != 5 {
		t.Fatal(c.GainsDB)
	}
	for _, data := range []string{`{"gainsDB":[1,2]}`, `{"gainsDB":[99,0,0,0,0],"highPassHz":80}`, `{"unexpected":true}`} {
		if json.Unmarshal([]byte(data), &c) == nil {
			t.Fatalf("accepted %s", data)
		}
	}
	if err := json.Unmarshal([]byte(`{"enabled":true,"gainsDB":[0,0,0,0,0],"highPassHz":80,"lowPassHz":12000}`), &c); err != nil {
		t.Fatal(err)
	}
	if c.GainsDB[0] >= 0 || c.GainsDB[9] >= 0 {
		t.Fatal("legacy cutoff response was lost")
	}
}
