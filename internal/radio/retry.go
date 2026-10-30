package radio

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type cancelBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *cancelBody) Close() error {
	defer b.cancel()
	return b.ReadCloser.Close()
}

// Custom directories remain on their configured host. Only official mirrors
// may fail over to the public Radio Browser pool.
func browserBases(base string) []string {
	base = strings.TrimRight(base, "/")
	bases := []string{base}
	u, err := url.Parse(base)
	if err == nil && u.Scheme == "https" && u.Port() == "" && u.Path == "" &&
		strings.HasSuffix(u.Hostname(), ".api.radio-browser.info") {
		for _, mirror := range []string{"https://all.api.radio-browser.info", "https://de1.api.radio-browser.info"} {
			if mirror != base {
				bases = append(bases, mirror)
			}
		}
	}
	return bases
}

func retryDelay(value string, attempt int, now time.Time) time.Duration {
	// Bound parsing before conversion to prevent duration overflow.
	if seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil {
		return max(0, date.Sub(now))
	}
	return (250 * time.Millisecond << attempt) + time.Duration(rand.IntN(100))*time.Millisecond
}

func (c Client) searchResponse(ctx context.Context, base string, query url.Values) (*http.Response, error) {
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	bases := browserBases(base)
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		requestCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, bases[attempt%len(bases)]+"/json/stations/search?"+query.Encode(), nil)
		if err != nil {
			cancel()
			return nil, err
		}
		req.Header.Set("User-Agent", "AuraPlayerCommunity/0.1.1")
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode != 429 && resp.StatusCode != 502 && resp.StatusCode != 503 && resp.StatusCode != 504 {
			// Keep the request context alive until Search finishes reading.
			resp.Body = &cancelBody{ReadCloser: resp.Body, cancel: cancel}
			return resp, nil
		}
		delay := retryDelay("", attempt, time.Now())
		lastErr = err
		if resp != nil {
			lastErr = fmt.Errorf("Radio Browser: HTTP %d", resp.StatusCode)
			delay = retryDelay(resp.Header.Get("Retry-After"), attempt, time.Now())
			resp.Body.Close()
		}
		cancel()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if attempt == 3 {
			break
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, fmt.Errorf("Radio Browser no disponible tras 4 intentos: %w", lastErr)
}
