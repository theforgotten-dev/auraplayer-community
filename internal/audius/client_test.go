package audius

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func publicTrack() Track {
	t := Track{ID: "abc12", Title: "Song", Duration: 120, Permalink: "/artist/song", License: "All rights reserved", IsStreamable: true}
	t.Access.Stream = true
	t.User.Name = "Artist"
	t.CopyrightLine.Year = "2026"
	t.CopyrightLine.Text = "Rights holder"
	return t
}

func testConfig(server *httptest.Server) Config {
	c := DefaultConfig()
	c.APIURL = server.URL + "/v1"
	c.APIKey = "public-application-key"
	return c
}

func TestSearchRejectsRestrictedAndDuplicateTracks(t *testing.T) {
	allowed := publicTrack()
	gated := allowed
	gated.ID = "gated"
	gated.IsStreamGated = true
	denied := allowed
	denied.ID = "denied"
	denied.Access.Stream = false
	private := allowed
	private.ID = "private"
	private.IsUnlisted = true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("query") != "jazz & blues" || q.Get("limit") != "20" || q.Get("app_name") != "AuraPlayerCommunity" || q.Get("api_key") != "public-application-key" {
			t.Error("incorrect query", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": []Track{allowed, gated, denied, private, allowed}})
	}))
	defer server.Close()
	tracks, err := (Client{Config: testConfig(server)}).Search(context.Background(), "jazz & blues")
	if err != nil || len(tracks) != 1 || tracks[0].Copyright() != "© 2026 Rights holder" {
		t.Fatalf("tracks=%v err=%v", tracks, err)
	}
}

func TestResolveRechecksAccessAndBuildsFullStreamEndpoint(t *testing.T) {
	allowed := true
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		track := publicTrack()
		track.Access.Stream = allowed
		json.NewEncoder(w).Encode(map[string]any{"data": track})
	}))
	defer server.Close()
	client := Client{Config: testConfig(server)}
	_, stream, err := client.Resolve(context.Background(), "abc12")
	if err != nil || !strings.Contains(stream, "/tracks/abc12/stream?") || strings.Contains(stream, "preview=") {
		t.Fatalf("stream=%s err=%v", stream, err)
	}
	allowed = false
	if _, _, err = client.Resolve(context.Background(), "abc12"); err == nil {
		t.Fatal("access revocation ignored")
	}
	if _, _, err = client.Resolve(context.Background(), "../tracks"); err == nil || calls != 2 {
		t.Fatal("invalid id reached API")
	}
}

func TestAudiusRetriesAreBoundedAndCancellable(t *testing.T) {
	calls := 0
	var slow atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if slow.Load() {
			w.Header().Set("Retry-After", "3600")
		} else {
			w.Header().Set("Retry-After", "0")
		}
		w.WriteHeader(503)
	}))
	defer server.Close()
	_, err := (Client{Config: testConfig(server)}).Search(context.Background(), "jazz")
	if err == nil || calls != 3 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	slow.Store(true)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = (Client{Config: testConfig(server)}).Search(ctx, "jazz")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancellation ignored: %v", err)
	}
}

func TestArtworkBoundsAndMissingCover(t *testing.T) {
	var small, large bytes.Buffer
	png.Encode(&small, image.NewRGBA(image.Rect(0, 0, 32, 32)))
	png.Encode(&large, image.NewGray(image.Rect(0, 0, 2048, 2048)))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/small":
			w.Write(small.Bytes())
		case "/large":
			w.Write(large.Bytes())
		case "/oversize":
			w.Write(bytes.Repeat([]byte{0}, 2*1024*1024+1))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	client := Client{}
	img, err := client.Artwork(context.Background(), server.URL+"/small")
	if err != nil || img.Bounds().Dx() != 32 {
		t.Fatal("valid thumbnail failed", err)
	}
	for _, suffix := range []string{"/large", "/oversize", "/missing"} {
		if _, err := client.Artwork(context.Background(), server.URL+suffix); err == nil {
			t.Fatal("invalid cover accepted", suffix)
		}
	}
}

func TestConfigAndMetadataSafety(t *testing.T) {
	for _, raw := range []string{"file:///tmp/key", "http://example.com/v1", "https://name:password@example.com/v1", "https://api.audius.co/v1?secret=x"} {
		c := DefaultConfig()
		c.APIURL = raw
		if c.Validate() == nil {
			t.Fatal("unsafe API URL accepted", raw)
		}
	}
	t1 := publicTrack()
	t1.Permalink = "//attacker.example/artist/song"
	if t1.PageURL() != "" {
		t.Fatal("foreign permalink accepted")
	}
	t1 = publicTrack()
	t1.StreamConditions = json.RawMessage(`{"usdc_purchase":{}}`)
	if t1.Streamable() {
		t.Fatal("gate accepted")
	}
	t1 = publicTrack()
	t1.AccessAuthorities = []string{"wallet"}
	if t1.Streamable() {
		t.Fatal("authority gate accepted")
	}
	t1 = publicTrack()
	t1.Title = strings.Repeat("A", 513)
	if t1.Streamable() {
		t.Fatal("oversized metadata accepted")
	}
}

func TestAudiusLiveReadOnly(t *testing.T) {
	if os.Getenv("AURA_AUDIUS_LIVE_TEST") != "1" {
		t.Skip("set AURA_AUDIUS_LIVE_TEST=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	client := Client{Config: DefaultConfig()}
	tracks, err := client.Search(ctx, "jazz")
	if err != nil || len(tracks) == 0 {
		t.Fatalf("search: %v", err)
	}
	track, stream, err := client.Resolve(ctx, tracks[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, stream, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", "bytes=0-1023")
	resp, err := client.httpClient().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		t.Fatal("stream HTTP", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil || len(data) < 64 {
		t.Fatal("stream payload", err)
	}
	if track.ArtworkURL() != "" {
		img, err := client.Artwork(ctx, track.ArtworkURL())
		if err != nil {
			t.Fatal("artwork", err)
		}
		t.Logf("artwork decoded: %v", img.Bounds())
	}
	t.Logf("public track %s; stream HTTP %d; %d bytes inspected, not saved", track.ID, resp.StatusCode, len(data))
}
