// Package audius provides read-only access to the public Audius music API.
package audius

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	DefaultAPI      = "https://api.audius.co/v1"
	TermsURL        = "https://audius.co/documents/ApiTerms.pdf"
	PrivacyURL      = "https://audius.co/documents/PrivacyPolicy.pdf"
	MusicLicenseURL = "https://audius.org/open-music-license.pdf"
)

type Config struct {
	APIURL  string `json:"apiURL"`
	AppName string `json:"appName"`
	APIKey  string `json:"apiKey,omitempty"` // Public application key, never a bearer token.
}

func DefaultConfig() Config { return Config{APIURL: DefaultAPI, AppName: "AuraPlayerCommunity"} }

func (c Config) Validate() error {
	u, err := url.Parse(c.APIURL)
	if err != nil || !safeURL(u) || u.RawQuery != "" || u.Fragment != "" || len(c.APIURL) > 2048 {
		return errors.New("Audius: configura una API HTTPS sin consulta ni fragmento")
	}
	if strings.TrimSpace(c.AppName) == "" || len(c.AppName) > 80 || strings.ContainsFunc(c.AppName, unicode.IsControl) {
		return errors.New("Audius: nombre de aplicación inválido")
	}
	if len(c.APIKey) > 256 || strings.ContainsFunc(c.APIKey, unicode.IsSpace) || strings.ContainsFunc(c.APIKey, unicode.IsControl) {
		return errors.New("Audius: clave pública de API inválida; no uses un Bearer Token")
	}
	return nil
}

// HTTP loopback is allowed for local contract tests, never remote plaintext.
func safeURL(u *url.URL) bool {
	if u == nil || u.Hostname() == "" || u.User != nil {
		return false
	}
	return u.Scheme == "https" || (u.Scheme == "http" && net.ParseIP(u.Hostname()).IsLoopback())
}

func validID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

type Track struct {
	CopyrightLine struct {
		Year string `json:"year"`
		Text string `json:"text"`
	} `json:"copyright_line"`
	ProducerCopyrightLine struct {
		Year string `json:"year"`
		Text string `json:"text"`
	} `json:"producer_copyright_line"`
	ID                string          `json:"id"`
	Title             string          `json:"title"`
	Duration          int             `json:"duration"`
	Permalink         string          `json:"permalink"`
	License           string          `json:"license"`
	IsStreamable      bool            `json:"is_streamable"`
	IsStreamGated     bool            `json:"is_stream_gated"`
	IsUnlisted        bool            `json:"is_unlisted"`
	IsDeleted         bool            `json:"is_delete"`
	StreamConditions  json.RawMessage `json:"stream_conditions"`
	AccessAuthorities []string        `json:"access_authorities"`
	Access            struct {
		Stream bool `json:"stream"`
	} `json:"access"`
	User struct {
		Name   string `json:"name"`
		Handle string `json:"handle"`
	} `json:"user"`
	Artwork struct {
		Small  string `json:"150x150"`
		Medium string `json:"480x480"`
	} `json:"artwork"`
}

func (t Track) Streamable() bool {
	if !validID(t.ID) || t.Title == "" || len(t.Title) > 512 || t.Artist() == "" || len(t.Artist()) > 256 {
		return false
	}
	if len(t.License) > 512 || len(t.CopyrightLine.Text) > 512 || len(t.ProducerCopyrightLine.Text) > 512 || strings.ContainsFunc(t.Title+t.Artist(), unicode.IsControl) {
		return false
	}
	conditions := strings.TrimSpace(string(t.StreamConditions))
	return t.IsStreamable && t.Access.Stream && !t.IsStreamGated && !t.IsUnlisted && !t.IsDeleted && len(t.AccessAuthorities) == 0 && (conditions == "" || conditions == "null" || conditions == "{}")
}

func (t Track) Copyright() string {
	var notices []string
	if t.CopyrightLine.Text != "" {
		notices = append(notices, strings.TrimSpace("© "+t.CopyrightLine.Year+" "+t.CopyrightLine.Text))
	}
	if t.ProducerCopyrightLine.Text != "" {
		notices = append(notices, strings.TrimSpace("℗ "+t.ProducerCopyrightLine.Year+" "+t.ProducerCopyrightLine.Text))
	}
	return strings.Join(notices, " · ")
}

