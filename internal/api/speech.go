package api

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/audio"
)

// WithSpeech adds an authenticated OpenAI-compatible speech synthesis endpoint.
func WithSpeech(next http.Handler, key string, registry *audio.Registry) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/audio/speech", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if key == "" {
			writeError(w, http.StatusServiceUnavailable, "gateway_not_configured", "Gateway API key is not configured")
			return
		}
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || len(token) != len(key) || subtle.ConstantTimeCompare([]byte(token), []byte(key)) != 1 {
			writeError(w, http.StatusUnauthorized, "invalid_api_key", "Invalid API key")
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 16<<10))
		if err != nil {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "Speech request too large")
			return
		}
		var input struct {
			Model string `json:"model"`
			Input string `json:"input"`
			Voice string `json:"voice"`
		}
		if json.Unmarshal(body, &input) != nil || input.Model == "" || strings.TrimSpace(input.Input) == "" || len(input.Input) > 4096 || input.Voice == "" || len(input.Voice) > 64 {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid speech request")
			return
		}
		route, found := registry.Resolve(input.Model)
		if !found {
			writeError(w, http.StatusBadRequest, "unknown_model", "Speech model is not configured")
			return
		}
		data, mime, err := route.Provider.Synthesize(r.Context(), route.UpstreamModel, input.Input, input.Voice)
		if err != nil {
			writeError(w, http.StatusBadGateway, "speech_upstream_unavailable", "Speech provider unavailable")
			return
		}
		w.Header().Set("Content-Type", mime)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
	mux.Handle("/", next)
	return mux
}
