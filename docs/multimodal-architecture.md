# Multimodal operation contracts

PR #8 introduced provider-independent operation descriptors. PRs #9–#10 subsequently added the first live image endpoint and shared image model configuration.

| Operation | Input | Output | Runtime status |
| --- | --- | --- | --- |
| `chat.completions` | text | text | Implemented |
| `images.generate` | text | image | Implemented for OpenAI |
| `audio.speech` | text | audio | Contract only |
| `audio.transcriptions` | audio | text | Contract only |
| `videos.generate` | text/image | video | Contract only |

An operation is not the same as a modality. The catalog does not grant provider support: individual image routes must explicitly declare `images.generate`.

## Planned work

- Native ComfyUI workflow adapter, safe templates, job polling and bounded image retrieval.
- Speech synthesis and transcription with distinct contracts and limits.
- Asynchronous video job management, optionally integrating the independent Ploos AI Video Connector.
- Per-operation provider credentials, rate limits, cost controls and privacy policies.

Private/local-only requests must never be automatically routed to a remote provider.

Chat registry entries still use `modalities: ["text"]`; image entries use the separate `image_models` registry section. A future unified capability schema should be migrated deliberately.
