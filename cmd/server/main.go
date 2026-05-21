package main

import (
	"fmt"
	"log"
	"net/http"

	openai "github.com/odysseythink/pantheon/providers/openai"
	"github.com/odysseythink/pantheon/extensions/embed"
	"github.com/odysseythink/go-wren-ai-service/internal/config"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/handler"
	"github.com/odysseythink/go-wren-ai-service/internal/provider/docstore"
	"github.com/odysseythink/go-wren-ai-service/internal/provider/embedder"
	"github.com/odysseythink/go-wren-ai-service/internal/provider/engine"
	"github.com/odysseythink/go-wren-ai-service/internal/provider/llm"
	"github.com/odysseythink/go-wren-ai-service/internal/service"
)

func main() {
	cfg := config.Load()

	// Initialize providers
	pantheonProvider, err := openai.New(cfg.LLMOpenAIAPIKey, openai.WithBaseURL(cfg.LLMOpenAIAPIBase))
	if err != nil {
		log.Fatalf("failed to create pantheon provider: %v", err)
	}

	llmProvider := llm.NewPantheonLLMProvider(pantheonProvider, cfg.GenerationModel, map[string]any{
		"temperature": 0,
		"n":          1,
		"max_tokens": 4096,
		"response_format": map[string]any{"type": "json_object"},
	})

	embedProvider, ok := pantheonProvider.(embed.Provider)
	if !ok {
		log.Fatalf("pantheon provider does not support embedding")
	}
	embedderProvider := embedder.NewPantheonEmbedderProvider(
		embedProvider,
		cfg.EmbeddingModel,
		cfg.EmbeddingModelDim,
	)

	docStoreProvider := docstore.NewQdrantProvider(
		fmt.Sprintf("http://%s:6333", cfg.QdrantHost),
		cfg.QdrantAPIKey,
		cfg.EmbeddingModelDim,
		cfg.QdrantTimeout,
	)

	var eng core.Engine
	switch cfg.Engine {
	case "wren_ibis":
		eng = engine.NewWrenIbis(cfg.WrenIbisEndpoint, cfg.WrenIbisSource, cfg.WrenIbisManifest, nil)
	case "wren_engine":
		eng = engine.NewWrenEngine(cfg.WrenEngineEndpoint, nil)
	default:
		eng = engine.NewWrenUI(cfg.WrenUIEndpoint)
	}

	components := core.PipelineComponent{
		LLMProvider:      llmProvider,
		EmbedderProvider: embedderProvider,
		DocStoreProvider: docStoreProvider,
		Engine:           eng,
	}

	container := service.NewContainer(components, cfg)
	router := handler.NewRouter(container)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("go-wren-ai-service starting on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
