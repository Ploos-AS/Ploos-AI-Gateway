package images

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"time"
)

// ProviderConfig declares an image backend independent of chat providers.
type ProviderConfig struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	Endpoint     string `json:"endpoint"`
	KeyEnv       string `json:"key_env,omitempty"`
	WorkflowFile string `json:"workflow_file,omitempty"`
	PromptNode   string `json:"prompt_node,omitempty"`
}

// ModelConfig binds a public image alias to one upstream model.
type ModelConfig struct {
	ID         string   `json:"id"`
	Provider   string   `json:"provider"`
	UpstreamID string   `json:"upstream_model"`
	Location   string   `json:"location"`
	Operations []string `json:"operations"`
}

// Route is a validated fixed destination.
type Route struct {
	UpstreamModel string
	Generator     ImageGenerator
}

// Registry never falls back to another provider.
type Registry struct {
	routes map[string]Route
}

// NewRegistry validates explicitly configured image routes.
func NewRegistry(providers []ProviderConfig, models []ModelConfig) (*Registry, error) {
	backends := make(map[string]ImageGenerator)
	kinds := make(map[string]string)
	for _, p := range providers {
		if p.ID == "" || backends[p.ID] != nil {
			return nil, errors.New("duplicate or empty image provider")
		}
		var g ImageGenerator
		var err error
		switch p.Kind {
		case "openai":
			if p.Endpoint != "https://api.openai.com/v1/images/generations" || p.KeyEnv != "OPENAI_API_KEY" {
				return nil, errors.New("invalid openai image provider configuration")
			}
			g, err = NewOpenAI(os.Getenv(p.KeyEnv))
		case "comfyui":
			if p.KeyEnv != "" {
				return nil, errors.New("comfyui does not support key_env")
			}
			if err = validateComfyEndpoint(p.Endpoint); err != nil {
				return nil, err
			}
			if p.WorkflowFile == "" || p.PromptNode == "" {
				return nil, errors.New("comfyui requires workflow_file and prompt_node")
			}
			workflow, readErr := os.ReadFile(p.WorkflowFile)
			if readErr != nil || len(workflow) > 1<<20 || !json.Valid(workflow) {
				return nil, errors.New("invalid comfyui workflow file")
			}
			transport := http.DefaultTransport.(*http.Transport).Clone()
			transport.Proxy = nil
			g = &ComfyGenerator{Endpoint: p.Endpoint, Workflow: workflow, PromptNode: p.PromptNode, Client: &http.Client{Timeout: 120 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
		default:
			return nil, fmt.Errorf("unsupported image provider kind %q", p.Kind)
		}
		if err != nil {
			return nil, err
		}
		backends[p.ID], kinds[p.ID] = g, p.Kind
	}
	reg := &Registry{routes: make(map[string]Route)}
	for _, m := range models {
		if m.ID == "" || m.UpstreamID == "" {
			return nil, errors.New("invalid image model")
		}
		if _, exists := reg.routes[m.ID]; exists {
			return nil, errors.New("duplicate image model")
		}
		g := backends[m.Provider]
		if g == nil {
			return nil, errors.New("unknown image provider")
		}
		expected := "remote"
		if kinds[m.Provider] == "comfyui" {
			expected = "selfhosted"
		}
		if m.Location != expected || len(m.Operations) != 1 || m.Operations[0] != "images.generate" {
			return nil, errors.New("image model must declare images.generate and correct location")
		}
		reg.routes[m.ID] = Route{UpstreamModel: m.UpstreamID, Generator: g}
	}
	return reg, nil
}

// Resolve looks up an explicitly configured image alias.
func (r *Registry) Resolve(id string) (Route, bool) {
	if r == nil {
		return Route{}, false
	}
	route, ok := r.routes[id]
	return route, ok
}

func validateComfyEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "/prompt" || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("invalid comfyui endpoint")
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil || port == "" {
		return errors.New("comfyui requires IP and port")
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || ip.Zone() != "" || ip.Is4In6() || (!ip.IsPrivate() && !ip.IsLoopback()) {
		return errors.New("comfyui requires private IP literal")
	}
	if _, err = net.LookupPort("tcp", port); err != nil {
		return errors.New("invalid comfyui port")
	}
	return nil
}
