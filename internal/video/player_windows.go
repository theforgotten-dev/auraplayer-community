//go:build windows

package video

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/windows"
)

// These declarations follow the libmpv client and render C APIs. The DLL stays
// loaded for the process lifetime; no function pointer outlives its library.
type api struct {
	create       func() uintptr
	initialize   func(uintptr) int32
	option       func(uintptr, string, string) int32
	command      func(uintptr, **byte) int32
	get          func(uintptr, string, int32, unsafe.Pointer) int32
	getString    func(uintptr, string) *byte
	free         func(unsafe.Pointer)
	set          func(uintptr, string, string) int32
	wait         func(uintptr, float64) *event
	destroy      func(uintptr)
	renderCreate func(*uintptr, uintptr, *param) int32
	renderUpdate func(uintptr) uint64
	render       func(uintptr, *param) int32
	renderFree   func(uintptr)
}

type param struct {
	kind int32
	data unsafe.Pointer
}
type event struct {
	id       int32
	err      int32
	userdata uint64
	data     unsafe.Pointer
}
type endFile struct {
	reason int32
	err    int32
}

var library struct {
	sync.Once
	api api
	err error
}

func loadAPI() (*api, error) {
	library.Do(func() {
		exe, err := os.Executable()
		if err != nil {
			library.err = err
			return
		}
		// Only explicit application locations are searched, never the working
		// directory or PATH. A missing video runtime must not break audio playback.
		paths := []string{filepath.Join(filepath.Dir(exe), "libmpv-2.dll")}
		if configured := os.Getenv("AURA_MPV_DLL"); configured != "" {
			if !filepath.IsAbs(configured) {
				library.err = errors.New("AURA_MPV_DLL must be an absolute path")
				return
			}
			paths = append([]string{configured}, paths...)
		}
		var dll *windows.DLL
		for _, path := range paths {
			var handle windows.Handle
			handle, err = windows.LoadLibraryEx(path, 0, windows.LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|windows.LOAD_LIBRARY_SEARCH_SYSTEM32)
			if err == nil {
				dll = &windows.DLL{Name: path, Handle: handle}
			}
			if err == nil {
				break
			}
		}
		if err != nil {
			library.err = fmt.Errorf("video runtime missing: place libmpv-2.dll beside Aura or set AURA_MPV_DLL: %w", err)
			return
		}
		versionProc, versionErr := dll.FindProc("mpv_client_api_version")
		if versionErr != nil {
			_ = dll.Release()
			library.err = versionErr
			return
		}
		var clientVersion func() uint64
		purego.RegisterFunc(&clientVersion, versionProc.Addr())
		if version := clientVersion(); version>>16 != 2 {
			_ = dll.Release()
			library.err = fmt.Errorf("unsupported libmpv client API: %x", version)
			return
		}
		a := &library.api
		bindings := []struct {
			target any
			name   string
		}{
			{&a.create, "mpv_create"}, {&a.initialize, "mpv_initialize"},
			{&a.option, "mpv_set_option_string"}, {&a.command, "mpv_command"},
			{&a.get, "mpv_get_property"}, {&a.set, "mpv_set_property_string"},
			{&a.getString, "mpv_get_property_string"}, {&a.free, "mpv_free"},
			{&a.wait, "mpv_wait_event"}, {&a.destroy, "mpv_terminate_destroy"},
			{&a.renderCreate, "mpv_render_context_create"}, {&a.renderUpdate, "mpv_render_context_update"},
			{&a.render, "mpv_render_context_render"}, {&a.renderFree, "mpv_render_context_free"},
		}
		for _, b := range bindings {
			proc, err := dll.FindProc(b.name)
			if err != nil {
				library.err = err
				return
			}
			purego.RegisterFunc(b.target, proc.Addr())
		}
	})
	return &library.api, library.err
}

type mpvPlayer struct {
	mu        sync.Mutex
	a         *api
	handle    uintptr
	context   uintptr
	done      chan struct{}
	finished  chan struct{}
	frame     atomic.Pointer[image.RGBA]
	renderErr atomic.Int32
	err       error
	volume    float64
	ended     bool
}

