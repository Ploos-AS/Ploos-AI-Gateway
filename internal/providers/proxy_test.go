package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUpstreamConnectionFailureIsSafeJSON(t *testing.T) {
	p, err := NewHTTPChatProvider("http://127.0.0.1:1/v1/chat/completions", "provider-secret")
	if err != nil {
		t.Fatal(err)
	}
	p.Client.Timeout = time.Second
	res := httptest.NewRecorder()
	_ = p.Complete(context.Background(), []byte(`{"model":"test","messages":[]}`), res)
	if res.Code != http.StatusBadGateway {
		t.Fatalf("status %d, want 502", res.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["error"]; !ok {
		t.Fatal("missing error")
	}
	if res.Body.String() == "" {
		t.Fatal("missing response")
	}
}

func TestUpstreamTimeout(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer upstream.Close()
	defer close(release)
	p, err := NewHTTPChatProvider(upstream.URL+"/v1/chat/completions", "")
	if err != nil {
		t.Fatal(err)
	}
	p.Client.Timeout = 20 * time.Millisecond
	res := httptest.NewRecorder()
	_ = p.Complete(context.Background(), []byte(`{}`), res)
	if res.Code != http.StatusGatewayTimeout {
		t.Fatalf("status %d, want 504", res.Code)
	}
}
