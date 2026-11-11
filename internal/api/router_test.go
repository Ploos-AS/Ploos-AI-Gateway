package api

import (
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 "github.com/Ploos-AS/Ploos-AI-Gateway/internal/routing"
)

func TestRoutedHandlerRejectsInvalidMessages(t *testing.T) {
 cfg := routing.Config{
  Providers: []routing.ProviderConfig{{ID: "local", Kind: "ollama", Endpoint: "http://127.0.0.1:11434/v1/chat/completions"}},
  Models: []routing.Model{{ID: "local/test", Provider: "local", UpstreamID: "test", Location: "local", Modalities: []string{"text"}}},
 }
 registry, err := routing.New(cfg)
 if err != nil { t.Fatal(err) }
 handler := NewRoutedHandler("client-secret", registry)
 for _, body := range []string{
  `{"model":"local/test","messages":[]}`,
  `{"model":"local/test","messages":[null]}`,
  `{"model":"local/test","messages":[{"content":"hello"}]}`,
  `{"model":"local/test","messages":"bad"}`,
 } {
  req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
  req.Header.Set("Authorization", "Bearer client-secret")
  res := httptest.NewRecorder()
  handler.ServeHTTP(res, req)
  if res.Code != http.StatusBadRequest { t.Errorf("body %s: status %d", body, res.Code) }
 }
}
