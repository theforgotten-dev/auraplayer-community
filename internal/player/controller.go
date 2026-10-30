// Package player coordinates the playlist, decoders, and PCM audio engine.
package player

import (
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"go.senan.xyz/taglib"

	"playerdoit/internal/audio"
	"playerdoit/internal/audio/otoengine"
	"playerdoit/internal/audio/pipeline"
	"playerdoit/internal/codecs"
	"playerdoit/internal/dsp"
	"playerdoit/internal/video"
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
	Live       bool    `json:"live"`
	LibMPV     bool    `json:"libmpv"`
	Video      bool    `json:"video"`
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Path       string  `json:"path"`
	Codec      string  `json:"codec"`
	Duration   float64 `json:"duration"`
	SampleRate int     `json:"sampleRate"`
	Channels   int     `json:"channels"`
}

type Snapshot struct {
	Tone             dsp.ToneConfig        `json:"tone"`
	SubtitleTracks   []video.SubtitleTrack `json:"subtitleTracks"`
	SubtitlesVisible bool                  `json:"subtitlesVisible"`
	State            PlaybackState         `json:"state"`
	Queue            []Track               `json:"queue"`
	CurrentIndex     int                   `json:"currentIndex"`
	Position         float64               `json:"position"`
	Volume           float64               `json:"volume"`
	Error            string                `json:"error"`
}

type Controller struct {
	tone       dsp.ToneConfig
	toneStream *dsp.ToneStream
	mu         sync.Mutex

	engine           audio.Engine
	player           audio.Player
	analyzer         *dsp.Analyzer
	decoderTask      io.Closer
	source           *os.File
	queue            []Track
	currentIndex     int
	state            PlaybackState
	position         time.Duration
	startedAt        time.Time
	volume           float64
	lastError        string
	subtitlesEnabled bool
	subtitleLanguage string
}

