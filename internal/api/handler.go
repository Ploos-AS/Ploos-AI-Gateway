// Package api contains the HTTP surface for Ploos AI Gateway.
package api

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/providers"
)

// NewHandler returns the gateway HTTP handler. A blank key disables model routes.
func NewHandler(apiKey ...string) http.Handler {
	key := ""
	if len(apiKey) > 0 {
		key = apiKey[0]
	}
	return NewHandlerWithProvider(key, nil)
}

// NewHandlerWithProvider wires an authenticated chat endpoint to a provider.
func NewHandlerWithProvider(key string, provider providers.ChatProvider) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(struct {
			Status string `json:"status"`
		}{Status: "ok"})
	})
	mux.HandleFunc("POST /v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		if key == "" {
			writeError(w, http.StatusServiceUnavailable, "gateway_not_configured", "Gateway API key is not configured")
			return
		}
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || len(token) != len(key) || subtle.ConstantTimeCompare([]byte(token), []byte(key)) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeError(w, http.StatusUnauthorized, "invalid_api_key", "Invalid API key")
			return
		}
		if provider == nil {
			writeError(w, http.StatusNotImplemented, "provider_unavailable", "No model provider is configured")
			return
		}
		if r.ContentLength > providers.MaxRequestBytes {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "Request body exceeds limit")
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, providers.MaxRequestBytes))
		if err != nil {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "Request body exceeds limit")
			return
		}
		var payload struct {
			Model    string          `json:"model"`
			Messages json.RawMessage `json:"messages"`
		}
		if !json.Valid(body) || json.Unmarshal(body, &payload) != nil || payload.Model == "" || len(payload.Messages) == 0 || payload.Messages[0] != '[' {
			writeError(w, http.StatusBadRequest, "invalid_request", "Expected JSON with model and messages array")
			return
		}
		if err := provider.Complete(r.Context(), body, w); err != nil {
			// Response may already have started during streaming; do not write a second JSON response.
			// A real network failure before headers is handled by the provider's caller in a later revision.
			return
		}
	})
	return mux
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}{
		Error: struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		}{Message: message, Type: "gateway_error", Code: code},
	})
}
