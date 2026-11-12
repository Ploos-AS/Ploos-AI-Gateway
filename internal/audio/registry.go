package audio

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
)

// ProviderConfig declares a fixed speech provider.
type ProviderConfig struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Endpoint string `json:"endpoint"`
	KeyEnv   string `json:"key_env,omitempty"`
}

// ModelConfig maps a public speech alias to one provider.
type ModelConfig struct {
	ID         string   `json:"id"`
	Provider   string   `json:"provider"`
	UpstreamID string   `json:"upstream_model"`
	Location   string   `json:"location"`
	Operations []string `json:"operations"`
}

// Route is an explicit speech model destination.
type Route struct {
	UpstreamModel string
	Provider      *SpeechProvider
}

// Registry resolves only declared speech models.
type Registry struct {
	routes map[string]Route
}

// NewRegistry validates providers and model capabilities before serving.
func NewRegistry(providers []ProviderConfig, models []ModelConfig) (*Registry, error) {
	backends := make(map[string]*SpeechProvider)
	locations := make(map[string]string)
	for _, p := range providers {
		if p.ID == "" || backends[p.ID] != nil {
			return nil, errors.New("duplicate or empty speech provider")
		}
		key := ""
		location := ""
		switch p.Kind {
		case "openai":
			if p.Endpoint != "https://api.openai.com/v1/audio/speech" || p.KeyEnv != "OPENAI_API_KEY" {
				return nil, errors.New("invalid openai speech provider")
			}
			key = os.Getenv(p.KeyEnv)
			if key == "" {
				return nil, errors.New("missing speech provider key")
			}
			location = "remote"
		case "selfhosted":
			if p.KeyEnv != "" {
				return nil, errors.New("selfhosted speech key_env is unsupported")
			}
			if err := validatePrivateSpeechEndpoint(p.Endpoint); err != nil {
				return nil, err
			}
			location = "selfhosted"
		default:
			return nil, fmt.Errorf("unsupported speech provider %q", p.Kind)
		}
		backends[p.ID] = &SpeechProvider{Endpoint: p.Endpoint, Key: key, Client: NewClient(location == "selfhosted")}
		locations[p.ID] = location
	}
	reg := &Registry{routes: make(map[string]Route)}
	for _, m := range models {
		if m.ID == "" || m.UpstreamID == "" {
			return nil, errors.New("invalid speech model")
		}
		if _, exists := reg.routes[m.ID]; exists {
			return nil, errors.New("duplicate speech model")
		}
		p := backends[m.Provider]
		if p == nil || m.Location != locations[m.Provider] || len(m.Operations) != 1 || m.Operations[0] != "audio.speech" {
			return nil, errors.New("invalid speech route or capability")
		}
		reg.routes[m.ID] = Route{UpstreamModel: m.UpstreamID, Provider: p}
	}
	return reg, nil
}

// Resolve returns a fixed speech route, never a fallback.
func (r *Registry) Resolve(alias string) (Route, bool) {
	if r == nil {
		return Route{}, false
	}
	route, ok := r.routes[alias]
	return route, ok
}

func validatePrivateSpeechEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "/v1/audio/speech" || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("invalid selfhosted speech endpoint")
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil || port == "" {
		return errors.New("selfhosted speech endpoint requires IP and port")
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || ip.Zone() != "" || ip.Is4In6() || (!ip.IsPrivate() && !ip.IsLoopback()) {
		return errors.New("selfhosted speech endpoint requires private IP")
	}
	if _, err := net.LookupPort("tcp", port); err != nil {
		return errors.New("invalid speech endpoint port")
	}
	return nil
}
