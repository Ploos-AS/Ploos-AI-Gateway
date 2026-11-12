package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/images"
)

func TestImageRouteAndAuthentication(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer upstream-key" {
			t.Error("wrong upstream auth")
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"model":"actual-model"`) {
			t.Errorf("wrong model: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[{"url":"https://example.invalid/image.png"}]}`)
	}))
	defer upstream.Close()
	gen := &images.Generator{Endpoint: upstream.URL, APIKey: "upstream-key", Client: &http.Client{Timeout: time.Second}}
	h := WithImageGeneration(http.NotFoundHandler(), "client-key", ImageRoute{Alias: "image/demo", UpstreamModel: "actual-model", Generator: gen})
	for _, tc := range []struct {
		key    string
		body   string
		status int
	}{
		{"wrong", `{"model":"image/demo","prompt":"hello"}`, 401},
		{"client-key", `{"model":"image/unknown","prompt":"hello"}`, 400},
		{"client-key", `{"model":"image/demo","prompt":""}`, 400},
		{"client-key", `{"model":"image/demo","prompt":"hello"}`, 200},
	} {
		req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(tc.body))
		req.Header.Set("Authorization", "Bearer "+tc.key)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != tc.status {
			t.Errorf("status=%d want=%d body=%s", rec.Code, tc.status, rec.Body.String())
		}
	}
	if calls != 1 {
		t.Fatalf("upstream calls=%d", calls)
	}
}
