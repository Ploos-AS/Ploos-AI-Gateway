# Private self-hosted model servers (PR #6)

Configure a provider with `kind: selfhosted` in the JSON registry:

```json
{"id":"wg-vllm","kind":"selfhosted","endpoint":"http://10.77.0.2:8000/v1/chat/completions"}
```

Model entry:

```json
{"id":"private/qwen","provider":"wg-vllm","upstream_model":"qwen","location":"selfhosted","modalities":["text"]}
```

The server must offer an OpenAI-compatible chat completions endpoint (for example vLLM, llama.cpp server or LM Studio). Use a literal private IP address with explicit port. DNS hostnames, public IPs, link-local metadata addresses, credentials in URLs, redirects and arbitrary paths are rejected. The selfhosted HTTP client ignores environment HTTP proxies.

**Locality:** `local` is reserved for loopback Ollama. `selfhosted` means a private-network model server; it does not mean physically on the gateway host. Neither route has automatic fallback to remote providers.

**Limitations:** A private IP alone does not establish trust or confidentiality. Protect the model server with WireGuard, firewall policies and HTTPS where appropriate. This is not an unrestricted SSRF defense against a privileged operator configuring an internal private address. No automatic service discovery, custom TLS CA, authentication to selfhosted upstreams, or multimodal APIs yet. Do not deploy publicly.
