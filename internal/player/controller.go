// Package player coordinates the playlist, decoders, and PCM audio engine.
package player

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.senan.xyz/taglib"

	"playerdoit/internal/audio"
	"playerdoit/internal/audio/otoengine"
	"playerdoit/internal/audio/pipeline"
	"playerdoit/internal/codecs"
	"playerdoit/internal/dsp"
)

type PlaybackState string

const (
	StateStopped PlaybackState = "stopped"
	StatePlaying PlaybackState = "playing"
	StatePaused  PlaybackState = "paused"
	StateLoading PlaybackState = "loading"
	StateError   PlaybackState = "error"
)

type Track struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Path       string  `json:"path"`
	Codec      string  `json:"codec"`
	Duration   float64 `json:"duration"`
	SampleRate int     `json:"sampleRate"`
	Channels   int     `json:"channels"`
}

type Snapshot struct {
	State        PlaybackState `json:"state"`
	Queue        []Track       `json:"queue"`
	CurrentIndex int           `json:"currentIndex"`
	Position     float64       `json:"position"`
	Volume       float64       `json:"volume"`
	Error        string        `json:"error"`
}

type Controller struct {
	mu sync.Mutex

	engine       audio.Engine
	player       audio.Player
	analyzer     *dsp.Analyzer
	decoderTask  io.Closer
	source       *os.File
	queue        []Track
	currentIndex int
	state        PlaybackState
	position     time.Duration
	startedAt    time.Time
	volume       float64
	lastError    string
}

func NewController() *Controller {
	return &Controller{
		currentIndex: -1,
		state:        StateStopped,
		volume:       0.8,
	}
}

func (c *Controller) AddFiles(paths []string) error {
	tracks := make([]Track, 0, len(paths))
	for _, path := range paths {
		track, err := inspectTrack(path)
		if err != nil {
			return err
		}
		tracks = append(tracks, track)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, track := range tracks {
		if !c.containsLocked(track.ID) {
			c.queue = append(c.queue, track)
		}
	}
	if c.currentIndex < 0 && len(c.queue) > 0 {
		c.currentIndex = 0
	}
	c.lastError = ""
	return nil
}

func (c *Controller) Play(index int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if index < 0 || index >= len(c.queue) {
		return c.failLocked(errors.New("select a track before playing"))
	}
	c.state = StateLoading
	c.lastError = ""

	if c.player == nil || index != c.currentIndex {
		if err := c.loadLocked(index); err != nil {
			return c.failLocked(err)
		}
	} else if duration := c.currentDurationLocked(); duration > 0 && c.position >= duration {
		if err := c.player.Seek(0); err != nil {
			return c.failLocked(err)
		}
		c.position = 0
	}

	c.player.Play()
	c.startedAt = time.Now()
	c.state = StatePlaying
	return nil
}

func (c *Controller) Toggle() error {
	c.mu.Lock()
	if c.state == StatePlaying {
		c.position = c.positionLocked()
		c.player.Pause()
		c.state = StatePaused
		c.mu.Unlock()
		return nil
	}
	index := c.currentIndex
	c.mu.Unlock()
	return c.Play(index)
}

func (c *Controller) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.player == nil {
		c.state = StateStopped
		c.position = 0
		return nil
	}
	if err := c.player.Stop(); err != nil {
		return c.failLocked(err)
	}
	c.position = 0
	c.state = StateStopped
	c.lastError = ""
	return nil
}

func (c *Controller) Seek(seconds float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.player == nil {
		return c.failLocked(errors.New("no track is loaded"))
	}
	if seconds < 0 {
		seconds = 0
	}
	duration := c.currentDurationLocked()
	position := time.Duration(seconds * float64(time.Second))
	if duration > 0 && position > duration {
		position = duration
	}
	if err := c.player.Seek(position); err != nil {
		return c.failLocked(err)
	}
	c.position = position
	if c.state == StatePlaying {
		c.startedAt = time.Now()
	}
	c.lastError = ""
	return nil
}

func (c *Controller) SetVolume(volume float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if volume < 0 || volume > 1 {
		return c.failLocked(errors.New("volume must be between 0 and 1"))
	}
	if c.player != nil {
		if err := c.player.SetVolume(volume); err != nil {
			return c.failLocked(err)
		}
	}
	c.volume = volume
	c.lastError = ""
	return nil
}

func (c *Controller) Next() error {
	c.mu.Lock()
	if len(c.queue) == 0 {
		c.mu.Unlock()
		return errors.New("the queue is empty")
	}
	index := c.currentIndex + 1
	if index >= len(c.queue) {
		index = 0
	}
	c.mu.Unlock()
	return c.Play(index)
}

func (c *Controller) Previous() error {
	c.mu.Lock()
	if len(c.queue) == 0 {
		c.mu.Unlock()
		return errors.New("the queue is empty")
	}
	index := c.currentIndex - 1
	if index < 0 {
		index = len(c.queue) - 1
	}
	c.mu.Unlock()
	return c.Play(index)
}

