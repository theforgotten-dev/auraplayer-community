package player

import (
	"errors"
	"playerdoit/internal/dsp"
)

type tonePlayer interface{ ConfigureTone(dsp.ToneConfig) error }

// ConfigureTone changes the sound-processing block without reloading the track.
func (c *Controller) ConfigureTone(config dsp.ToneConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.toneStream != nil {
		if err := c.toneStream.Configure(config); err != nil {
			return err
		}
	}
	if p, ok := c.player.(tonePlayer); ok {
		if err := p.ConfigureTone(config); err != nil {
			return err
		}
	} else if c.player != nil && c.toneStream == nil && config.Enabled {
		return errors.New("este reproductor no admite ecualizacion")
	}
	c.tone = config
	return nil
}
