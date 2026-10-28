package radio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestSearchFiltersAndValidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tag") != "jazz & blues" || r.URL.Query().Get("hidebroken") != "true" {
			t.Error(r.URL)
		}
		w.Write([]byte(`[{"stationuuid":"a","url_resolved":"https://radio.example/live"},{"stationuuid":"b","url_resolved":"file:///tmp/audio"},{"stationuuid":"c","url_resolved":"https://radio.example/live"}]`))
	}))
	defer server.Close()
	stations, err := (Client{BaseURL: server.URL}).Search(context.Background(), "jazz & blues")
	if err != nil || len(stations) != 1 {
		t.Fatalf("stations=%v err=%v", stations, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (Client{BaseURL: server.URL}).Search(ctx, ""); err == nil {
		t.Fatal("ignored cancellation")
	}
}

func TestConfiguredSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "1" || q.Get("countrycode") != "CL" || q.Get("hidebroken") != "false" {
			t.Error(r.URL)
		}
		w.Write([]byte(`[{"url_resolved":"https://example.org/one"},{"url_resolved":"https://example.org/two"}]`))
	}))
	defer server.Close()
	c := DefaultConfig()
	c.BaseURL = server.URL
	c.Limit = 1
	c.Country = "CL"
	c.HideBroken = false
	stations, err := (Client{Config: &c}).Search(context.Background(), "")
	if err != nil || len(stations) != 1 {
		t.Fatalf("%v %v", stations, err)
	}
	c.Personalized = false
	c.Styles = []string{"ambient"}
	if styles := SuggestedStyles(c, map[string]int{"rock": 10}); len(styles) != 1 || styles[0] != "ambient" {
		t.Fatal(styles)
	}
}

func TestIcecastFeed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<?xml version="1.0"?><directory><entry><server_name>Jazz Test</server_name><genre>jazz blues</genre><listen_url>https://example.org/live</listen_url><server_type>audio/mpeg</server_type></entry><entry><server_name>Missing URL</server_name></entry><entry><genre>rock</genre><listen_url>https://example.org/rock</listen_url></entry></directory>`))
	}))
	defer server.Close()
	c := DefaultConfig()
	c.Provider = "icecast"
	c.BaseURL = server.URL
	stations, err := (Client{Config: &c}).Search(context.Background(), "jazz")
	if err != nil || len(stations) != 1 || stations[0].Tags != "jazz,blues" {
		t.Fatalf("%v %v", stations, err)
	}
	c.Country = "CL"
	if c.Validate() == nil {
		t.Fatal("accepted unsupported country filter")
	}
}

func TestIcecastLiveDirectory(t *testing.T) {
	if os.Getenv("AURA_RADIO_LIVE_TEST") != "1" {
		t.Skip("set AURA_RADIO_LIVE_TEST=1")
	}
	c := DefaultConfig()
	c.Provider = "icecast"
	c.BaseURL = DefaultIcecastURL
	c.Limit = 3
	stations, err := (Client{Config: &c}).Search(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(stations) == 0 {
		t.Fatal("no usable stream URLs")
	}
	for _, s := range stations {
		t.Logf("Icecast: %s (%s)", s.Name, s.Codec)
	}
}

func TestAffinityAndStableRanking(t *testing.T) {
	weights := map[string]int{"jazz": 4, "rock": -2}
	stations := []Station{{Name: "rock", Tags: "rock"}, {Name: "mixed", Tags: "jazz,rock"}, {Name: "jazz", Tags: " Jazz "}, {Name: "unknown"}}
	Rank(stations, weights)
	if stations[0].Name != "jazz" || stations[1].Name != "mixed" || stations[3].Name != "rock" {
		t.Fatal(stations)
	}
	if Styles(weights)[0] != "jazz" {
		t.Fatal(Styles(weights))
	}
}
