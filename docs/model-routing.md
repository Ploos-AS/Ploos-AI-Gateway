# Model registry and routing (PR #4)

Set `PLOOS_GATEWAY_REGISTRY=config/models.example.json` to enable model-alias routing. Set `PLOOS_GATEWAY_API_KEY` and, if OpenRouter is configured, `OPENROUTER_API_KEY`. The gateway fails startup if configuration is invalid or a required credential is missing.

The example uses **JSON** rather than YAML in this first iteration, to keep the gateway dependency-free and the configuration strict. YAML support can be added as a separate parser without changing the registry or routing logic.

Requests use public aliases such as `local/llama` and `openrouter/example`. The gateway replaces the model field with the configured upstream model name and forwards the request to the exact configured provider. Unknown models return HTTP 400. No fallback exists.

**Local-only invariant:** A model declared `location: local` must use an Ollama provider configured with a loopback endpoint. A remote provider cannot be labeled local. The router never retries a local request on a remote provider.

Current scope is chat and text only. Multimodal support, arbitrary self-hosted endpoints over WireGuard, admin management, SQLite and fallback policies are not implemented. For now, Ollama must share the gateway's network namespace. Do not expose this development gateway publicly; no quotas, production error mapping or comprehensive request validation exist yet.
