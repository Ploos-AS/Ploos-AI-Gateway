package api

import (
	"crypto/subtle"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/audio"
)

func WithTranscriptions(next http.Handler, key string, registry *audio.TranscriptionRegistry) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/audio/transcriptions", func(w http.ResponseWriter, r *http.Request) {
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
		r.Body = http.MaxBytesReader(w, r.Body, audio.MaxTranscriptionBytes+(1<<20))
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid multipart audio request")
			return
		}
		model := r.FormValue("model")
		route, found := registry.Resolve(model)
		if !found {
			writeError(w, http.StatusBadRequest, "unknown_model", "Transcription model is not configured")
			return
		}
		f, h, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "missing_file", "Audio file is required")
			return
		}
		defer f.Close()
		if h.Size > audio.MaxTranscriptionBytes {
			writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "Audio file too large")
			return
		}
		name := filepath.Base(h.Filename)
		ext := strings.ToLower(filepath.Ext(name))
		allowed := map[string]bool{".mp3": true, ".mp4": true, ".m4a": true, ".wav": true, ".webm": true, ".ogg": true, ".flac": true}
		if !allowed[ext] {
			writeError(w, http.StatusBadRequest, "invalid_file", "Unsupported audio file extension")
			return
		}
		data, err := io.ReadAll(io.LimitReader(f, audio.MaxTranscriptionBytes+1))
		if err != nil || len(data) == 0 || int64(len(data)) > audio.MaxTranscriptionBytes {
			writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "Invalid audio file size")
			return
		}
		result, err := route.Provider.Transcribe(r.Context(), route.UpstreamModel, name, data)
		if err != nil {
			writeError(w, http.StatusBadGateway, "transcription_upstream_unavailable", "Transcription provider unavailable")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(result)
	})
	mux.Handle("/", next)
	return mux
}