func (t Track) Artist() string {
	if t.User.Name != "" {
		return t.User.Name
	}
	return t.User.Handle
}

func (t Track) PageURL() string {
	if t.Permalink == "" || len(t.Permalink) > 2048 {
		return ""
	}
	u, err := url.Parse(t.Permalink)
	if err != nil {
		return ""
	}
	if !u.IsAbs() {
		u = (&url.URL{Scheme: "https", Host: "audius.co"}).ResolveReference(u)
	}
	if u.Scheme != "https" || u.Host != "audius.co" || u.User != nil || !strings.HasPrefix(u.Path, "/") {
		return ""
	}
	return u.String()
}

func (t Track) ArtworkURL() string {
	if t.Artwork.Small != "" {
		return t.Artwork.Small
	}
	return t.Artwork.Medium
}

type Client struct {
	Config Config
	HTTP   *http.Client
}

func (c Client) endpoint(path string, q url.Values) (string, error) {
	if err := c.Config.Validate(); err != nil {
		return "", err
	}
	if q == nil {
		q = url.Values{}
	}
	q.Set("app_name", c.Config.AppName)
	if c.Config.APIKey != "" {
		q.Set("api_key", c.Config.APIKey)
	}
	return strings.TrimRight(c.Config.APIURL, "/") + path + "?" + q.Encode(), nil
}

func (c Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || !safeURL(req.URL) {
			return errors.New("Audius: redirección no permitida")
		}
		return nil
	}}
}

func (c Client) get(ctx context.Context, path string, query url.Values, target any) error {
	endpoint, err := c.endpoint(path, query)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "AuraPlayerCommunity")
		resp, err := c.httpClient().Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return errors.New("Audius: no se pudo conectar con la API")
		}
		if resp.StatusCode == http.StatusOK {
			data, readErr := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024+1))
			resp.Body.Close()
			if readErr != nil {
				return readErr
			}
			if len(data) > 2*1024*1024 {
				return errors.New("Audius: respuesta demasiado grande")
			}
			return json.Unmarshal(data, target)
		}
		status, retry := resp.StatusCode, resp.Header.Get("Retry-After")
		resp.Body.Close()
		if attempt == 2 || (status != 429 && status != 502 && status != 503 && status != 504) {
			return fmt.Errorf("Audius: HTTP %d", status)
		}
		delay := 250 * time.Millisecond << attempt
		if n, e := strconv.ParseInt(retry, 10, 32); e == nil && n >= 0 {
			delay = time.Duration(n) * time.Second
		} else if when, e := http.ParseTime(retry); e == nil {
			delay = max(0, time.Until(when))
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return errors.New("Audius: consulta incompleta")
}

func (c Client) Search(ctx context.Context, query string) ([]Track, error) {
	query = strings.TrimSpace(query)
	if query == "" || len(query) > 200 {
		return nil, errors.New("Audius: escribe una búsqueda de hasta 200 caracteres")
	}
	var response struct {
		Data []Track `json:"data"`
	}
	if err := c.get(ctx, "/tracks/search", url.Values{"query": {query}, "limit": {"20"}}, &response); err != nil {
		return nil, err
	}
	result := make([]Track, 0, 20)
	seen := map[string]bool{}
	for _, track := range response.Data {
		if len(result) == 20 {
			break
		}
		if track.Streamable() && !seen[track.ID] && track.PageURL() != "" {
			result = append(result, track)
			seen[track.ID] = true
		}
	}
	return result, nil
}

// Resolve rechecks current access before each new playback, never bypasses gates.
func (c Client) Resolve(ctx context.Context, id string) (Track, string, error) {
	if !validID(id) {
		return Track{}, "", errors.New("Audius: identificador inválido")
	}
	var response struct {
		Data Track `json:"data"`
	}
	if err := c.get(ctx, "/tracks/"+id, nil, &response); err != nil {
		return Track{}, "", err
	}
	if response.Data.ID != id || !response.Data.Streamable() || response.Data.PageURL() == "" {
		return Track{}, "", errors.New("Audius: la pista ya no permite streaming público completo")
	}
	stream, err := c.endpoint("/tracks/"+id+"/stream", nil)
	return response.Data, stream, err
}
