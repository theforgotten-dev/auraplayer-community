package radio

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// searchIcecast streams the public XML feed without loading the whole catalog.
func (c Client) searchIcecast(ctx context.Context, config Config, style string) ([]Station, error) {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", config.BaseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "AuraPlayer/1.0")
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Icecast: HTTP %d", resp.StatusCode)
	}
	decoder := xml.NewDecoder(io.LimitReader(resp.Body, 32<<20))
	result := []Station{}
	seen := map[string]bool{}
	style = strings.ToLower(strings.TrimSpace(style))
	root, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	// Skip declarations and whitespace before the directory element.
	for {
		if start, ok := root.(xml.StartElement); ok {
			if start.Name.Local != "directory" {
				return nil, fmt.Errorf("Icecast: se esperaba un directorio XML")
			}
			break
		}
		root, err = decoder.Token()
		if err != nil {
			return nil, err
		}
	}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return nil, io.ErrUnexpectedEOF
		}
		if err != nil {
			return nil, err
		}
		if end, ok := token.(xml.EndElement); ok && end.Name.Local == "directory" {
			return result, nil
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "entry" {
			continue
		}
		var entry struct {
			Name  string `xml:"server_name"`
			URL   string `xml:"listen_url"`
			Genre string `xml:"genre"`
			Codec string `xml:"server_type"`
		}
		if err := decoder.DecodeElement(&entry, &start); err != nil {
			return nil, err
		}
		if !ValidURL(entry.URL) || seen[entry.URL] {
			continue
		}
		if style != "" && !strings.Contains(strings.ToLower(entry.Genre+" "+entry.Name), style) {
			continue
		}
		seen[entry.URL] = true
		result = append(result, Station{ID: entry.URL, Name: entry.Name, URL: entry.URL, Tags: strings.Join(strings.Fields(entry.Genre), ","), Codec: entry.Codec})
		if len(result) >= config.Limit {
			return result, nil
		}
	}
}
