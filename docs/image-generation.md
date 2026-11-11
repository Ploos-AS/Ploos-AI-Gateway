# Image generation (PR #9)

Initial text-to-image support adds `POST /v1/images/generations` with a fixed OpenAI upstream. Set `PLOOS_IMAGE_MODEL_ALIAS` (for example `images/default`), `PLOOS_IMAGE_UPSTREAM_MODEL` (an image model enabled on your account) and `OPENAI_API_KEY`. Clients authenticate with `PLOOS_GATEWAY_API_KEY`.

Example: `{"model":"images/default","prompt":"A red kite"}`.

Only model, prompt and optional n (maximum 1) are accepted; only explicitly whitelisted parameters are forwarded. Upstream errors return generic 502 responses. Prompts and generated images are not persisted by the gateway.

This is intentionally a narrow first implementation: OpenAI only; no local image backend, image editing, cost controls, integrated JSON chat model registry or provider selection. Responses larger than 12 MiB are rejected. The architecture should be expanded to capability-aware routing and local image generation in later PRs before production use.