func NewController() *Controller {
	return &Controller{
		currentIndex:     -1,
		state:            StateStopped,
		volume:           0.8,
		subtitlesEnabled: true,
		subtitleLanguage: "spa",
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
	if err := c.player.Err(); err != nil {
		return c.failLocked(err)
	}
	c.startedAt = time.Now()
	c.state = StatePlaying
	return nil
}

func (c *Controller) Toggle() error {
	c.mu.Lock()
	if c.state == StatePlaying {
		c.position = c.positionLocked()
		c.player.Pause()
		if err := c.player.Err(); err != nil {
			c.failLocked(err)
			c.mu.Unlock()
			return err
		}
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
	if c.currentIndex >= 0 && c.currentIndex < len(c.queue) && c.queue[c.currentIndex].Live {
		return errors.New("la radio en vivo no permite buscar una posicion")
	}
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
		if err := c.player.Err(); err != nil {
			c.failLocked(err)
		} else {
			c.position = c.currentDurationLocked()
			c.state = StateStopped
		}
	}

	queue := make([]Track, len(c.queue))
	copy(queue, c.queue)
	subtitlesVisible := false
	var subtitleTracks []video.SubtitleTrack
	if p, ok := c.player.(video.Player); ok {
		subtitlesVisible = p.SubtitlesVisible()
		subtitleTracks = p.SubtitleTracks()
	}
	return Snapshot{
		Tone:             c.tone,
		SubtitleTracks:   subtitleTracks,
		SubtitlesVisible: subtitlesVisible,
		State:            c.state,
		Queue:            queue,
		CurrentIndex:     c.currentIndex,
		Position:         c.positionLocked().Seconds(),
		Volume:           c.volume,
		Error:            c.lastError,
	}
}

// Spectrum returns the latest FFT result without interrupting playback.
func (c *Controller) Spectrum() dsp.Spectrum {
	c.mu.Lock()
	analyzer := c.analyzer
	if analyzer == nil {
		if source, ok := c.player.(interface{ Spectrum() dsp.Spectrum }); ok {
			s := source.Spectrum()
			c.mu.Unlock()
			return s
		}
	}
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

// VideoFrame returns an immutable frame rendered independently of the UI.
func (c *Controller) VideoFrame() *image.RGBA {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p, ok := c.player.(video.Player); ok {
		return p.Frame()
	}
	return nil
}

func (c *Controller) LoadSubtitles(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, ok := c.player.(video.Player)
	if !ok {
		return errors.New("play a video before loading subtitles")
	}
	return p.LoadSubtitles(path)
}

func (c *Controller) SelectSubtitle(id int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, ok := c.player.(video.Player)
	if !ok {
		return errors.New("play a video before selecting subtitles")
	}
	return p.SelectSubtitle(id)
}

func (c *Controller) ConfigureSubtitles(enabled bool, language string) error {
	switch language {
	case "auto", "spa", "eng", "fra":
	default:
		return errors.New("unsupported subtitle language")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if p, ok := c.player.(video.Player); ok {
		if err := configureVideoSubtitles(p, enabled, language); err != nil {
			return err
		}
	}
	c.subtitlesEnabled = enabled
	c.subtitleLanguage = language
	return nil
}

func configureVideoSubtitles(p video.Player, enabled bool, language string) error {
	if language != "auto" {
		for _, track := range p.SubtitleTracks() {
			if track.Language == language {
				if err := p.SelectSubtitle(track.ID); err != nil {
					return err
				}
				break
			}
		}
	}
	return p.SetSubtitlesVisible(enabled)
}

func (c *Controller) SetSubtitlesVisible(visible bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, ok := c.player.(video.Player)
	if !ok {
		return errors.New("play a video before changing subtitles")
	}
	return p.SetSubtitlesVisible(visible)
}

// SupportedExtensions includes both audio decoders and video containers.
func SupportedExtensions() []string {
	all := append(codecs.SupportedExtensions(), video.SupportedExtensions()...)
	for _, ext := range video.AudioExtensions() {
		if !slices.Contains(all, ext) {
			all = append(all, ext)
		}
	}
	return all
}

func IsSupported(extension string) bool {
	return codecs.IsSupported(extension) || video.IsSupported(extension) || video.IsAudioSupported(extension)
}

func (c *Controller) loadLocked(index int) error {
	// This call is synchronous and waits for the current decoder task to exit.
	// A new per-track task is created only after the previous one is gone.
	if err := c.closeCurrentLocked(); err != nil {
		return err
	}

	track := c.queue[index]
	if track.Video || track.LibMPV {
		return c.loadMediaLocked(index)
	}
	// #nosec G304 -- track paths originate from explicit user file selection.
	file, err := os.Open(track.Path)
	if err != nil {
		return fmt.Errorf("open %q: %w", track.Title, err)
	}
	stream, err := codecs.Decode(filepath.Ext(track.Path), file)
	if err != nil {
		if closeErr := file.Close(); closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
			return errors.Join(err, closeErr)
		}
		return c.loadMediaLocked(index)
	}
	decoderTask, _ := stream.PCM.(io.Closer)

	outputConfig := pipeline.OutputConfig()
	normalized, err := pipeline.Normalize(stream.PCM, stream.Config, outputConfig)
	if err != nil {
		return errors.Join(err, closeResource(decoderTask), file.Close())
	}
	toneStream, err := dsp.NewToneStream(normalized, outputConfig, c.tone)
	if err != nil {
		return errors.Join(err, closeResource(decoderTask), file.Close())
	}
	analyzer, err := dsp.NewAnalyzer(toneStream, outputConfig)
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
	c.toneStream = toneStream
	c.analyzer = analyzer
	c.decoderTask = decoderTask
	c.source = file
	c.currentIndex = index
	c.position = 0
	return nil
}

func (c *Controller) loadMediaLocked(index int) error {
	track := c.queue[index]
	var p video.Player
	var err error
	if track.Video {
		p, err = video.Open(track.Path)
	} else {
		p, err = video.OpenAudio(track.Path)
	}
	if err != nil {
		return err
	}
	if err := p.SetVolume(c.volume); err != nil {
		return errors.Join(err, p.Close())
	}
	if tone, ok := p.(tonePlayer); ok {
		if err := tone.ConfigureTone(c.tone); err != nil {
			return errors.Join(err, p.Close())
		}
	} else if c.tone.Enabled {
		return errors.Join(errors.New("ecualizacion no disponible"), p.Close())
	}
	if track.Video {
		if err := configureVideoSubtitles(p, c.subtitlesEnabled, c.subtitleLanguage); err != nil {
			return errors.Join(err, p.Close())
		}
	}
	c.player = p
	c.currentIndex = index
	c.queue[index].LibMPV = true
	if !track.Live {
		c.queue[index].Codec = strings.TrimPrefix(strings.ToUpper(filepath.Ext(track.Path)), ".")
	}
	c.queue[index].Duration = p.Duration().Seconds()
	c.queue[index].SampleRate, c.queue[index].Channels = p.AudioFormat()
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
	c.toneStream = nil
	if c.decoderTask != nil {
		if err := c.decoderTask.Close(); closeErr == nil {
			closeErr = err
		}
		c.decoderTask = nil
	}
	if c.source != nil {
		// Beep decoders can own and close the original file themselves.
		if err := c.source.Close(); closeErr == nil && !errors.Is(err, os.ErrClosed) {
			closeErr = err
		}
		c.source = nil
	}
	return closeErr
}

func (c *Controller) positionLocked() time.Duration {
	if p, ok := c.player.(video.Player); ok && c.state != StateStopped {
		return p.Position()
	}
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
	if !IsSupported(extension) {
		return Track{}, fmt.Errorf("unsupported file %q: expected %s", path, strings.Join(SupportedExtensions(), ", "))
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return Track{}, fmt.Errorf("inspect %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return Track{}, fmt.Errorf("audio source %q is not a regular file", path)
	}
	if video.IsSupported(extension) || video.IsAudioSupported(extension) {
		return Track{ID: absPath, Path: absPath, Title: strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath)), Codec: strings.TrimPrefix(strings.ToUpper(extension), "."), Video: video.IsSupported(extension), LibMPV: true}, nil
	}
	properties, err := taglib.ReadProperties(absPath)
	if err != nil {
		return Track{}, fmt.Errorf("read audio properties for %q: %w", path, err)
	}
	if properties.SampleRate == 0 || properties.Channels == 0 {
		return Track{}, fmt.Errorf("unsupported audio layout in %q: %d Hz, %d channels", path, properties.SampleRate, properties.Channels)
	}
	title := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
	return Track{
		ID:         absPath,
		Title:      title,
		Path:       absPath,
		Codec:      codecs.Name(extension),
		LibMPV:     properties.Channels > 2,
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
