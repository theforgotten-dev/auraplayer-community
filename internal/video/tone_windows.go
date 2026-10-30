//go:build windows

package video

import "playerdoit/internal/dsp"

// ConfigureTone installs the same numeric EQ/filter parameters in libavfilter.
func (p *mpvPlayer) ConfigureTone(c dsp.ToneConfig) error {
	graph, err := c.LavfiGraph()
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if graph != "" {
		graph = "lavfi=[" + graph + "]"
	}
	p.spectrumEnabled = p.audioOnly && p.number("audio-params/channel-count") <= 2
	if p.spectrumEnabled {
		if graph != "" {
			graph += ","
		}
		graph += "@aura-spectrum:lavfi=[" + spectrumGraph() + "]"
	}
	return p.set("af", graph)
}