func Open(path string) (Player, error) {
	return openMedia(path, true)
}

func OpenAudio(path string) (Player, error) { return openMedia(path, false) }

func openMedia(path string, withVideo bool) (Player, error) {
	var subtitles []string
	if withVideo {
		var err error
		subtitles, err = ContextSubtitles(path)
		if err != nil {
			subtitles = nil // Optional discovery must not reject playable media.
		}
	}
	a, err := loadAPI()
	if err != nil {
		return nil, err
	}
	p := &mpvPlayer{a: a, handle: a.create(), volume: 0.8, done: make(chan struct{}), finished: make(chan struct{})}
	if p.handle == 0 {
		return nil, errors.New("create libmpv player failed")
	}
	for _, setting := range [][2]string{{"config", "no"}, {"terminal", "no"}, {"vo", "libmpv"}, {"hwdec", "no"}, {"pause", "yes"}, {"keep-open", "yes"}, {"idle", "yes"}, {"audio-display", "no"}, {"osc", "no"}, {"input-default-bindings", "no"}, {"sub-auto", "no"}, {"sub-visibility", "yes"}} {
		if code := a.option(p.handle, setting[0], setting[1]); code < 0 {
			a.destroy(p.handle)
			return nil, fmt.Errorf("set video option %s: libmpv error %d", setting[0], code)
		}
	}
	if !withVideo {
		if code := a.option(p.handle, "vid", "no"); code < 0 {
			a.destroy(p.handle)
			return nil, fmt.Errorf("disable video output: %d", code)
		}
	}
	if code := a.initialize(p.handle); code < 0 {
		a.destroy(p.handle)
		return nil, fmt.Errorf("initialize libmpv: %d", code)
	}
	// Rendering runs independently from client API calls. Never wait on the
	// client mutex from this goroutine (libmpv render threading requirement).
	if withVideo {
		ready := make(chan error, 1)
		go p.renderLoop(ready)
		if err := <-ready; err != nil {
			<-p.finished
			a.destroy(p.handle)
			return nil, err
		}
	} else {
		close(p.finished)
	}
	if err := p.command("loadfile", path, "replace"); err != nil {
		p.Close()
		return nil, err
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		e := a.wait(p.handle, 0.05)
		if e.id == 8 { // MPV_EVENT_FILE_LOADED
			if !withVideo {
				found := false
				for i := 0; i < int(p.number("track-list/count")); i++ {
					if p.text(fmt.Sprintf("track-list/%d/type", i)) == "audio" {
						found = true
						break
					}
				}
				if !found {
					p.Close()
					return nil, errors.New("the selected file has no playable audio stream")
				}
				return p, nil
			}
			var width int64
			if a.get(p.handle, "width", 4, unsafe.Pointer(&width)) < 0 || width <= 0 {
				p.Close()
				return nil, errors.New("the selected file has no playable video stream")
			}
			preferred := ""
			firstLoaded := ""
			for _, sub := range subtitles {
				title := strings.TrimSuffix(filepath.Base(sub), filepath.Ext(sub))
				if err := p.command("sub-add", sub, "auto", title, subtitleLanguage(sub)); err != nil {
					continue // Broken optional subtitles must not abort video.
				}
				if firstLoaded == "" {
					firstLoaded = sub
				}
				if subtitleLanguage(sub) == "spa" && preferred == "" {
					preferred = sub
				}
			}
			if preferred == "" {
				preferred = firstLoaded
			}
			if preferred != "" {
				_ = p.command("sub-add", preferred, "cached")
			}
			return p, nil
		}
		if e.id == 7 { // MPV_EVENT_END_FILE, before FILE_LOADED
			p.Close()
			return nil, errors.New("libmpv could not decode the selected video")
		}
	}
	p.Close()
	return nil, errors.New("timed out opening video")
}

