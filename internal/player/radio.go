package player

import (
	"errors"
	"playerdoit/internal/radio"
)

// Tune reuses one radio slot and closes the previous source before opening a stream.
func (c *Controller) Tune(station radio.Station) error {
	if !radio.ValidURL(station.URL) {
		return errors.New("invalid radio URL: expected HTTP or HTTPS")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.closeCurrentLocked(); err != nil {
		return c.failLocked(err)
	}
	c.state = StateLoading
	track := Track{ID: "radio:" + station.ID, Title: station.Name, Path: station.URL, LibMPV: true, Codec: station.Codec, Live: true}
	index := -1
	for i, t := range c.queue {
		if t.Live {
			index = i
			break
		}
	}
	if index < 0 {
		index = len(c.queue)
		c.queue = append(c.queue, track)
	} else {
		c.queue[index] = track
	}
	if err := c.loadMediaLocked(index); err != nil {
		return c.failLocked(err)
	}
	c.player.Play()
	if err := c.player.Err(); err != nil {
		return c.failLocked(err)
	}
	c.state = StatePlaying
	c.lastError = ""
	return nil
}
