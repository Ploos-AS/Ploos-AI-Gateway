# Audio transcription (PR #15)

The gateway exposes authenticated `POST /v1/audio/transcriptions` with multipart fields `model` and `file`. The response is OpenAI-style JSON with a `text` field.

Configure `transcription_providers` and `transcription_models` in the shared registry:

```json
{
  "transcription_providers": [
    {"id":"local-whisper","kind":"selfhosted","endpoint":"http://127.0.0.1:8000/v1/audio/transcriptions"}
  ],
  "transcription_models": [
    {"id":"transcribe/local","provider":"local-whisper","upstream_model":"whisper-1","location":"selfhosted","operations":["audio.transcriptions"]}
  ]
}
```

Local services must expose an OpenAI-compatible multipart transcription endpoint. Remote OpenAI is supported only with the exact `https://api.openai.com/v1/audio/transcriptions` endpoint and `key_env: "OPENAI_API_KEY"`. Local requests never fall back to remote providers.

Uploaded audio is limited to 25 MiB and accepted extensions are mp3, mp4, m4a, wav, webm, ogg, and flac. Extensions are only preliminary validation, not proof of file format. Audio is buffered in memory for this first implementation. Upstream responses must be JSON, limited to 1 MiB. The response is normalized to `{"text":"..."}`. No audio files are deliberately persisted by the gateway, but deployment logging, reverse proxies and upstream services require separate review.

Language hints, timestamps, diarization, native Whisper adapters and streaming remain out of scope.
