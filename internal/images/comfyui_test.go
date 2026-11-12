package images

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestComfyUIWorkflow(t *testing.T) {
	seen := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/prompt":
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), "a small fox") {
				t.Error("prompt not injected")
			}
			seen = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"prompt_id":"job-1"}`)
		case r.URL.Path == "/history/job-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"job-1":{"outputs":{"save":{"images":[{"filename":"out.png","type":"output"}]}}}}`)
		case r.URL.Path == "/view":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10, 0})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	workflow := json.RawMessage(`{"text":{"class_type":"CLIPTextEncode","inputs":{"text":"placeholder"}}}`)
	g := &ComfyGenerator{Endpoint: server.URL + "/prompt", Client: &http.Client{Timeout: 5 * time.Second}, Workflow: workflow, PromptNode: "text"}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	status, response, err := g.Generate(ctx, []byte(`{"prompt":"a small fox"}`))
	if err != nil || status != http.StatusOK || !seen || !strings.Contains(string(response), "b64_json") {
		t.Fatalf("status=%d err=%v response=%s", status, err, response)
	}
}
