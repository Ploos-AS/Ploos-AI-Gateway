// Package api contains the HTTP surface for Ploos AI Gateway.
package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// NewHandler returns the gateway HTTP handler. A blank key disables model routes.
func NewHandler(apiKey ...string) http.Handler {
	key := ""
	if len(apiKey) > 0 {
		key = apiKey[0]
	}
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
		// Never accept or forward requests before a provider adapter is configured.
		writeError(w, http.StatusNotImplemented, "provider_unavailable", "No model provider is configured")
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
