package routing

import "testing"

func TestPrivateEndpointValidation(t *testing.T) {
	tests := []struct {
		endpoint string
		allowed  bool
	}{
		{"http://10.77.0.2:8000/v1/chat/completions", true},
		{"http://127.0.0.1:8000/v1/chat/completions", true},
		{"https://[fd00::2]:8443/v1/chat/completions", true},
		{"http://169.254.169.254:80/v1/chat/completions", false},
		{"http://8.8.8.8:8000/v1/chat/completions", false},
		{"http://example.com:8000/v1/chat/completions", false},
		{"http://10.77.0.2:8000/admin", false},
		{"http://10.77.0.2:8000/v1/chat/completions?next=x", false},
		{"http://user:pass@10.77.0.2:8000/v1/chat/completions", false},
	}
	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			err := validatePrivateEndpoint(tt.endpoint)
			if (err == nil) != tt.allowed {
				t.Fatalf("validation = %v, want allowed %v", err, tt.allowed)
			}
		})
	}
}

func TestSelfhostedRegistryAndLocality(t *testing.T) {
	cfg := Config{
		Providers: []ProviderConfig{{ID: "private", Kind: "selfhosted", Endpoint: "http://10.77.0.2:8000/v1/chat/completions"}},
		Models:    []Model{{ID: "private/qwen", Provider: "private", UpstreamID: "qwen", Location: "selfhosted", Modalities: []string{"text"}}},
	}
	reg, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Resolve("private/qwen"); !ok {
		t.Fatal("missing selfhosted model")
	}
	cfg.Models[0].Location = "local"
	if _, err := New(cfg); err == nil {
		t.Fatal("selfhosted model must not be marked local")
	}
	cfg.Models[0].Location = "selfhosted"
	cfg.Providers[0].Endpoint = "http://169.254.169.254:80/v1/chat/completions"
	if _, err := New(cfg); err == nil {
		t.Fatal("metadata endpoint must be rejected")
	}
}
