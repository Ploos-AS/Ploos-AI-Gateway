package images

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ComfyGenerator executes a server-owned workflow with one prompt input.
type ComfyGenerator struct {
	Endpoint   string
	Client     *http.Client
	Workflow   json.RawMessage
	PromptNode string
	MaxDuration time.Duration
	MaxConcurrent int
	mu sync.Mutex
	active int
}

// Generate implements the image generator contract using ComfyUI's queue/history/view APIs.
func (g *ComfyGenerator) Generate(ctx context.Context, body []byte) (int, []byte, error) {
	g.mu.Lock()
	limit := g.MaxConcurrent
	if limit <= 0 {
		limit = 1
	}
	if g.active >= limit {
		g.mu.Unlock()
		return 0, nil, errors.New("comfyui capacity reached")
	}
	g.active++
	g.mu.Unlock()
	defer func() {
		g.mu.Lock()
		g.active--
		g.mu.Unlock()
	}()
	duration := g.MaxDuration
	if duration <= 0 {
		duration = 2 * time.Minute
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, duration)
	defer cancel()

	var input struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(body, &input); err != nil || strings.TrimSpace(input.Prompt) == "" {
		return 0, nil, errors.New("invalid prompt")
	}
	var workflow map[string]map[string]json.RawMessage
	if err := json.Unmarshal(g.Workflow, &workflow); err != nil || workflow[g.PromptNode] == nil {
		return 0, nil, errors.New("invalid workflow template")
	}
	node := workflow[g.PromptNode]
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(node["inputs"], &fields); err != nil || fields == nil {
		return 0, nil, errors.New("workflow missing prompt inputs")
	}
	prompt, _ := json.Marshal(input.Prompt)
	fields["text"] = prompt
	node["inputs"], _ = json.Marshal(fields)
	payload, _ := json.Marshal(map[string]interface{}{"prompt": workflow})
	status, queued, err := g.request(ctx, http.MethodPost, g.Endpoint, payload)
	if err != nil || status != http.StatusOK {
		return 0, nil, errors.New("comfyui queue failed")
	}
	var job struct {
		PromptID string `json:"prompt_id"`
	}
	if json.Unmarshal(queued, &job) != nil || job.PromptID == "" {
		return 0, nil, errors.New("invalid comfyui job")
	}
	base := strings.TrimSuffix(g.Endpoint, "/prompt")
	defer g.deleteQueuedJob(base, job.PromptID)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return 0, nil, ctx.Err()
		case <-ticker.C:
			path := base + "/history/" + url.PathEscape(job.PromptID)
			code, result, err := g.request(ctx, http.MethodGet, path, nil)
			if err != nil || code != http.StatusOK {
				return 0, nil, errors.New("comfyui history failed")
			}
			var history map[string]struct {
				Status struct {
					StatusStr string `json:"status_str"`
				} `json:"status"`
				Outputs map[string]struct {
					Images []struct {
						Filename  string `json:"filename"`
						Subfolder string `json:"subfolder"`
						Type      string `json:"type"`
					} `json:"images"`
				} `json:"outputs"`
			}
			if json.Unmarshal(result, &history) != nil {
				return 0, nil, errors.New("invalid comfyui history")
			}
			entry, done := history[job.PromptID]
			if !done {
				continue
			}
			if entry.Status.StatusStr == "error" {
				return 0, nil, errors.New("comfyui job failed")
			}
			for _, output := range entry.Outputs {
				for _, item := range output.Images {
					if item.Filename == "" || item.Type != "output" {
						continue
					}
					query := url.Values{"filename": {item.Filename}, "subfolder": {item.Subfolder}, "type": {"output"}}
					code, data, err := g.request(ctx, http.MethodGet, base+"/view?"+query.Encode(), nil)
					if err != nil || code != http.StatusOK || len(data) == 0 {
						return 0, nil, errors.New("comfyui image retrieval failed")
					}
					mime := http.DetectContentType(data)
					if mime != "image/png" && mime != "image/jpeg" && mime != "image/webp" {
						return 0, nil, errors.New("unexpected image type")
					}
					response, _ := json.Marshal(map[string]interface{}{"created": time.Now().Unix(), "data": []map[string]string{{"b64_json": base64.StdEncoding.EncodeToString(data)}}})
					if int64(len(response)) > MaxResponseBytes {
						return 0, nil, errors.New("image too large")
					}
					return http.StatusOK, response, nil
				}
			}
			return 0, nil, errors.New("comfyui job completed without image")
		}
	}
}

func (g *ComfyGenerator) request(ctx context.Context, method, endpoint string, body []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := g.Client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return 0, nil, errors.New("redirect refused")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	if err != nil || int64(len(data)) > MaxResponseBytes {
		return 0, nil, fmt.Errorf("comfyui response exceeded limit")
	}
	return resp.StatusCode, data, nil
}

func (g *ComfyGenerator) deleteQueuedJob(base, id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	body, _ := json.Marshal(map[string]interface{}{"delete": []string{id}})
	_, _, _ = g.request(ctx, http.MethodPost, base+"/queue", body)
}
