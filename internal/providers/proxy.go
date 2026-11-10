// Package providers implements fixed-destination chat completion adapters.
package providers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const MaxRequestBytes int64 = 1 << 20

// ChatProvider forwards a validated OpenAI-style chat request.
type ChatProvider interface {
	Complete(ctx context.Context, body []byte, w http.ResponseWriter) error
}

// HTTPChatProvider supports OpenRouter and Ollama's OpenAI-compatible endpoint.
type HTTPChatProvider struct {
	Endpoint string
	APIKey   string
	Client   *http.Client
}

func NewHTTPChatProvider(endpoint, apiKey string) (*HTTPChatProvider, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("invalid provider endpoint")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost")) {
		return nil, errors.New("provider endpoint must use HTTPS or local loopback HTTP")
	}
	if !strings.HasSuffix(u.Path, "/chat/completions") {
		return nil, errors.New("provider endpoint must be a chat completions URL")
	}
	return &HTTPChatProvider{
		Endpoint: endpoint,
		APIKey: apiKey,
		Client: &http.Client{
			Timeout: 120 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

// Complete streams the upstream response, preserving status and content type.
// The caller must authenticate and validate the request first.
func (p *HTTPChatProvider) Complete(ctx context.Context, body []byte, w http.ResponseWriter) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return fmt.Errorf("upstream request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return errors.New("upstream redirect rejected")
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" && !strings.HasPrefix(contentType, "application/json;") && contentType != "text/event-stream" && !strings.HasPrefix(contentType, "text/event-stream;") {
		return errors.New("unexpected upstream content type")
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(resp.StatusCode)
	buf := make([]byte, 16*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := w.Write(buf[:n]); err != nil {
				return err
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}
