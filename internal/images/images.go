// Package images implements fixed-destination image generation.
package images

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

const MaxRequestBytes int64 = 32 << 10
const MaxResponseBytes int64 = 12 << 20

// ImageGenerator is the common image generation contract.
type ImageGenerator interface {
	Generate(ctx context.Context, body []byte) (int, []byte, error)
}

// Generator forwards image requests to a fixed upstream.
type Generator struct {
	Endpoint string
	APIKey   string
	Client   *http.Client
}

// NewOpenAI creates a fixed-endpoint image generator.
func NewOpenAI(key string) (*Generator, error) {
	if key == "" {
		return nil, errors.New("image provider key required")
	}
	return &Generator{
		Endpoint: "https://api.openai.com/v1/images/generations",
		APIKey:   key,
		Client: &http.Client{
			Timeout: 120 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

// Generate returns a bounded upstream JSON response.
func (g *Generator) Generate(ctx context.Context, body []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.Endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.Client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return 0, nil, errors.New("upstream redirect rejected")
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		return 0, nil, errors.New("unexpected upstream content type")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	if err != nil {
		return 0, nil, err
	}
	if int64(len(data)) > MaxResponseBytes || !json.Valid(data) {
		return 0, nil, errors.New("invalid or oversized upstream response")
	}
	return resp.StatusCode, data, nil
}
