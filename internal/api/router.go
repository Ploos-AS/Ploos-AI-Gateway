package api

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/providers"
	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/routing"
)

// NewRoutedHandler serves authenticated model-based routing with no fallback.
func NewRoutedHandler(key string, registry *routing.Registry) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = io.WriteString(w, "{\"status\":\"ok\"}\n")
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
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, providers.MaxRequestBytes))
		if err != nil {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "Request body exceeds limit")
			return
		}
		var payload struct {
			Model    string          `json:"model"`
			Messages json.RawMessage `json:"messages"`
		}
		if json.Unmarshal(body, &payload) != nil || payload.Model == "" || len(payload.Messages) == 0 || payload.Messages[0] != '[' {
			writeError(w, http.StatusBadRequest, "invalid_request", "Expected model and messages array")
			return
		}
		route, found := registry.Resolve(payload.Model)
		if !found {
			writeError(w, http.StatusBadRequest, "unknown_model", "Model alias is not configured")
			return
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal(body, &object); err != nil || object == nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Expected a JSON object")
			return
		}
		upstreamModel, _ := json.Marshal(route.Model.UpstreamID)
		object["model"] = upstreamModel
		forwarded, err := json.Marshal(object)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Could not encode request")
			return
		}
		// Buffer headers until the provider responds; do not send a second response after streaming starts.
		_ = bytes.NewReader(forwarded)
		if err := route.Provider.Complete(r.Context(), forwarded, w); err != nil {
			return
		}
	})
	return mux
}
