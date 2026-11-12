package images

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestComfyUIConcurrencyLimit(t *testing.T) {
	var queued atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/prompt" {
			queued.Add(1)
			close(started)
			<-release
			_, _ = w.Write([]byte(`{"prompt_id":"job-1"}`))
			return
		}
		if r.URL.Path == "/history/job-1" {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	g := &ComfyGenerator{
		Endpoint:      server.URL + "/prompt",
		Client:        server.Client(),
		Workflow:      []byte(`{"6":{"inputs":{"text":"old"}}}`),
		PromptNode:    "6",
		MaxConcurrent: 1,
		MaxDuration:   2 * time.Second,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, _ = g.Generate(ctx, []byte(`{"prompt":"first"}`))
	}()
	<-started
	_, _, err := g.Generate(context.Background(), []byte(`{"prompt":"second"}`))
	if err == nil || !strings.Contains(err.Error(), "capacity") {
		t.Fatalf("expected capacity error, got %v", err)
	}
	cancel()
	close(release)
	<-done
	if queued.Load() != 1 {
		t.Fatalf("queued %d jobs, want one", queued.Load())
	}
}

func TestComfyUITimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/prompt" {
			_, _ = w.Write([]byte(`{"prompt_id":"job-1"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	g := &ComfyGenerator{
		Endpoint:    server.URL + "/prompt",
		Client:      server.Client(),
		Workflow:    []byte(`{"6":{"inputs":{"text":"old"}}}`),
		PromptNode:  "6",
		MaxDuration: 50 * time.Millisecond,
	}
	_, _, err := g.Generate(context.Background(), []byte(`{"prompt":"test"}`))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}
