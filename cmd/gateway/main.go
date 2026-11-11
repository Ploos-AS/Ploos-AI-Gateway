package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/api"
	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/providers"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	addr := os.Getenv("PLOOS_GATEWAY_LISTEN")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}

	var provider providers.ChatProvider
	switch os.Getenv("PLOOS_GATEWAY_PROVIDER") {
	case "":
		// No model endpoint configured.
	case "openrouter":
		key := os.Getenv("OPENROUTER_API_KEY")
		if key == "" {
			logger.Error("OPENROUTER_API_KEY required for openrouter provider")
			os.Exit(1)
		}
		provider, _ = providers.NewHTTPChatProvider("https://openrouter.ai/api/v1/chat/completions", key)
	case "ollama":
		// Default loopback Ollama endpoint; configure network access outside this process.
		provider, _ = providers.NewHTTPChatProvider("http://127.0.0.1:11434/v1/chat/completions", "")
	default:
		logger.Error("unknown provider")
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewHandlerWithProvider(os.Getenv("PLOOS_GATEWAY_API_KEY"), provider),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
		}
	}()

	logger.Info("gateway starting", "listen", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}
