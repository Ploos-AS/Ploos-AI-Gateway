package routing

import "testing"

func TestRegistryRejectsRemoteMasqueradingAsLocal(t *testing.T) {
	cfg := Config{
		Providers: []ProviderConfig{{ID: "remote", Kind: "openrouter", Endpoint: "https://openrouter.ai/api/v1/chat/completions", KeyEnv: "OPENROUTER_API_KEY"}},
		Models: []Model{{ID: "local/private", Provider: "remote", UpstreamID: "example", Location: "local", Modalities: []string{"text"}}},
	}
	t.Setenv("OPENROUTER_API_KEY", "fake-test-key")
	if _, err := New(cfg); err == nil {
		t.Fatal("expected local model on remote provider to be rejected")
	}
}

func TestRegistryLocalAndUnknownModel(t *testing.T) {
	cfg := Config{
		Providers: []ProviderConfig{{ID: "ollama", Kind: "ollama", Endpoint: "http://127.0.0.1:11434/v1/chat/completions"}},
		Models: []Model{{ID: "local/llama", Provider: "ollama", UpstreamID: "llama3.2", Location: "local", Modalities: []string{"text"}}},
	}
	reg, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	route, ok := reg.Resolve("local/llama")
	if !ok || route.Model.UpstreamID != "llama3.2" {
		t.Fatal("local model not resolved")
	}
	if _, ok := reg.Resolve("remote/unknown"); ok {
		t.Fatal("unknown model must not resolve")
	}
}

func TestRegistryRejectsRemoteHTTP(t *testing.T) {
	cfg := Config{Providers: []ProviderConfig{{ID: "ollama", Kind: "ollama", Endpoint: "http://example.com/v1/chat/completions"}}}
	if _, err := New(cfg); err == nil {
		t.Fatal("expected non-loopback Ollama endpoint rejection")
	}
}
