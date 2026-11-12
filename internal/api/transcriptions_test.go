package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/audio"
)

func TestTranscriptionRoutingAndAuth(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.FormValue("model") != "whisper-local" {
			t.Errorf("unexpected upstream model: %q", r.FormValue("model"))
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Error(err)
			return
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		if string(data) != "ID3test" {
			t.Errorf("unexpected audio: %q", data)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"hello world"}`))
	}))
	defer upstream.Close()
	reg, err := audio.NewTranscriptionRegistry([]audio.ProviderConfig{{ID: "local", Kind: "selfhosted", Endpoint: "http://127.0.0.1:8081/v1/audio/transcriptions"}}, []audio.ModelConfig{{ID: "transcribe/local", Provider: "local", UpstreamID: "whisper-local", Location: "selfhosted", Operations: []string{"audio.transcriptions"}}})
	if err != nil {
		t.Fatal(err)
	}
	route, _ := reg.Resolve("transcribe/local")
	route.Provider.Endpoint = upstream.URL + "/v1/audio/transcriptions"
	handler := WithTranscriptions(http.NotFoundHandler(), "secret", reg)
	post := func(key, model, filename string) *httptest.ResponseRecorder {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("model", model)
		part, _ := writer.CreateFormFile("file", filename)
		_, _ = part.Write([]byte("ID3test"))
		_ = writer.Close()
		req := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	if got := post("", "transcribe/local", "test.mp3").Code; got != http.StatusUnauthorized {
		t.Fatalf("auth status %d", got)
	}
	if got := post("secret", "missing", "test.mp3").Code; got != http.StatusBadRequest {
		t.Fatalf("unknown model status %d", got)
	}
	if got := post("secret", "transcribe/local", "test.exe").Code; got != http.StatusBadRequest {
		t.Fatalf("invalid extension status %d", got)
	}
	rec := post("secret", "transcribe/local", "test.mp3")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil || result.Text != "hello world" || calls != 1 {
		t.Fatalf("unexpected response: %s calls=%d", rec.Body.String(), calls)
	}
}
