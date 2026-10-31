package audius

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"time"

	_ "golang.org/x/image/webp"
)

// Artwork decodes a bounded thumbnail. The caller retains it in session memory.
func (c Client) Artwork(ctx context.Context, rawURL string) (image.Image, error) {
	u, err := url.Parse(rawURL)
	if err != nil || !safeURL(u) {
		return nil, errors.New("Audius: URL de carátula inválida")
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, errors.New("Audius: carátula no disponible")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Audius: carátula no disponible")
	}
	const maxBytes = 2 * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxBytes {
		return nil, errors.New("Audius: carátula demasiado grande")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > 2048 || config.Height > 2048 || int64(config.Width)*int64(config.Height) > 1024*1024 {
		return nil, errors.New("Audius: dimensiones de carátula fuera de rango")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}
