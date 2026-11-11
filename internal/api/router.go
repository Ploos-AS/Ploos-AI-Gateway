package api

import (
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
		if json.Unmarshal(body, &payload) != nil || payload.Model == "" || !validMessages(payload.Messages) {
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
		if err := route.Provider.Complete(r.Context(), forwarded, w); err != nil {
			return
		}
	})
	return mux
}

 // validMessages requires a non-empty JSON array with object entries.
 // Detailed per-provider message schemas remain a separate compatibility concern.
 func validMessages(raw json.RawMessage) bool {
  var messages []json.RawMessage
  if json.Unmarshal(raw, &messages) != nil || len(messages) == 0 {
   return false
  }
  for _, item := range messages {
   var obj map[string]json.RawMessage
   if json.Unmarshal(item, &obj) != nil || obj == nil {
    return false
   }
   var role string
   if json.Unmarshal(obj["role"], &role) != nil || role == "" {
    return false
   }
  }
  return true
 }
