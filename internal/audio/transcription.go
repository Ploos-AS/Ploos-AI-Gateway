package audio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

const MaxTranscriptionBytes int64 = 25 << 20

type TranscriptionProvider struct {
	Endpoint string
	Key      string
	Client   *http.Client
}

func (p *TranscriptionProvider) Transcribe(ctx context.Context, model, filename string, audio []byte) ([]byte, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if err := w.WriteField("model", model); err != nil {
		return nil, err
	}
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err = part.Write(audio); err != nil {
		return nil, err
	}
	if err = w.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, &b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if p.Key != "" {
		req.Header.Set("Authorization", "Bearer "+p.Key)
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, errors.New("transcription provider unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("transcription provider rejected request")
	}
	if !strings.HasPrefix(strings.ToLower(resp.Header.Get("Content-Type")), "application/json") {
		return nil, errors.New("unexpected transcription response type")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return nil, errors.New("invalid transcription response size")
	}
	var result struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(data, &result) != nil {
		return nil, errors.New("invalid transcription response")
	}
	return json.Marshal(result)
}
