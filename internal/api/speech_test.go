package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/audio"
)

func TestSpeechAuthenticationAndRouting(t *testing.T) {
	var upstreamCalls int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		var request struct {
			Model string `json:"model"`
			Input string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request.Model != "tts-local" || request.Input != "hello" {
			t.Errorf("unexpected upstream request: %+v", request)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("ID3fake-mp3"))
	}))
	defer upstream.Close()
	registry, err := audio.NewRegistry(
		[]audio.ProviderConfig{{ID: "local", Kind: "selfhosted", Endpoint: "http://127.0.0.1:8081/v1/audio/speech"}},
		[]audio.ModelConfig{{ID: "speech/local", Provider: "local", UpstreamID: "tts-local", Location: "selfhosted", Operations: []string{"audio.speech"}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	route, ok := registry.Resolve("speech/local")
	if !ok {
		t.Fatal("missing speech route")
	}
	route.Provider.Endpoint = upstream.URL + "/v1/audio/speech"
	handler := WithSpeech(http.NotFoundHandler(), "secret", registry)
	post := func(key, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/audio/speech", strings.NewReader(body))
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}
	if w := post("", `{"model":"speech/local","input":"hello","voice":"alloy"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated request returned %d", w.Code)
	}
	if w := post("secret", `{"model":"speech/unknown","input":"hello","voice":"alloy"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("unknown model returned %d", w.Code)
	}
	w := post("secret", `{"model":"speech/local","input":"hello","voice":"alloy"}`)
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("speech response: %d %s", w.Code, w.Body.String())
	}
	data, _ := io.ReadAll(w.Body)
	if string(data) != "ID3fake-mp3" || upstreamCalls != 1 {
		t.Fatalf("unexpected response or upstream calls: %q %d", data, upstreamCalls)
	}
}
