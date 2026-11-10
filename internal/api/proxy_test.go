package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/providers"
)

func TestProxyAndAuthentication(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer upstream-secret" {
			t.Errorf("missing upstream authorization")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"hello"}}]}`)
	}))
	defer upstream.Close()
	p, err := providers.NewHTTPChatProvider(upstream.URL+"/v1/chat/completions", "upstream-secret")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandlerWithProvider("client-secret", p)
	for _, tc := range []struct {
		key  string
		body string
		want int
	}{
		{"wrong", `{"model":"test","messages":[]}`, http.StatusUnauthorized},
		{"client-secret", `{"model":"test"}`, http.StatusBadRequest},
		{"client-secret", `{"model":"test","messages":[]}`, http.StatusOK},
	} {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(tc.body))
		req.Header.Set("Authorization", "Bearer "+tc.key)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != tc.want {
			t.Fatalf("status = %d, want %d: %s", res.Code, tc.want, res.Body.String())
		}
	}
	if calls != 1 {
		t.Fatalf("upstream calls = %d, want 1", calls)
	}
}

func TestProviderRejectsInsecureRemoteEndpoint(t *testing.T) {
	if _, err := providers.NewHTTPChatProvider("http://example.com/v1/chat/completions", ""); err == nil {
		t.Fatal("expected insecure remote endpoint to be rejected")
	}
}
