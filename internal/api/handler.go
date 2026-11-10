// Package api contains the HTTP surface for Ploos AI Gateway.
package api

import (
	"encoding/json"
	"net/http"
)

// NewHandler returns the M0 API handler. Provider routes are intentionally absent.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(struct {
			Status string `json:"status"`
		}{Status: "ok"})
	})
	return mux
}
