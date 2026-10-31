package player

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"playerdoit/internal/audius"
)

func publicAudiusTrack() audius.Track {
	t := audius.Track{ID: "abc12", Title: "Song", Permalink: "/artist/song", IsStreamable: true, Duration: 10}
	t.User.Name = "Artist"
	t.Access.Stream = true
	return t
}

func TestAudiusQueueClearPreservesOtherSources(t *testing.T) {
	c := NewController()
	defer c.Close()
	c.queue = []Track{{ID: "local", Title: "Local"}, {ID: "radio:one", Live: true}}
	c.currentIndex = 0
	track := publicAudiusTrack()
	config := audius.DefaultConfig()
	index, err := c.QueueAudius(config, track)
	if err != nil || index != 2 {
		t.Fatal(index, err)
	}
	index, err = c.QueueAudius(config, track)
	if err != nil || index != 2 || len(c.queue) != 3 {
		t.Fatal("duplicate queued", index, err)
	}
	if c.queue[2].Live || !c.queue[2].LibMPV || c.queue[2].Artist != "Artist" || !strings.HasPrefix(c.queue[2].Path, "https://audius.co/") {
		t.Fatal("incorrect source metadata")
	}
	if err = c.ClearAudius(); err != nil {
		t.Fatal(err)
	}
	if len(c.queue) != 2 || c.currentIndex != 0 || len(c.audiusSources) != 0 {
		t.Fatal("clearing lost other sources")
	}
}

func TestAudiusAccessIsRecheckedBeforeNativeLoad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		track := publicAudiusTrack()
		track.Access.Stream = false
		json.NewEncoder(w).Encode(map[string]any{"data": track})
	}))
	defer server.Close()
	config := audius.DefaultConfig()
	config.APIURL = server.URL + "/v1"
	c := NewController()
	defer c.Close()
	if err := c.PlayAudius(config, publicAudiusTrack()); err == nil || !strings.Contains(err.Error(), "streaming público") {
		t.Fatalf("revoked track reached media loader: %v", err)
	}
}

func TestAudiusPlaybackLive(t *testing.T) {
	if os.Getenv("AURA_AUDIUS_LIVE_TEST") != "1" {
		t.Skip("set AURA_AUDIUS_LIVE_TEST=1 with AURA_MPV_DLL")
	}
	client := audius.Client{Config: audius.DefaultConfig()}
	tracks, err := client.Search(context.Background(), "jazz")
	if err != nil || len(tracks) == 0 {
		t.Fatal("search", err)
	}
	c := NewController()
	defer c.Close()
	if err := c.SetVolume(0); err != nil {
		t.Fatal(err)
	}
	if err := c.PlayAudius(client.Config, tracks[0]); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		s := c.Snapshot()
		if s.Position > .1 {
			if err := c.Toggle(); err != nil {
				t.Fatal(err)
			}
			if err := c.Seek(1); err != nil {
				t.Fatal(err)
			}
			if err := c.ClearAudius(); err != nil {
				t.Fatal(err)
			}
			if len(c.Snapshot().Queue) != 0 {
				t.Fatal("session metadata retained")
			}
			t.Log("muted public Audius playback advanced; pause, seek and clear succeeded")
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatal("remote playback did not advance")
}
