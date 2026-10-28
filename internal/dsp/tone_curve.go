package dsp

import "math"

func curveFrequency(index, count int) float64 {
	return 20 * math.Pow(1000, float64(index)/float64(count-1))
}

// ToneCurve evaluates the combined filter response on a logarithmic frequency
// axis from 20 Hz to 20 kHz. Compute on parameter changes, not on the audio thread.
func ToneCurve(c ToneConfig) [128]float64 {
	var result [128]float64
	if !c.Enabled {
		return result
	}
	chain := designTone(c, 48000)
	preamp := c.EffectivePreamp()
	for i := range result {
		result[i] = chain.responseDB(curveFrequency(i, len(result)), 48000) + preamp
	}
	return result
}
