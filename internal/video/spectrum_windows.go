//go:build windows

package video

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"playerdoit/internal/dsp"
)

// Preserve the first stereo pair for playback. Extra pairs measure band energy
// and are discarded after astats attaches their measurements to the frame.
func spectrumGraph() string {
	edges := [...]float64{40, 63, 100, 160, 250, 400, 630, 1000, 1600, 2500, 4000, 6300, 8000, 10000, 12500, 16000, 20000}
	var g strings.Builder
	g.WriteString("aformat=sample_rates=48000:channel_layouts=stereo,asplit=17[dry]")
	for i := 0; i < 16; i++ {
		fmt.Fprintf(&g, "[b%d]", i)
	}
	g.WriteByte(';')
	for i := 0; i < 16; i++ {
		fmt.Fprintf(&g, "[b%d]bandpass=f=%.3f:t=h:w=%.3f,aformat=sample_fmts=fltp[m%d];", i, math.Sqrt(edges[i]*edges[i+1]), edges[i+1]-edges[i], i)
	}
	g.WriteString("[dry]aformat=sample_fmts=fltp[mDry];[mDry]")
	for i := 0; i < 16; i++ {
		fmt.Fprintf(&g, "[m%d]", i)
	}
	g.WriteString("amerge=inputs=17,astats=metadata=1:reset=1:measure_perchannel=RMS_level:measure_overall=none,pan=stereo|c0=c0|c1=c1")
	return g.String()
}

func (p *mpvPlayer) Spectrum() dsp.Spectrum {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.handle == 0 || !p.spectrumEnabled {
		return dsp.EmptySpectrum()
	}
	if time.Since(p.spectrumRead) < time.Second/30 {
		return p.spectrum
	}
	p.spectrumRead = time.Now()
	s := dsp.EmptySpectrum()
	level := func(channel int) float64 {
		value, err := strconv.ParseFloat(p.text(fmt.Sprintf("af-metadata/aura-spectrum/lavfi.astats.%d.RMS_level", channel)), 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return 0
		}
		return max(0, min(1, (value+72)/72))
	}
	s.LeftLevel, s.RightLevel = level(1), level(2)
	for i := range s.Left {
		s.Left[i], s.Right[i] = level(3+i*2), level(4+i*2)
	}
	p.spectrum = s
	return s
}
