# Speech synthesis (PR #14)

The authenticated `POST /v1/audio/speech` endpoint accepts `model`, `input` and `voice` and returns MP3 audio. This initial implementation buffers a bounded response (16 MiB); **incremental audio streaming is not yet implemented**. Only MP3 is supported.

Speech models are opt-in via `audio_providers` and `audio_models` in the shared JSON registry.

```json
{
  "audio_providers": [
    {"id":"local-tts","kind":"selfhosted","endpoint":"http://127.0.0.1:8000/v1/audio/speech"}
  ],
  "audio_models": [
    {"id":"speech/local","provider":"local-tts","upstream_model":"my-tts-model","location":"selfhosted","operations":["audio.speech"]}
  ]
}
```

The local backend **must expose an OpenAI-compatible speech endpoint**. This does not integrate Piper, Kokoro or other TTS engines directly. An external OpenAI provider may instead be configured with fixed endpoint `https://api.openai.com/v1/audio/speech` and `key_env: "OPENAI_API_KEY"`.

Local endpoints must be private or loopback IP literals, never arbitrary client URLs. The gateway rejects unknown aliases without fallback. The provider request is fixed to `response_format: "mp3"`. Request bodies are limited to 16 KiB and audio responses to 16 MiB. Credentials remain on the gateway.

This is a foundation for future streaming, configurable voices, local engine adapters, rate limiting and speech transcription.
