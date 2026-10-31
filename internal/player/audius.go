package player

import (
	"context"
	"errors"
	"time"

	"playerdoit/internal/audius"
)

// QueueAudius stores metadata and provider configuration in memory only. It
// never stores a resolved content-node URL or bypasses stream access checks.
func (c *Controller) QueueAudius(config audius.Config, source audius.Track) (int, error) {
	if err := config.Validate(); err != nil {
		return -1, err
	}
	if !source.Streamable() || source.PageURL() == "" {
		return -1, errors.New("Audius: pista sin acceso público")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	id := "audius:" + source.ID
	if c.audiusSources == nil {
		c.audiusSources = map[string]audius.Config{}
	}
	c.audiusSources[id] = config
	for i, t := range c.queue {
		if t.ID == id {
			return i, nil
		}
	}
	t := audiusPlayerTrack(source)
	c.queue = append(c.queue, t)
	if c.currentIndex < 0 {
		c.currentIndex = len(c.queue) - 1
	}
	return len(c.queue) - 1, nil
}

func audiusPlayerTrack(t audius.Track) Track {
	return Track{ID: "audius:" + t.ID, Provider: "audius", ProviderID: t.ID, Title: t.Title + " — " + t.Artist(), Artist: t.Artist(), Permalink: t.PageURL(), ArtworkURL: t.ArtworkURL(), License: t.License, Copyright: t.Copyright(), Path: t.PageURL(), Codec: "MP3", LibMPV: true, Duration: float64(t.Duration)}
}

func (c *Controller) PlayAudius(config audius.Config, source audius.Track) error {
	index, err := c.QueueAudius(config, source)
	if err != nil {
		return err
	}
	c.mu.Lock()
	if c.currentIndex == index {
		err = c.closeCurrentLocked()
		c.state = StateStopped
		c.position = 0
	}
	c.mu.Unlock()
	if err != nil {
		return err
	}
	return c.Play(index)
}

func (c *Controller) resolveAudiusLocked(t Track) (Track, string, error) {
	config, ok := c.audiusSources[t.ID]
	if !ok {
		return Track{}, "", errors.New("Audius: vuelve a buscar la pista en esta sesión")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	source, stream, err := (audius.Client{Config: config}).Resolve(ctx, t.ProviderID)
	if err != nil {
		return Track{}, "", err
	}
	return audiusPlayerTrack(source), stream, nil
}

// ClearAudius removes provider metadata on explicit session clearing, preserving
// local tracks and the radio slot. All controller mutations remain serialized.
func (c *Controller) ClearAudius() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	oldIndex := c.currentIndex
	if oldIndex >= 0 && oldIndex < len(c.queue) && c.queue[oldIndex].Provider == "audius" {
		if err := c.closeCurrentLocked(); err != nil {
			return err
		}
		c.state = StateStopped
		c.position = 0
	}
	queue := make([]Track, 0, len(c.queue))
	newIndex := -1
	for i, t := range c.queue {
		if t.Provider != "audius" {
			if i == oldIndex {
				newIndex = len(queue)
			}
			queue = append(queue, t)
		}
	}
	c.queue = queue
	c.audiusSources = nil
	c.currentIndex = newIndex
	if c.currentIndex < 0 && len(queue) > 0 {
		c.currentIndex = 0
	}
	return nil
}
