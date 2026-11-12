// Package routing provides explicit, fail-closed model routing.
package routing

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/audio"
	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/images"
	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/providers"
)

// Model defines a public alias and its fixed upstream model.
type Model struct {
	ID         string   `json:"id"`
	Provider   string   `json:"provider"`
	UpstreamID string   `json:"upstream_model"`
	Location   string   `json:"location"`
	Modalities []string `json:"modalities"`
}

// ProviderConfig describes a configured provider endpoint.
type ProviderConfig struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Endpoint string `json:"endpoint"`
	KeyEnv   string `json:"key_env,omitempty"`
}

// Config contains a provider and model registry.
type Config struct {
	Providers      []ProviderConfig        `json:"providers"`
	Models         []Model                 `json:"models"`
	ImageProviders []images.ProviderConfig `json:"image_providers,omitempty"`
	ImageModels    []images.ModelConfig    `json:"image_models,omitempty"`
	AudioProviders []audio.ProviderConfig  `json:"audio_providers,omitempty"`
	AudioModels    []audio.ModelConfig     `json:"audio_models,omitempty"`
}

// Route is a resolved, fixed upstream destination.
type Route struct {
	Model    Model
	Provider providers.ChatProvider
}

// Registry maps public aliases to configured providers without fallback.
type Registry struct {
	routes map[string]Route
}

// Load reads a strict JSON registry.
func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()
	var cfg Config
	decoder := json.NewDecoder(io.LimitReader(f, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, err
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		return Config{}, errors.New("unexpected trailing data in registry")
	}
	return cfg, nil
}

// New validates all model/provider references before serving requests.
func New(cfg Config) (*Registry, error) {
	instances := make(map[string]providers.ChatProvider)
	locations := make(map[string]string)
	for _, p := range cfg.Providers {
		if p.ID == "" || instances[p.ID] != nil {
			return nil, fmt.Errorf("empty or duplicate provider id %q", p.ID)
		}
		if p.Kind != "openrouter" && p.Kind != "ollama" && p.Kind != "selfhosted" {
			return nil, fmt.Errorf("unsupported provider kind %q", p.Kind)
		}
		if p.Kind == "openrouter" && (p.Endpoint != "https://openrouter.ai/api/v1/chat/completions" || p.KeyEnv != "OPENROUTER_API_KEY") {
			return nil, fmt.Errorf("openrouter must use its fixed endpoint and OPENROUTER_API_KEY")
		}
		if (p.Kind == "ollama" || p.Kind == "selfhosted") && p.KeyEnv != "" {
			return nil, fmt.Errorf("ollama key_env is not supported")
		}
		if p.Kind == "ollama" && !strings.HasPrefix(p.Endpoint, "http://127.0.0.1:") && !strings.HasPrefix(p.Endpoint, "http://localhost:") {
			return nil, fmt.Errorf("ollama requires a loopback HTTP endpoint")
		}
		if p.Kind == "selfhosted" {
			if err := validatePrivateEndpoint(p.Endpoint); err != nil {
				return nil, fmt.Errorf("selfhosted provider %q: %w", p.ID, err)
			}
		}
		key := ""
		if p.KeyEnv != "" {
			key = os.Getenv(p.KeyEnv)
			if key == "" {
				return nil, fmt.Errorf("provider %q missing credential environment variable", p.ID)
			}
		}
		var instance *providers.HTTPChatProvider
		var err error
		if p.Kind == "selfhosted" {
			instance, err = providers.NewPrivateHTTPChatProvider(p.Endpoint)
		} else {
			instance, err = providers.NewHTTPChatProvider(p.Endpoint, key)
		}
		if err != nil {
			return nil, fmt.Errorf("provider %q: %w", p.ID, err)
		}
		instances[p.ID] = instance
		locations[p.ID] = p.Kind
	}
	reg := &Registry{routes: make(map[string]Route)}
	for _, m := range cfg.Models {
		if m.ID == "" || m.UpstreamID == "" || reg.routes[m.ID].Provider != nil {
			return nil, fmt.Errorf("invalid or duplicate model id %q", m.ID)
		}
		instance := instances[m.Provider]
		if instance == nil {
			return nil, fmt.Errorf("model %q references unknown provider", m.ID)
		}
		expected := "remote"
		if locations[m.Provider] == "ollama" {
			expected = "local"
		} else if locations[m.Provider] == "selfhosted" {
			expected = "selfhosted"
		}
		if m.Location != expected {
			return nil, fmt.Errorf("model %q location must be %q", m.ID, expected)
		}
		if len(m.Modalities) == 0 {
			return nil, fmt.Errorf("model %q must declare modalities", m.ID)
		}
		for _, modality := range m.Modalities {
			if modality != "text" {
				return nil, fmt.Errorf("unsupported modality %q for model %q", modality, m.ID)
			}
		}
		reg.routes[m.ID] = Route{Model: m, Provider: instance}
	}
	return reg, nil
}

// Resolve returns a route only for an explicitly configured alias.
func (r *Registry) Resolve(id string) (Route, bool) {
	if r == nil {
		return Route{}, false
	}
	route, ok := r.routes[id]
	return route, ok
}