func (p *mpvPlayer) renderLoop(ready chan<- error) {
	defer close(p.finished)
	kind := []byte("sw\x00")
	params := []param{{1, unsafe.Pointer(&kind[0])}, {}}
	code := p.a.renderCreate(&p.context, p.handle, &params[0])
	runtime.KeepAlive(kind)
	if code < 0 {
		ready <- fmt.Errorf("create video renderer: %d", code)
		return
	}
	defer p.a.renderFree(p.context)
	ready <- nil
	ticker := time.NewTicker(time.Second / 60)
	defer ticker.Stop()
	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			if p.a.renderUpdate(p.context)&1 == 0 {
				continue
			}
			// Bound the software render surface. mpv preserves the source aspect
			// ratio with letterboxing; Gio scales this surface to the viewport.
			frame := image.NewRGBA(image.Rect(0, 0, 960, 540))
			size := [2]int32{960, 540}
			stride := uintptr(frame.Stride)
			format := []byte("rgb0\x00")
			params := []param{{17, unsafe.Pointer(&size[0])}, {18, unsafe.Pointer(&format[0])}, {19, unsafe.Pointer(&stride)}, {20, unsafe.Pointer(&frame.Pix[0])}, {}}
			code := p.a.render(p.context, &params[0])
			runtime.KeepAlive(frame)
			runtime.KeepAlive(format)
			if code < 0 {
				p.renderErr.Store(code)
				continue
			}
			for i := 3; i < len(frame.Pix); i += 4 {
				frame.Pix[i] = 255
			}
			// Published frames are immutable, including while Gio uploads them.
			p.frame.Store(frame)
		}
	}
}

func (p *mpvPlayer) command(args ...string) error {
	ptrs := make([]*byte, len(args)+1)
	for i, arg := range args {
		b, err := windows.BytePtrFromString(arg)
		if err != nil {
			return err
		}
		ptrs[i] = b
	}
	code := p.a.command(p.handle, &ptrs[0])
	runtime.KeepAlive(ptrs)
	if code < 0 {
		return fmt.Errorf("video command %s failed: %d", args[0], code)
	}
	return nil
}

func (p *mpvPlayer) set(name, value string) error {
	if p.handle == 0 {
		return errors.New("video player is closed")
	}
	if code := p.a.set(p.handle, name, value); code < 0 {
		return fmt.Errorf("video property %s failed: %d", name, code)
	}
	return nil
}
func (p *mpvPlayer) number(name string) float64 {
	var n float64
	if p.handle != 0 {
		p.a.get(p.handle, name, 5, unsafe.Pointer(&n))
	}
	return n
}
func (p *mpvPlayer) flag(name string) bool {
	var n int32
	if p.handle != 0 {
		p.a.get(p.handle, name, 3, unsafe.Pointer(&n))
	}
	return n != 0
}
func (p *mpvPlayer) Play() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.handle == 0 {
		p.err = errors.New("video player is closed")
		return
	}
	if p.ended || p.flag("eof-reached") {
		p.err = p.command("seek", "0", "absolute+exact")
		p.ended = false
	}
	if p.err == nil {
		p.err = p.set("pause", "no")
	}
}
func (p *mpvPlayer) Pause() { p.mu.Lock(); defer p.mu.Unlock(); p.err = p.set("pause", "yes") }
func (p *mpvPlayer) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.set("pause", "yes"); err != nil {
		return err
	}
	p.err = p.command("seek", "0", "absolute+exact")
	p.ended = false
	return p.err
}
func (p *mpvPlayer) Seek(position time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.handle == 0 {
		return errors.New("video player is closed")
	}
	p.err = p.command("seek", strconv.FormatFloat(max(0, position.Seconds()), 'f', 6, 64), "absolute+exact")
	p.ended = false
	return p.err
}
func (p *mpvPlayer) SetVolume(volume float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if volume < 0 || volume > 1 {
		return errors.New("volume must be between 0 and 1")
	}
	if err := p.set("volume", strconv.FormatFloat(volume*100, 'f', 3, 64)); err != nil {
		return err
	}
	p.volume = volume
	return nil
}
func (p *mpvPlayer) Volume() float64 { p.mu.Lock(); defer p.mu.Unlock(); return p.volume }
func (p *mpvPlayer) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.handle == 0 {
		return false
	}
	for {
		e := p.a.wait(p.handle, 0)
		if e.id == 0 {
			break
		}
		if e.id == 7 && e.data != nil {
			end := (*endFile)(e.data)
			if end.reason == 4 {
				p.err = fmt.Errorf("video playback failed: libmpv error %d", end.err)
			}
			if end.reason == 0 {
				p.ended = true
			}
		}
	}
	return p.err == nil && p.renderErr.Load() >= 0 && !p.ended && !p.flag("pause") && !p.flag("eof-reached")
}
func (p *mpvPlayer) Err() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if code := p.renderErr.Load(); code < 0 {
		return fmt.Errorf("video rendering failed: %d", code)
	}
	return p.err
}
func (p *mpvPlayer) Position() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return time.Duration(p.number("time-pos") * float64(time.Second))
}
func (p *mpvPlayer) Duration() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return time.Duration(p.number("duration") * float64(time.Second))
}
func (p *mpvPlayer) Frame() *image.RGBA { return p.frame.Load() }

