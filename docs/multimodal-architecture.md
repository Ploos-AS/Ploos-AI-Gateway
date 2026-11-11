# Multimodal architecture (PR #8)

This PR defines **capability contracts**, not new live API endpoints. The existing chat gateway continues to serve text chat only.

## Operations

| Operation | Inputs | Output | Lifecycle |
| --- | --- | --- | --- |
| `chat.completions` | text | text | optional streaming |
| `images.generate` | text, optional image | image | synchronous |
| `audio.speech` | text | audio | optional streaming |
| `audio.transcriptions` | audio | text | synchronous |
| `videos.generate` | text, optional image | video | asynchronous job |

An operation is not synonymous with a modality: speech and transcription both involve audio but require different request and response contracts.

The `internal/capabilities` package defines operations, media descriptors, and explicit per-model capability declarations. Unknown or undeclared operations fail closed. Catalog entries are metadata only and **must not** be interpreted as upstream feature support.

## Next steps

- PR #9: image generation API, typed provider adapter, content limits, error handling and tests.
- PR #10: speech synthesis and transcription; size and duration limits.
- PR #11: asynchronous video jobs with status polling, expiry and cancellation. Keep Ploos AI Video Connector independent; optionally integrate through a provider adapter.
- Provider credentials, rate limits and privacy boundaries must be enforced separately for every operation.
- Never forward a request to a remote provider when a local-only model was selected.

## Compatibility

No new runtime routes or JSON registry fields are introduced in this PR. The existing registry's `modalities: ["text"]` remains unchanged. Future PRs should migrate that field deliberately rather than implying that all models automatically support every operation.
