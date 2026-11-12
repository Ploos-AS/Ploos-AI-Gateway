package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/video"
)

// WithVideoConnector exposes read-only connector capability discovery.
func WithVideoConnector(next http.Handler, key string, connector *video.Connector) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/video/capabilities", func(w http.ResponseWriter, r *http.Request) {
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
		result, err := connector.Capabilities(r.Context())
		if err != nil {
			writeError(w, http.StatusBadGateway, "video_connector_unavailable", "Video connector unavailable")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]json.RawMessage{"result": result})
	})
	mux.Handle("/", next)
	return mux
}
