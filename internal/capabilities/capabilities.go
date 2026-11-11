// Package capabilities describes generative AI operations independently of providers.
package capabilities

import (
	"errors"
	"fmt"
)

// Operation identifies an API operation, not merely an input or output type.
type Operation string

const (
	Chat            Operation = "chat.completions"
	ImageGeneration Operation = "images.generate"
	Speech          Operation = "audio.speech"
	Transcription   Operation = "audio.transcriptions"
	VideoGeneration Operation = "videos.generate"
)

// Modality describes the media consumed or produced by an operation.
type Modality string

const (
	Text  Modality = "text"
	Image Modality = "image"
	Audio Modality = "audio"
	Video Modality = "video"
)

// Descriptor is a provider-independent operation contract.
type Descriptor struct {
	Operation Operation
	Input     []Modality
	Output    []Modality
	Streaming bool
	Async     bool
}

// Catalog is immutable to callers and does not imply a provider implements an operation.
var catalog = map[Operation]Descriptor{
	Chat:            {Operation: Chat, Input: []Modality{Text}, Output: []Modality{Text}, Streaming: true},
	ImageGeneration: {Operation: ImageGeneration, Input: []Modality{Text, Image}, Output: []Modality{Image}},
	Speech:          {Operation: Speech, Input: []Modality{Text}, Output: []Modality{Audio}, Streaming: true},
	Transcription:   {Operation: Transcription, Input: []Modality{Audio}, Output: []Modality{Text}},
	VideoGeneration: {Operation: VideoGeneration, Input: []Modality{Text, Image}, Output: []Modality{Video}, Async: true},
}

// Lookup returns a defensive copy of the operation descriptor.
func Lookup(op Operation) (Descriptor, bool) {
	d, ok := catalog[op]
	if !ok {
		return Descriptor{}, false
	}
	d.Input = append([]Modality(nil), d.Input...)
	d.Output = append([]Modality(nil), d.Output...)
	return d, true
}

// Model declares operations a particular upstream model explicitly supports.
type Model struct {
	ID         string
	Operations []Operation
}

// Validate rejects unknown operations and duplicate declarations.
func (m Model) Validate() error {
	if m.ID == "" {
		return errors.New("model ID is required")
	}
	if len(m.Operations) == 0 {
		return errors.New("at least one operation is required")
	}
	seen := make(map[Operation]bool)
	for _, op := range m.Operations {
		if _, ok := Lookup(op); !ok {
			return fmt.Errorf("unknown operation %q", op)
		}
		if seen[op] {
			return fmt.Errorf("duplicate operation %q", op)
		}
		seen[op] = true
	}
	return nil
}

// Supports is fail-closed: undeclared operations cannot be routed.
func (m Model) Supports(op Operation) bool {
	if m.Validate() != nil {
		return false
	}
	for _, supported := range m.Operations {
		if supported == op {
			return true
		}
	}
	return false
}
