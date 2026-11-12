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
	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/images"
	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/providers"
	"github.com/Ploos-AS/Ploos-AI-Gateway/internal/routing"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	addr := os.Getenv("PLOOS_GATEWAY_LISTEN")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}

	var handler http.Handler
	if path := os.Getenv("PLOOS_GATEWAY_REGISTRY"); path != "" {
		cfg, err := routing.Load(path)
		if err != nil {
			logger.Error("cannot load model registry", "error", err)
			os.Exit(1)
		}
		registry, err := routing.New(cfg)
		if err != nil {
			logger.Error("invalid model registry", "error", err)
			os.Exit(1)
		}
		handler = api.NewRoutedHandler(os.Getenv("PLOOS_GATEWAY_API_KEY"), registry)
		if len(cfg.ImageProviders) > 0 || len(cfg.ImageModels) > 0 {
			imageRegistry, err := images.NewRegistry(cfg.ImageProviders, cfg.ImageModels)
			if err != nil {
				logger.Error("invalid image registry", "error", err)
				os.Exit(1)
			}
			handler = api.WithImageRegistry(handler, os.Getenv("PLOOS_GATEWAY_API_KEY"), func(id string) (api.ImageRoute, bool) {
				route, ok := imageRegistry.Resolve(id)
				return api.ImageRoute{Alias: id, UpstreamModel: route.UpstreamModel, Generator: route.Generator}, ok
			})
		}
	} else {
		var provider providers.ChatProvider
		switch os.Getenv("PLOOS_GATEWAY_PROVIDER") {
		case "":
		case "openrouter":
			key := os.Getenv("OPENROUTER_API_KEY")
			if key == "" {
				logger.Error("OPENROUTER_API_KEY required for openrouter provider")
				os.Exit(1)
			}
			provider, _ = providers.NewHTTPChatProvider("https://openrouter.ai/api/v1/chat/completions", key)
		case "ollama":
			provider, _ = providers.NewHTTPChatProvider("http://127.0.0.1:11434/v1/chat/completions", "")
		default:
			logger.Error("unknown provider")
			os.Exit(1)
		}
		handler = api.NewHandlerWithProvider(os.Getenv("PLOOS_GATEWAY_API_KEY"), provider)
	}

	if alias := os.Getenv("PLOOS_IMAGE_MODEL_ALIAS"); alias != "" {
		if os.Getenv("PLOOS_GATEWAY_REGISTRY") != "" {
			logger.Error("legacy image environment routing cannot be combined with image registry")
			os.Exit(1)
		}
		model := os.Getenv("PLOOS_IMAGE_UPSTREAM_MODEL")
		generator, err := images.NewOpenAI(os.Getenv("OPENAI_API_KEY"))
		if err != nil || model == "" {
			logger.Error("image generation requires OPENAI_API_KEY and PLOOS_IMAGE_UPSTREAM_MODEL")
			os.Exit(1)
		}
		handler = api.WithImageGeneration(handler, os.Getenv("PLOOS_GATEWAY_API_KEY"), api.ImageRoute{
			Alias: alias, UpstreamModel: model, Generator: generator,
		})
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
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
