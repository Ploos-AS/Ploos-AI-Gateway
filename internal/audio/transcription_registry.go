package audio

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type TranscriptionRoute struct {
	UpstreamModel string
	Provider *TranscriptionProvider
}

type TranscriptionRegistry struct {
	routes map[string]TranscriptionRoute
}

func NewTranscriptionRegistry(providers []ProviderConfig, models []ModelConfig) (*TranscriptionRegistry, error) {
	backends := make(map[string]*TranscriptionProvider)
	locations := make(map[string]string)
	for _, p := range providers {
		if p.ID == "" || backends[p.ID] != nil {
			return nil, errors.New("duplicate or empty transcription provider")
		}
		key := ""
		location := ""
		switch p.Kind {
		case "openai":
			if p.Endpoint != "https://api.openai.com/v1/audio/transcriptions" || p.KeyEnv != "OPENAI_API_KEY" {
				return nil, errors.New("invalid openai transcription provider")
			}
			key = os.Getenv(p.KeyEnv)
			if key == "" {
				return nil, errors.New("missing transcription provider key")
			}
			location = "remote"
		case "selfhosted":
			if p.KeyEnv != "" {
				return nil, errors.New("selfhosted transcription key_env unsupported")
			}
			if err := validatePrivateTranscriptionEndpoint(p.Endpoint); err != nil {
				return nil, err
			}
			location = "selfhosted"
		default:
			return nil, fmt.Errorf("unsupported transcription provider %q", p.Kind)
		}
		backends[p.ID] = &TranscriptionProvider{Endpoint: p.Endpoint, Key: key, Client: NewClient(location == "selfhosted")}
		locations[p.ID] = location
	}
	reg := &TranscriptionRegistry{routes: make(map[string]TranscriptionRoute)}
	for _, m := range models {
		if m.ID == "" || m.UpstreamID == "" {
			return nil, errors.New("invalid transcription model")
		}
		if _, exists := reg.routes[m.ID]; exists {
			return nil, errors.New("duplicate transcription model")
		}
		p := backends[m.Provider]
		if p == nil || m.Location != locations[m.Provider] || len(m.Operations) != 1 || m.Operations[0] != "audio.transcriptions" {
			return nil, errors.New("invalid transcription route or capability")
		}
		reg.routes[m.ID] = TranscriptionRoute{UpstreamModel: m.UpstreamID, Provider: p}
	}
	return reg, nil
}

func (r *TranscriptionRegistry) Resolve(alias string) (TranscriptionRoute, bool) {
	if r == nil {
		return TranscriptionRoute{}, false
	}
	route, ok := r.routes[alias]
	return route, ok
}

func validatePrivateTranscriptionEndpoint(endpoint string) error {
	if !strings.HasSuffix(endpoint, "/v1/audio/transcriptions") {
		return errors.New("invalid transcription endpoint path")
	}
	return validatePrivateSpeechEndpoint(strings.TrimSuffix(endpoint, "/v1/audio/transcriptions")+"/v1/audio/speech")
}
