package api

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/images"
)

// ImageRoute binds a public alias to one fixed upstream model.
type ImageRoute struct {
	Alias         string
	UpstreamModel string
	Generator     *images.Generator
}

// WithImageGeneration adds an authenticated text-to-image endpoint.
func WithImageGeneration(chat http.Handler, key string, route ImageRoute) http.Handler {
	return WithImageRegistry(chat, key, func(id string) (ImageRoute, bool) { return route, id == route.Alias })
}

// WithImageRegistry resolves each image request against explicit model aliases.
func WithImageRegistry(chat http.Handler, key string, resolve func(string) (ImageRoute, bool)) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/images/generations", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
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
		if resolve == nil {
			writeError(w, http.StatusServiceUnavailable, "image_provider_unavailable", "Image generation is not configured")
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, images.MaxRequestBytes))
		if err != nil {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "Image request too large")
			return
		}
		var input struct {
			Model  string `json:"model"`
			Prompt string `json:"prompt"`
			N      int    `json:"n"`
		}
		if json.Unmarshal(body, &input) != nil || input.Model == "" || strings.TrimSpace(input.Prompt) == "" || len(input.Prompt) > 4000 || input.N < 0 || input.N > 1 {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid image generation request")
			return
		}
		route, found := resolve(input.Model)
		if !found {
			writeError(w, http.StatusBadRequest, "unknown_model", "Image model is not configured")
			return
		}
		if route.Generator == nil || route.UpstreamModel == "" {
			writeError(w, http.StatusServiceUnavailable, "image_provider_unavailable", "Image generation is not configured")
			return
		}
		encoded, _ := json.Marshal(map[string]interface{}{"model": route.UpstreamModel, "prompt": input.Prompt, "n": 1})
		status, response, err := route.Generator.Generate(r.Context(), encoded)
		if err != nil {
			writeError(w, http.StatusBadGateway, "image_upstream_unavailable", "Image provider unavailable")
			return
		}
		if status < 200 || status >= 300 {
			writeError(w, http.StatusBadGateway, "image_upstream_error", "Image provider rejected request")
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	})
	mux.Handle("/", chat)
	return mux
}
