// Package audio implements fixed-destination speech synthesis.
package audio

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

const MaxSpeechBytes int64 = 16 << 20

// SpeechProvider generates audio using an OpenAI-compatible speech endpoint.
type SpeechProvider struct {
	Endpoint string
	Key      string
	Client   *http.Client
}

// Synthesize sends a strictly constructed speech request and returns bounded audio.
func (p *SpeechProvider) Synthesize(ctx context.Context, model, text, voice string) ([]byte, string, error) {
	payload, err := json.Marshal(map[string]string{
		"model": model,
		"input": text,
		"voice": voice,
		"response_format": "mp3",
	})
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.Key != "" {
		req.Header.Set("Authorization", "Bearer "+p.Key)
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, "", errors.New("speech provider unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", errors.New("speech provider rejected request")
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0]))
	if contentType != "audio/mpeg" && contentType != "audio/mp3" && contentType != "application/octet-stream" {
		return nil, "", errors.New("unexpected speech response content type")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxSpeechBytes+1))
	if err != nil || len(data) == 0 || int64(len(data)) > MaxSpeechBytes {
		return nil, "", errors.New("invalid speech response")
	}
	return data, "audio/mpeg", nil
}

// NewClient prevents upstream redirects and bounds request duration.
func NewClient(private bool) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if private {
		transport.Proxy = nil
	}
	return &http.Client{
		Timeout: 90 * time.Second,
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
