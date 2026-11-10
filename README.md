# Ploos AI Gateway

A lightweight, self-hostable, provider-neutral AI API gateway built in Go.

> **Status:** Early development (M1 API/auth scaffold). No model-provider integration yet. Not production-ready.

## Principles

- **Open source:** MIT-licensed, self-hostable, no mandatory Ploos services.
- **Provider-neutral:** adapters for direct APIs and model aggregators.
- **Privacy-conscious:** no prompt or response persistence by default.
- **Small footprint:** single Go binary, standard library first.
- **Safe defaults:** listen on loopback by default; model route requires an API key.
- **Portable:** separate core logic from infrastructure and managed hosting.

## Roadmap

| Milestone | Goal |
|---|---|
| M0 | Go project, health endpoint, tests, CI, Docker, documentation |
| M1 | Authenticated OpenAI-compatible chat completions (including SSE streaming), OpenAI and OpenRouter adapters |
| M2 | SQLite-backed model registry, API-key lifecycle, usage accounting |
| M3 | Anthropic and Gemini adapters, policy routing, fallback and budget controls |

## Quick start

Requires Go 1.23 or newer.

```sh
export PLOOS_GATEWAY_API_KEY="replace-with-a-long-random-secret"
go run ./cmd/gateway
curl --fail http://127.0.0.1:8080/healthz
```

The chat route is currently a protected placeholder returning 501 after successful authentication; no provider calls are implemented yet. See [M1 API notes](docs/m1-api.md).

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
