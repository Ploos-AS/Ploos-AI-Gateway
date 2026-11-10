# Ploos AI Gateway

A lightweight, self-hostable, provider-neutral AI API gateway built in Go.

> **Status:** Early development (M0 foundation). No model-provider integration yet. Not production-ready.

## Principles

- **Open source:** MIT-licensed, self-hostable, no mandatory Ploos services.
- **Provider-neutral:** adapters for direct APIs and model aggregators.
- **Privacy-conscious:** no prompt or response persistence by default.
- **Small footprint:** single Go binary, standard library first.
- **Safe defaults:** listen on loopback by default; API access will be protected before provider routing is enabled.
- **Portable:** separate core logic from infrastructure and managed hosting.

## Roadmap

| Milestone | Goal |
|---|---|
| M0 | Go project, health endpoint, tests, CI, Docker, documentation |
| M1 | Authenticated OpenAI-compatible chat completions (including SSE streaming), OpenAI and OpenRouter adapters |
| M2 | SQLite-backed model registry, API-key lifecycle, usage accounting |
| M3 | Anthropic and Gemini adapters, policy routing, fallback and budget controls |

No actual provider calls are implemented in M0. Do not expose this development service directly to the internet.

## Quick start

Requires Go 1.23 or newer.

```sh
go run ./cmd/gateway
curl --fail http://127.0.0.1:8080/healthz
```

Override the address with `PLOOS_GATEWAY_LISTEN` (for example, `127.0.0.1:9090`).

### Docker Compose

```sh
docker compose up --build -d
curl --fail http://127.0.0.1:8080/healthz
docker compose down
```

The development container is published **only on localhost**. Place a properly secured reverse proxy in front of a later production release; remember that Docker-published ports can bypass host firewall expectations.

## API (M0)

- `GET /healthz`: returns `{"status":"ok"}`.
- Other routes return `404`.

## Security and privacy

M0 has no authentication or model-proxy endpoints. Never add provider credentials to Git, request logs, URL query parameters, or container images. No request body is logged by the server.

## Development

```sh
go fmt ./...
go vet ./...
go test ./...
go build ./cmd/gateway
```

Contributions welcome once the initial API contract is published. See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).

Copyright (c) 2026 Ploos AS. Licensed under the [MIT License](LICENSE).
