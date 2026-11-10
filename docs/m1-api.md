# M1 API contract (part 1)

This pull request adds a protected endpoint placeholder, **not model inference**.

## Configuration

Set `PLOOS_GATEWAY_API_KEY` to a high-entropy secret in the process environment. Never commit it to source control. Without this value, model requests return HTTP 503.

`GET /healthz` remains public for local health checks.

`POST /v1/chat/completions` requires `Authorization: Bearer <key>`. Missing or invalid credentials return HTTP 401 with an OpenAI-style JSON error. Authenticated requests currently return HTTP 501 (`provider_unavailable`), since no provider has been wired up.

Do not expose the gateway publicly yet. The single shared key is a bootstrap mechanism, not a multi-tenant key-management system. Use TLS and network restrictions before enabling remote access.

## Next steps

Add typed chat request/response handling, model alias validation, upstream provider adapters, SSE streaming, request limits and timeout policies, with mocked upstream integration tests. Provider credentials must remain separate from caller API keys.
