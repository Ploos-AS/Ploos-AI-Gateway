package video

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestConnectorCapabilities(t *testing.T) {
	const token = "0123456789abcdef0123456789abcdef"
	t.Setenv("PLOOS_VIDEO_CONNECTOR_TOKEN", token)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" || r.Header.Get("Authorization") != "Bearer "+token {
			t.Errorf("unexpected connector request")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"Diagnostic only"}]}}`))
	}))
	defer server.Close()
	c, err := New(Config{Endpoint: server.URL + "/mcp", TokenEnv: "PLOOS_VIDEO_CONNECTOR_TOKEN"})
	if err != nil { t.Fatal(err) }
	result, err := c.Capabilities(context.Background())
	if err != nil || !strings.Contains(string(result), "Diagnostic only") {
		t.Fatalf("unexpected capabilities %s: %v", result, err)
	}
}

func TestConnectorRejectsPublicAndMissingToken(t *testing.T) {
	t.Setenv("PLOOS_VIDEO_CONNECTOR_TOKEN", "0123456789abcdef0123456789abcdef")
	for _, endpoint := range []string{
		"https://example.com/mcp",
		"http://8.8.8.8:8787/mcp",
		"http://127.0.0.1:8787/other",
		"http://127.0.0.1:8787/mcp?redirect=1",
	} {
		if _, err := New(Config{Endpoint: endpoint, TokenEnv: "PLOOS_VIDEO_CONNECTOR_TOKEN"}); err == nil {
			t.Fatalf("accepted unsafe endpoint %s", endpoint)
		}
	}
	_ = os.Unsetenv("PLOOS_VIDEO_CONNECTOR_TOKEN")
	if _, err := New(Config{Endpoint: "http://127.0.0.1:8787/mcp", TokenEnv: "PLOOS_VIDEO_CONNECTOR_TOKEN"}); err == nil {
		t.Fatal("accepted missing token")
	}
}
