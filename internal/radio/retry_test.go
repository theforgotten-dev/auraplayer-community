package radio

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type radioTransport func(*http.Request) (*http.Response, error)

func (f radioTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSearchFailsOverOfficialMirror(t *testing.T) {
	var hosts []string
	client := &http.Client{Transport: radioTransport(func(r *http.Request) (*http.Response, error) {
		hosts = append(hosts, r.URL.Host)
		response := &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[]`))}
		if len(hosts) == 1 {
			response.StatusCode = 503
			response.Header.Set("Retry-After", "0")
		}
		return response, nil
	})}
	config := DefaultConfig()
	config.Mirrors = []string{"https://mirror.example"}
	_, err := (Client{HTTP: client, Config: &config}).Search(context.Background(), "")
	if err != nil || len(hosts) != 2 || hosts[0] == hosts[1] {
		t.Fatalf("hosts=%v err=%v", hosts, err)
	}
}

func TestSearchStopsAfterThreeFailures(t *testing.T) {
	calls := 0
	var hosts []string
	client := &http.Client{Transport: radioTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		hosts = append(hosts, r.URL.Host)
		return &http.Response{StatusCode: 503, Header: http.Header{"Retry-After": {"0"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	config := DefaultConfig()
	config.Mirrors = []string{DefaultBaseURL, "https://mirror1.example", "https://mirror2.example", "https://unused.example"}
	_, err := (Client{HTTP: client, Config: &config}).Search(context.Background(), "")
	if err == nil || calls != 3 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	if strings.Join(hosts, ",") != "de1.api.radio-browser.info,mirror1.example,mirror2.example" {
		t.Fatalf("configured order not respected: %v", hosts)
	}
}

func TestSearchRecoversFrom503(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(503)
			return
		}
		w.Write([]byte(`[{"url_resolved":"https://example.org/live"}]`))
	}))
	defer server.Close()
	stations, err := (Client{BaseURL: server.URL}).Search(context.Background(), "jazz")
	if err != nil || len(stations) != 1 || calls != 3 {
		t.Fatalf("stations=%v calls=%d err=%v", stations, calls, err)
	}
}

func TestSearchCancelsRetryAfter(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(503)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := (Client{BaseURL: server.URL}).Search(ctx, "")
	if !errors.Is(err, context.DeadlineExceeded) || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestSearchDoesNotRetry404(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(404)
	}))
	defer server.Close()
	_, err := (Client{BaseURL: server.URL}).Search(context.Background(), "")
	if err == nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestRetryDelayAndMirrors(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	if retryDelay("2", 0, now) != 2*time.Second || retryDelay(now.Add(3*time.Second).Format(http.TimeFormat), 0, now) != 3*time.Second {
		t.Fatal("Retry-After seconds/date not respected")
	}
	for attempt := 0; attempt < 3; attempt++ {
		delay := retryDelay("invalid", attempt, now)
		if delay < 250*time.Millisecond<<attempt || delay >= (250*time.Millisecond<<attempt)+100*time.Millisecond {
			t.Fatal("invalid exponential backoff", delay)
		}
	}
	config := DefaultConfig()
	config.BaseURL = "https://custom.example"
	config.Mirrors = nil
	if len(browserBases(config)) != 1 {
		t.Fatal("incorrect mirror selection")
	}
}
