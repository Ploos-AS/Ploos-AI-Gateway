package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatAuthentication(t *testing.T) {
	tests := []struct {
		name, configured, authorization string
		want                            int
	}{
		{"missing configuration", "", "Bearer test-secret", http.StatusServiceUnavailable},
		{"missing header", "test-secret", "", http.StatusUnauthorized},
		{"wrong key", "test-secret", "Bearer wrong", http.StatusUnauthorized},
		{"wrong scheme", "test-secret", "Basic test-secret", http.StatusUnauthorized},
		{"valid key", "test-secret", "Bearer test-secret", http.StatusNotImplemented},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader("{}"))
			req.Header.Set("Authorization", tt.authorization)
			res := httptest.NewRecorder()
			NewHandler(tt.configured).ServeHTTP(res, req)
			if res.Code != tt.want {
				t.Fatalf("status = %d, want %d", res.Code, tt.want)
			}
			if got := res.Header().Get("Cache-Control"); got != "no-store" {
				t.Fatalf("cache control = %q", got)
			}
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil || body.Error.Code == "" {
				t.Fatalf("invalid API error: %s (%v)", res.Body.String(), err)
			}
			if strings.Contains(res.Body.String(), "test-secret") {
				t.Fatal("secret leaked in response")
			}
		})
	}
}

func TestHealthDoesNotRequireKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()
	NewHandler("test-secret").ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("health status = %d", res.Code)
	}
}
