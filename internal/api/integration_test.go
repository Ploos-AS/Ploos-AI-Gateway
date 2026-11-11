package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/routing"
)

const integrationChatBody = `{"model":"private/demo","messages":[{"role":"user","content":"hello"}]}`

func integrationHandler(t *testing.T, endpoint string) http.Handler {
	t.Helper()
	cfg := routing.Config{
		Providers: []routing.ProviderConfig{{ID: "private", Kind: "selfhosted", Endpoint: endpoint}},
		Models:    []routing.Model{{ID: "private/demo", Provider: "private", UpstreamID: "upstream-model", Location: "selfhosted", Modalities: []string{"text"}}},
	}
	reg, err := routing.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return NewRoutedHandler("client-key", reg)
}

func integrationRequest(ctx context.Context) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(integrationChatBody)).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer client-key")
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestPrivateModelEndToEndJSON(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/v1/chat/completions" || r.Method != http.MethodPost {
			t.Errorf("unexpected upstream request %s %s", r.Method, r.URL.Path)
		}
		var payload struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if payload.Model != "upstream-model" {
			t.Errorf("upstream model = %q", payload.Model)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`)
	}))
	defer upstream.Close()
	handler := integrationHandler(t, upstream.URL+"/v1/chat/completions")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, integrationRequest(context.Background()))
	if response.Code != http.StatusOK {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"content":"ok"`) {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
	if calls.Load() != 1 {
		t.Fatalf("upstream called %d times", calls.Load())
	}
}

func TestPrivateModelEndToEndSSE(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()
	handler := integrationHandler(t, upstream.URL+"/v1/chat/completions")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, integrationRequest(context.Background()))
	if response.Code != http.StatusOK {
		t.Fatalf("status %d", response.Code)
	}
	if !strings.HasPrefix(response.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("content type %q", response.Header().Get("Content-Type"))
	}
	if !strings.Contains(response.Body.String(), "data: first") || !strings.Contains(response.Body.String(), "data: [DONE]") {
		t.Fatalf("unexpected SSE body %q", response.Body.String())
	}
}

func TestPrivateModelUnavailableNoFallback(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	endpoint := upstream.URL + "/v1/chat/completions"
	upstream.Close()
	handler := integrationHandler(t, endpoint)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, integrationRequest(context.Background()))
	if response.Code != http.StatusBadGateway {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), endpoint) {
		t.Fatal("internal endpoint leaked")
	}
}

func TestPrivateModelCancelStopsUpstream(t *testing.T) {
	started := make(chan struct{})
	finished := make(chan struct{})
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		close(started)
		select {
		case <-r.Context().Done():
		case <-release:
		}
		close(finished)
	}))
	defer upstream.Close()
	defer close(release)
	handler := integrationHandler(t, upstream.URL+"/v1/chat/completions")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.ServeHTTP(httptest.NewRecorder(), integrationRequest(ctx))
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		cancel()
		t.Fatal("upstream did not start")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("gateway did not stop on cancellation")
	}
	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("upstream did not observe cancellation")
	}
}
