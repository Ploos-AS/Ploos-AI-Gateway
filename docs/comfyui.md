# ComfyUI adapter (PR #12)

A ComfyUI provider uses a fixed private-IP `/prompt` endpoint and a server-owned API-format workflow JSON file. Configure `workflow_file` and `prompt_node` in the `image_providers` entry. The designated node must expose an `inputs.text` field. Only that field is replaced with the authenticated caller's prompt.

Example provider:

```json
{"id":"comfy","kind":"comfyui","endpoint":"http://127.0.0.1:8188/prompt","workflow_file":"/etc/ploos/comfy-workflow.json","prompt_node":"6"}
```

Register a corresponding `image_models` entry with `location: "selfhosted"` and `operations: ["images.generate"]`.

The adapter queues a workflow, polls history and retrieves one output image, returning OpenAI-style base64 JSON. Request cancellation stops polling. No remote fallback is configured. The ComfyUI API must be reachable on a trusted private network only.

**Limitations:** This is an initial synchronous adapter with a 120-second HTTP timeout per request; the overall generation duration follows the inbound request context. The workflow must already be valid for the installed ComfyUI model. No workflow upload, custom arbitrary graph or client-supplied URLs are allowed. Production deployment requires resource quotas, concurrency limits and deeper security review.