func (p *mpvPlayer) AudioFormat() (int, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.handle == 0 {
		return 0, 0
	}
	rate, channels := int(p.number("audio-params/samplerate")), int(p.number("audio-params/channel-count"))
	if rate == 0 || channels == 0 {
		for i := 0; i < min(int(p.number("track-list/count")), 256); i++ {
			prefix := fmt.Sprintf("track-list/%d/", i)
			if p.text(prefix+"type") != "audio" {
				continue
			}
			if rate == 0 {
				rate = int(p.number(prefix + "demux-samplerate"))
			}
			if channels == 0 {
				channels = int(p.number(prefix + "demux-channel-count"))
			}
			break
		}
	}
	return rate, channels
}

func (p *mpvPlayer) LoadSubtitles(path string) error {
	if !IsSubtitle(filepath.Ext(path)) {
		return errors.New("expected an SRT, ASS, SSA or VTT subtitle file")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("inspect subtitles: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > maxSubtitleBytes {
		return errors.New("subtitles must be a regular file no larger than 16 MiB")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.handle == 0 {
		return errors.New("video player is closed")
	}
	if err := p.command("sub-add", abs, "cached"); err != nil {
		return err
	}
	return p.set("sub-visibility", "yes")
}

func (p *mpvPlayer) SetSubtitlesVisible(visible bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	value := "no"
	if visible {
		value = "yes"
	}
	return p.set("sub-visibility", value)
}

func (p *mpvPlayer) SubtitlesVisible() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.flag("sub-visibility")
}

func (p *mpvPlayer) text(name string) string {
	value := p.a.getString(p.handle, name)
	if value == nil {
		return ""
	}
	defer p.a.free(unsafe.Pointer(value))
	return windows.BytePtrToString(value)
}

func (p *mpvPlayer) SubtitleTracks() []SubtitleTrack {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.handle == 0 {
		return nil
	}
	var tracks []SubtitleTrack
	for i := 0; i < min(int(p.number("track-list/count")), 256); i++ {
		prefix := fmt.Sprintf("track-list/%d/", i)
		if p.text(prefix+"type") != "sub" {
			continue
		}
		title := p.text(prefix + "title")
		if title == "" {
			title = filepath.Base(p.text(prefix + "external-filename"))
		}
		if title == "" || title == "." {
			title = "SUBTITLE"
		}
		tracks = append(tracks, SubtitleTrack{ID: int64(p.number(prefix + "id")), Title: title, Language: p.text(prefix + "lang"), Selected: p.flag(prefix + "selected")})
	}
	return tracks
}

func (p *mpvPlayer) SelectSubtitle(id int64) error {
	valid := false
	for _, track := range p.SubtitleTracks() {
		if track.ID == id {
			valid = true
			break
		}
	}
	if !valid {
		return errors.New("subtitle track is not available for this video")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.set("sid", strconv.FormatInt(id, 10)); err != nil {
		return err
	}
	return p.set("sub-visibility", "yes")
}
func (p *mpvPlayer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.handle == 0 {
		return nil
	}
	close(p.done)
	<-p.finished
	p.a.destroy(p.handle)
	p.handle = 0
	p.frame.Store(nil)
	return nil
}
