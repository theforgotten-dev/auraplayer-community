package dsp

import (
	"errors"
	"fmt"
	"math"
	"math/cmplx"
	"strings"
)

// ToneConfig is shared by the PCM processor and libmpv's audio filters.
// Zero values mean unmodified audio. Cutoffs are derived from the band controls.
type ToneConfig struct {
	Enabled  bool        `json:"enabled"`
	PreampDB float64     `json:"preampDB"`
	GainsDB  [10]float64 `json:"gainsDB"`
}

func ToneFrequencies() [10]float64 {
	return [10]float64{31, 63, 125, 250, 500, 1000, 2000, 4000, 8000, 16000}
}

func (c ToneConfig) Validate() error {
	valid := func(v, lo, hi float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= lo && v <= hi }
	if !valid(c.PreampDB, -24, 12) {
		return errors.New("preamplificador fuera de rango (-24 a +12 dB)")
	}
	for _, gain := range c.GainsDB {
		if !valid(gain, -12, 12) {
			return errors.New("ecualizador fuera de rango (-12 a +12 dB)")
		}
	}
	return nil
}

// EffectivePreamp reserves headroom from the combined frequency response plus
// 0.5 dB of margin. This is gain staging, not a true-peak limiter.
func (c ToneConfig) EffectivePreamp() float64 {
	positive := false
	for _, g := range c.GainsDB {
		positive = positive || g > 0
	}
	if !positive {
		return c.PreampDB
	}
	chain := designTone(c, 48000)
	peak := 0.0
	for i := 0; i <= 512; i++ {
		hz := 20 * math.Pow(1000, float64(i)/512)
		peak = math.Max(peak, chain.responseDB(hz, 48000))
	}
	return c.PreampDB - peak - .5
}

// AutomaticCutoffs is an explicit product policy, not content analysis.
// Edge groups weighted 60/30/10 start adding roll-off after a 3 dB net cut.
// Flat and boosted edges add no high/low pass filter.
func (c ToneConfig) AutomaticCutoffs() (high, low float64) {
	strength := func(a, b, d float64) float64 { return math.Max(0, math.Min(1, (-(.6*a+.3*b+.1*d)-3)/9)) }
	bass := strength(c.GainsDB[0], c.GainsDB[1], c.GainsDB[2])
	treble := strength(c.GainsDB[9], c.GainsDB[8], c.GainsDB[7])
	high = math.Round(125*bass*bass*10) / 10
	if high < 1 {
		high = 0
	}
	if treble > 0 {
		low = math.Round((24000-19000*treble)*10) / 10
		if low >= 23999 {
			low = 0
		}
	}
	return
}

// ToneResponseDB samples the actual designed transfer function, including
// automatic roll-off and effective preamp. It is not a measured spectrum.
func ToneResponseDB(c ToneConfig, hz float64) float64 {
	if !c.Enabled {
		return 0
	}
	return designTone(c, 48000).responseDB(hz, 48000) + c.EffectivePreamp()
}

// LavfiGraph uses only validated numeric parameters, never user-supplied expressions.
func (c ToneConfig) LavfiGraph() (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}
	if !c.Enabled {
		return "", nil
	}
	filters := []string{"aresample=48000", fmt.Sprintf("volume=%.6fdB", c.EffectivePreamp())}
	high, low := c.AutomaticCutoffs()
	if high > 0 {
		filters = append(filters, fmt.Sprintf("highpass=f=%.6f:t=q:w=0.7071067811865476", high))
	}
	frequencies := ToneFrequencies()
	for i, gain := range c.GainsDB {
		if gain != 0 {
			filters = append(filters, fmt.Sprintf("equalizer=f=%.0f:t=q:w=1.4142135623730951:g=%.6f", frequencies[i], gain))
		}
	}
	if low > 0 {
		filters = append(filters, fmt.Sprintf("lowpass=f=%.6f:t=q:w=0.7071067811865476", low))
	}
	return strings.Join(filters, ","), nil
}

type biquad struct {
	b0, b1, b2, a1, a2 float64
	z1, z2             [2]float64
}

func (b *biquad) sample(x float64, ch int) float64 {
	y := b.b0*x + b.z1[ch]
	b.z1[ch] = b.b1*x - b.a1*y + b.z2[ch]
	b.z2[ch] = b.b2*x - b.a2*y
	return y
}

// Coefficients follow the RBJ Audio EQ Cookbook (W3C), normalized by a0.
func makeBiquad(kind string, hz, gain, rate float64) biquad {
	w := 2 * math.Pi * hz / rate
	cs := math.Cos(w)
	sn := math.Sin(w)
	alpha := sn / (2 * math.Sqrt2)
	var b0, b1, b2, a0, a1, a2 float64
	switch kind {
	case "peak":
		a := math.Pow(10, gain/40)
		b0 = 1 + alpha*a
		b1 = -2 * cs
		b2 = 1 - alpha*a
		a0 = 1 + alpha/a
		a1 = -2 * cs
		a2 = 1 - alpha/a
	case "high":
		alpha = sn / math.Sqrt2
		b0 = (1 + cs) / 2
		b1 = -(1 + cs)
		b2 = b0
		a0 = 1 + alpha
		a1 = -2 * cs
		a2 = 1 - alpha
	case "low":
		alpha = sn / math.Sqrt2
		b0 = (1 - cs) / 2
		b1 = 1 - cs
		b2 = b0
		a0 = 1 + alpha
		a1 = -2 * cs
		a2 = 1 - alpha
	}
	return biquad{b0: b0 / a0, b1: b1 / a0, b2: b2 / a0, a1: a1 / a0, a2: a2 / a0}
}

type toneChain struct {
	filters [12]biquad
	count   int
	gain    float64
	enabled bool
}

func makeToneChain(c ToneConfig, rate float64) toneChain {
	chain := designTone(c, rate)
	chain.gain = math.Pow(10, c.EffectivePreamp()/20)
	return chain
}

func designTone(c ToneConfig, rate float64) toneChain {
	chain := toneChain{enabled: c.Enabled, gain: 1}
	add := func(kind string, hz, gain float64) {
		chain.filters[chain.count] = makeBiquad(kind, hz, gain, rate)
		chain.count++
	}
	high, low := c.AutomaticCutoffs()
	if high > 0 {
		add("high", high, 0)
	}
	for i, hz := range ToneFrequencies() {
		if c.GainsDB[i] != 0 {
			add("peak", hz, c.GainsDB[i])
		}
	}
	if low > 0 {
		add("low", low, 0)
	}
	return chain
}

func (c toneChain) responseDB(hz, rate float64) float64 {
	w := 2 * math.Pi * hz / rate
	z := complex(math.Cos(w), -math.Sin(w))
	z2 := z * z
	amplitude := 1.0
	for i := 0; i < c.count; i++ {
		b := c.filters[i]
		amplitude *= cmplx.Abs((complex(b.b0, 0) + complex(b.b1, 0)*z + complex(b.b2, 0)*z2) / (1 + complex(b.a1, 0)*z + complex(b.a2, 0)*z2))
	}
	return 20 * math.Log10(math.Max(amplitude, 1e-12))
}
func (c *toneChain) sample(x float64, ch int) float64 {
	if !c.enabled {
		return x
	}
	x *= c.gain
	for i := 0; i < c.count; i++ {
		x = c.filters[i].sample(x, ch)
	}
	return x
}
