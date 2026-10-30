// Package radio searches the community Radio Browser directory and ranks styles locally.
package radio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Station struct {
	ID      string `json:"stationuuid"`
	Name    string `json:"name"`
	URL     string `json:"url_resolved"`
	Tags    string `json:"tags"`
	Country string `json:"country"`
	Codec   string `json:"codec"`
}

func ValidURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil
}

type Client struct {
	BaseURL string
	HTTP    *http.Client
	Config  *Config
}

func (c Client) Search(ctx context.Context, style string) ([]Station, error) {
	config := DefaultConfig()
	if c.Config != nil {
		config = *c.Config
	}
	if c.BaseURL != "" {
		config.BaseURL = c.BaseURL
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if config.Provider == "icecast" {
		return c.searchIcecast(ctx, config, style)
	}
	q := url.Values{"hidebroken": {strconv.FormatBool(config.HideBroken)}, "limit": {strconv.Itoa(config.Limit)}, "order": {"clickcount"}, "reverse": {"true"}}
	if config.Country != "" {
		q.Set("countrycode", config.Country)
	}
	if style = strings.TrimSpace(style); style != "" {
		q.Set("tag", style)
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	resp, err := c.searchResponse(ctx, config.BaseURL, q)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Radio Browser: HTTP %d", resp.StatusCode)
	}
	var stations []Station
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&stations); err != nil {
		return nil, err
	}
	result := make([]Station, 0, len(stations))
	seen := map[string]bool{}
	for _, s := range stations {
		if len(result) == config.Limit {
			break
		}
		if ValidURL(s.URL) && !seen[s.URL] {
			result = append(result, s)
			seen[s.URL] = true
		}
	}
	return result, nil
}

// Styles ranks explicit local affinities alongside a small starter catalog.
func Styles(weights map[string]int) []string {
	return SuggestedStyles(DefaultConfig(), weights)
}

func SuggestedStyles(config Config, weights map[string]int) []string {
	styles := append([]string(nil), config.Styles...)
	if !config.Personalized {
		return styles
	}
	var extra []string
	for tag := range weights {
		if !contains(styles, tag) {
			extra = append(extra, tag)
		}
	}
	sort.Strings(extra)
	styles = append(styles, extra...)
	sort.SliceStable(styles, func(i, j int) bool { return weights[styles[i]] > weights[styles[j]] })
	return styles
}
func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

// Score is normalized content affinity across the station's genre tags.
func Score(s Station, weights map[string]int) float64 {
	tags := strings.Split(strings.ToLower(s.Tags), ",")
	score := 0
	for _, t := range tags {
		score += weights[strings.TrimSpace(t)]
	}
	return float64(score) / float64(len(tags))
}
func Rank(stations []Station, weights map[string]int) {
	sort.SliceStable(stations, func(i, j int) bool { return Score(stations[i], weights) > Score(stations[j], weights) })
}
