# Architecture (initial)

Ploos AI Gateway is a standalone, vendor-neutral Go service. It must not require any other Ploos product or cloud platform.

## M0

- `cmd/gateway`: process lifecycle, configurable bind address, graceful shutdown.
- `internal/api`: versionable HTTP surface and health check.
- Container defaults: non-root, read-only filesystem, loopback-only host publishing.

## Planned M1

- Auth middleware that verifies caller keys before any provider request.
- OpenAI-compatible `POST /v1/chat/completions`, including SSE streaming.
- Registry of configured model aliases and provider adapters.
- OpenAI and OpenRouter adapters; credentials loaded from environment or a secret store, never returned in API responses.
- Explicit per-request timeout and bounded request/response sizes, with streaming-aware limits.
- SSRF-safe provider endpoint configuration and guarded redirects; no arbitrary user-supplied upstream URLs.
- Reject unsupported models and do not silently downgrade to another model.
- Redact bearer tokens and user-provided content from logs.
- Unit and local fake-upstream integration tests (no real provider keys in CI).

## Later

SQLite usage and metadata, key lifecycle, per-key permissions, budgets, fallback policies, more providers, and optional deployment via Caddy/WireGuard. Provider availability, cost ceilings and regional data-handling claims must be tested against actual upstream behavior.

## Explicit non-goals for M0

No provider calls, no model billing, no multi-tenant control plane and no public production endpoint.
