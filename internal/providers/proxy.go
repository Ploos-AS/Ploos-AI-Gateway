// Package providers implements fixed-destination chat completion adapters.
package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/netip"
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
	return newHTTPChatProvider(endpoint, apiKey, false)
}

// NewPrivateHTTPChatProvider connects to an explicitly configured private-IP model server.
func NewPrivateHTTPChatProvider(endpoint string) (*HTTPChatProvider, error) {
	return newHTTPChatProvider(endpoint, "", true)
}

func newHTTPChatProvider(endpoint, apiKey string, allowPrivate bool) (*HTTPChatProvider, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("invalid provider endpoint")
	}
	privateHTTP := false
	if allowPrivate {
		ip, err := netip.ParseAddr(u.Hostname())
		privateHTTP = err == nil && (ip.IsPrivate() || ip.IsLoopback()) && !ip.Is4In6() && ip.Zone() == ""
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" || privateHTTP)) {
		return nil, errors.New("provider endpoint must use HTTPS or local loopback HTTP")
	}
	if !strings.HasSuffix(u.Path, "/chat/completions") {
		return nil, errors.New("provider endpoint must be a chat completions URL")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if allowPrivate {
		transport.Proxy = nil
	}
	return &HTTPChatProvider{
		Endpoint: endpoint,
		APIKey:   apiKey,
		Client: &http.Client{
			Timeout:   120 * time.Second,
			Transport: transport,
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
		writeUpstreamError(w, err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		writeUpstreamError(w, errors.New("upstream redirect rejected"))
		return errors.New("upstream redirect rejected")
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" && !strings.HasPrefix(contentType, "application/json;") && contentType != "text/event-stream" && !strings.HasPrefix(contentType, "text/event-stream;") {
		writeUpstreamError(w, errors.New("unexpected upstream content type"))
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

// writeUpstreamError emits a safe error before any upstream headers or body are forwarded.
// It deliberately never includes provider URLs, credentials or response bodies.
func writeUpstreamError(w http.ResponseWriter, err error) {
	status := http.StatusBadGateway
	code := "upstream_unavailable"
	if errors.Is(err, context.DeadlineExceeded) {
		status = http.StatusGatewayTimeout
		code = "upstream_timeout"
	} else {
		var netErr interface{ Timeout() bool }
		if errors.As(err, &netErr) && netErr.Timeout() {
			status = http.StatusGatewayTimeout
			code = "upstream_timeout"
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{"message": "Model provider is unavailable", "type": "gateway_error", "code": code},
	})
}
