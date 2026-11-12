# Architecture — current state

Ploos AI Gateway is a standalone, MIT-licensed Go service with no mandatory Ploos Cloud dependency.

## Implemented through PR #10

- `cmd/gateway`: process lifecycle, configurable loopback bind and graceful shutdown.
- `internal/api`: authenticated OpenAI-compatible chat and image generation endpoints, plus health checks.
- `internal/providers`: fixed-destination chat adapters for OpenRouter, Ollama and self-hosted OpenAI-compatible model servers.
- `internal/routing`: strict JSON model registry, explicit location and model aliases; no implicit fallback.
- `internal/capabilities`: provider-independent multimodal operation descriptors.
- `internal/images`: fixed OpenAI image generation and image registry; ComfyUI is validation-only, **not functional**.
- Local mock-upstream integration tests for chat, SSE, cancellation and initial image generation.

## Architectural constraints

- Authentication before forwarding; provider credentials stay server-side.
- No arbitrary client-selected upstream URL.
- No implicit local/private-to-remote fallback.
- Bounded requests and responses; generic upstream errors.
- Explicit operation capability declarations for image routes.
- Provider independence: no cloud provider required for chat deployments using local models. Image generation currently requires OpenAI.

## Next milestones

Native ComfyUI workflow adapter, image API hardening, audio operations, asynchronous video jobs, usage accounting, quotas, additional provider adapters and observability.

**Not production-ready:** security review, operational controls and broader integration testing remain outstanding.
