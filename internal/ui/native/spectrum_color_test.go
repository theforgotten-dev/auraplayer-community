package native

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestSpectrumRGBContinuousCycle(t *testing.T) {
	previous := spectrumRGB(0)
	for step := 1; step <= 1000; step++ {
		hue := float64(step) / 1000
		c := spectrumRGB(hue)
		if c.A != 255 {
			t.Fatal("spectrum colors must be opaque")
		}
		for _, pair := range [][2]uint8{{c.R, previous.R}, {c.G, previous.G}, {c.B, previous.B}} {
			if math.Abs(float64(pair[0])-float64(pair[1])) > 2 {
				t.Fatalf("abrupt transition at hue %f", hue)
			}
		}
		previous = c
	}
	if spectrumRGB(0) != spectrumRGB(1) || spectrumRGB(-.25) != spectrumRGB(.75) {
		t.Fatal("hue must wrap seamlessly")
	}
}

func TestSpectrumRGBPreferencesPersist(t *testing.T) {
	p := defaultPreferences()
	p.SpectrumRGB = true
	p.SpectrumCycleSeconds = 48
	path := filepath.Join(t.TempDir(), "preferences.json")
	if err := writePreferences(path, p); err != nil {
		t.Fatal(err)
	}
	got, err := readPreferences(path)
	if err != nil || !got.SpectrumRGB || got.SpectrumCycleSeconds != 48 {
		t.Fatalf("RGB preferences lost: %+v, %v", got, err)
	}
	p.SpectrumCycleSeconds = 0
	if err := p.validate(); err == nil {
		t.Fatal("accepted zero cycle duration")
	}
}

func TestLegacyPreferencesReceiveSpectrumDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"volume":0.8}`), 0600); err != nil {
		t.Fatal(err)
	}
	p, err := readPreferences(path)
	if err != nil || p.SpectrumRGB || p.SpectrumCycleSeconds != 24 {
		t.Fatalf("legacy settings must retain skin colors and a valid RGB cycle: %+v, %v", p, err)
	}
}
