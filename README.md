# Ploos AI Gateway

A lightweight, self-hostable, provider-neutral AI API gateway built in Go.

> **Status:** Active development. Authenticated chat routing and initial image generation are implemented. Not yet production-ready.

## Principles

- **Open source:** MIT-licensed, self-hostable, no mandatory Ploos services.
- **Provider-neutral:** adapters for direct APIs and model aggregators.
- **Privacy-conscious:** no prompt or response persistence by default.
- **Small footprint:** single Go binary, standard library first.
- **Safe defaults:** listen on loopback by default; model route requires an API key.
- **Portable:** separate core logic from infrastructure and managed hosting.

## Current capabilities (through PR #10)

- Authenticated OpenAI-compatible `POST /v1/chat/completions` with SSE streaming.
- Explicit JSON model aliases for OpenRouter, Ollama and private-IP self-hosted chat servers.
- Safe upstream error handling, timeouts, request limits and integration tests.
- Multimodal operation descriptors for chat, image, audio and video (metadata, not live endpoints).
- Authenticated `POST /v1/images/generations` using a configured OpenAI image model.
- Optional `image_providers` and `image_models` in the shared JSON registry.
- No implicit local-to-remote fallback.
- **ComfyUI is not yet operational:** its native workflow adapter remains unimplemented; configurations fail closed.

## Roadmap

| Phase | Status | Scope |
| --- | --- | --- |
| PRs #1–#3 | Implemented | Go service, CI, authentication, chat API and initial providers |
| PRs #4–#7 | Implemented | Validation, private chat routing and integration tests |
| PR #8 | Implemented | Multimodal capability contracts |
| PR #9 | Implemented | Initial OpenAI text-to-image API |
| PR #10 | Implemented | Shared image registry and fail-closed routing |
| Native ComfyUI | Next | Workflow templates, job polling and safe image retrieval |
| Image hardening | Planned | More providers, quotas and observability |
| Audio | Planned | Speech synthesis and transcription |
| Video | Planned | Async jobs; optional independent Ploos AI Video Connector integration |
| Operations | Planned | API key lifecycle, usage, budgets, rate limiting |
| Additional LLMs | Planned | Anthropic/Gemini adapters and explicit routing policies |

Local-only model requests must never silently fall back to remote providers.

See [architecture](docs/architecture.md), [image generation](docs/image-generation.md), [image routing](docs/image-routing.md), and [multimodal contracts](docs/multimodal-architecture.md).

## Quick start

Requires Go 1.23 or newer.

```sh
export PLOOS_GATEWAY_API_KEY="replace-with-a-long-random-secret"
go run ./cmd/gateway
curl --fail http://127.0.0.1:8080/healthz
```

To enable inference, configure `PLOOS_GATEWAY_REGISTRY` and appropriate upstream credentials. See [model routing](docs/model-routing.md) and [providers](docs/providers.md). Without a configured provider or registry, inference is unavailable.

Override the address with `PLOOS_GATEWAY_LISTEN` (for example, `127.0.0.1:9090`).

### Docker Compose

```sh
docker compose up --build -d
curl --fail http://127.0.0.1:8080/healthz
docker compose down
```

The development container is published **only on localhost**. Place a properly secured reverse proxy in front of a later production release; remember that Docker-published ports can bypass host firewall expectations.

## Development

```sh
go fmt ./...
go vet ./...
go test ./...
go build ./cmd/gateway
```

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).

Copyright (c) 2026 Ploos AS. Licensed under the [MIT License](LICENSE).
