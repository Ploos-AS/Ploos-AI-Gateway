# Image routing in the shared registry (PR #10)

The JSON registry accepts optional `image_providers` and `image_models` alongside the existing chat configuration.

Example:

```json
{
  "providers": [],
  "models": [],
  "image_providers": [
    {"id":"openai-images","kind":"openai","endpoint":"https://api.openai.com/v1/images/generations","key_env":"OPENAI_API_KEY"}
  ],
  "image_models": [
    {"id":"images/default","provider":"openai-images","upstream_model":"gpt-image-1","location":"remote","operations":["images.generate"]}
  ]
}
```

Only explicitly declared `images.generate` models resolve at the image endpoint. Chat and image routes are separate; no automatic fallback is performed. Legacy image environment configuration cannot be combined with registry mode.

**ComfyUI limitation:** private-IP endpoint validation is included, but ComfyUI's native `/prompt` workflow and asynchronous image retrieval require a separate adapter. ComfyUI configurations intentionally fail at startup rather than forwarding an incompatible OpenAI request. This PR does not yet deliver functional self-hosted image generation.
