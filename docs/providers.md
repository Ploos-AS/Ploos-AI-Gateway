# Chat provider adapters (M1)

Ploos AI Gateway supports a single selected upstream chat provider per running instance at this stage. This is a first step toward a registry of independently configurable providers and modalities.

## OpenRouter

```sh
export PLOOS_GATEWAY_API_KEY="your-long-random-client-key"
export PLOOS_GATEWAY_PROVIDER=openrouter
export OPENROUTER_API_KEY="your-openrouter-provider-key"
go run ./cmd/gateway
```

Send an OpenAI-compatible chat completion to `http://127.0.0.1:8080/v1/chat/completions` using the **gateway client key**. The gateway sends the request to OpenRouter with the separate provider key.

## Ollama

Start Ollama on the **same network namespace** as the gateway, and ensure an appropriate model is installed:

```sh
export PLOOS_GATEWAY_API_KEY="your-long-random-client-key"
export PLOOS_GATEWAY_PROVIDER=ollama
go run ./cmd/gateway
```

The adapter targets `http://127.0.0.1:11434/v1/chat/completions`. This is suitable for a native binary running alongside Ollama, **not** for the default Compose container, whose localhost is isolated. Docker deployment with Ollama requires an explicitly designed private network configuration in a later PR.

## API

```sh
curl -sS http://127.0.0.1:8080/v1/chat/completions \
  -H "Authorization: Bearer $PLOOS_GATEWAY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"your-model-id","messages":[{"role":"user","content":"Hello"}]}'
```

Streaming is supported when the upstream emits `text/event-stream` (request with `"stream":true`). Provider responses are forwarded without conversation storage.

## Security and limitations

- **Development only.** No per-client quotas, model allowlists, usage metering, circuit breakers, or cost limits.
- Only a fixed OpenRouter HTTPS URL or loopback Ollama URL can be selected by environment; clients cannot supply arbitrary upstream destinations.
- A gateway client API key is mandatory. Keep both provider and gateway keys out of logs and Git.
- Requests are limited to 1 MiB. Upstream calls time out after 120 seconds; long streaming sessions may be cut off.
- Upstream network failures currently cannot always be represented as structured API errors. This will be improved before production.
- Ollama may not implement every OpenAI feature. Check compatibility for individual models and endpoints.
- Hosting the gateway in Europe does not guarantee EU-only processing when using remote providers.