func (c *Controller) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == StatePlaying && c.player != nil && !c.player.IsPlaying() {
		c.position = c.currentDurationLocked()
		c.state = StateStopped
	}

	queue := make([]Track, len(c.queue))
	copy(queue, c.queue)
	return Snapshot{
		State:        c.state,
		Queue:        queue,
		CurrentIndex: c.currentIndex,
		Position:     c.positionLocked().Seconds(),
		Volume:       c.volume,
		Error:        c.lastError,
	}
}

// Spectrum returns the latest FFT result without interrupting playback.
func (c *Controller) Spectrum() dsp.Spectrum {
	c.mu.Lock()
	analyzer := c.analyzer
	c.mu.Unlock()
	if analyzer == nil {
		return dsp.EmptySpectrum()
	}
	return analyzer.Snapshot()
}

func (c *Controller) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closeCurrentLocked()
}

func (c *Controller) loadLocked(index int) error {
	// This call is synchronous and waits for the current decoder task to exit.
	// A new per-track task is created only after the previous one is gone.
	if err := c.closeCurrentLocked(); err != nil {
		return err
	}

	track := c.queue[index]
	// #nosec G304 -- track paths originate from explicit user file selection.
	file, err := os.Open(track.Path)
	if err != nil {
		return fmt.Errorf("open %q: %w", track.Title, err)
	}
	stream, err := codecs.Decode(filepath.Ext(track.Path), file)
	if err != nil {
		return errors.Join(err, file.Close())
	}
	decoderTask, _ := stream.PCM.(io.Closer)

	outputConfig := pipeline.OutputConfig()
	normalized, err := pipeline.Normalize(stream.PCM, stream.Config, outputConfig)
	if err != nil {
		return errors.Join(err, closeResource(decoderTask), file.Close())
	}
	analyzer, err := dsp.NewAnalyzer(normalized, outputConfig)
	if err != nil {
		return errors.Join(err, closeResource(decoderTask), file.Close())
	}

	if c.engine == nil {
		engine, err := otoengine.New(outputConfig)
		if err != nil {
			return errors.Join(err, closeResource(decoderTask), file.Close())
		}
		c.engine = engine
	}

	pcmPlayer := c.engine.NewPlayer(analyzer)
	if err := pcmPlayer.SetVolume(c.volume); err != nil {
		return errors.Join(err, pcmPlayer.Close(), closeResource(decoderTask), file.Close())
	}
	c.player = pcmPlayer
	c.analyzer = analyzer
	c.decoderTask = decoderTask
	c.source = file
	c.currentIndex = index
	c.position = 0
	return nil
}

func (c *Controller) closeCurrentLocked() error {
	var closeErr error
	if c.player != nil {
		closeErr = c.player.Close()
		c.player = nil
	}
	c.analyzer = nil
	if c.decoderTask != nil {
		if err := c.decoderTask.Close(); closeErr == nil {
			closeErr = err
		}
		c.decoderTask = nil
	}
	if c.source != nil {
		if err := c.source.Close(); closeErr == nil {
			closeErr = err
		}
		c.source = nil
	}
	return closeErr
}

func (c *Controller) positionLocked() time.Duration {
	position := c.position
	if c.state == StatePlaying {
		position += time.Since(c.startedAt)
	}
	duration := c.currentDurationLocked()
	if duration > 0 && position > duration {
		return duration
	}
	return position
}

func (c *Controller) currentDurationLocked() time.Duration {
	if c.currentIndex < 0 || c.currentIndex >= len(c.queue) {
		return 0
	}
	return time.Duration(c.queue[c.currentIndex].Duration * float64(time.Second))
}

func (c *Controller) containsLocked(id string) bool {
	for _, track := range c.queue {
		if strings.EqualFold(track.ID, id) {
			return true
		}
	}
	return false
}

func (c *Controller) failLocked(err error) error {
	c.state = StateError
	c.lastError = err.Error()
	return err
}

func inspectTrack(path string) (Track, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Track{}, err
	}
	extension := strings.ToLower(filepath.Ext(absPath))
	if !codecs.IsSupported(extension) {
		return Track{}, fmt.Errorf("unsupported file %q: expected %s", path, strings.Join(codecs.SupportedExtensions(), ", "))
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return Track{}, fmt.Errorf("inspect %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return Track{}, fmt.Errorf("audio source %q is not a regular file", path)
	}
	properties, err := taglib.ReadProperties(absPath)
	if err != nil {
		return Track{}, fmt.Errorf("read audio properties for %q: %w", path, err)
	}
	if properties.SampleRate == 0 || properties.Channels == 0 || properties.Channels > 2 {
		return Track{}, fmt.Errorf("unsupported audio layout in %q: %d Hz, %d channels", path, properties.SampleRate, properties.Channels)
	}
	title := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
	return Track{
		ID:         absPath,
		Title:      title,
		Path:       absPath,
		Codec:      codecs.Name(extension),
		Duration:   properties.Length.Seconds(),
		SampleRate: int(properties.SampleRate),
		Channels:   int(properties.Channels),
	}, nil
}

func closeResource(resource io.Closer) error {
	if resource == nil {
		return nil
	}
	return resource.Close()
}
