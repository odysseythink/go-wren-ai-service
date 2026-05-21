package service

import (
	"encoding/json"
	"fmt"

	"github.com/odysseythink/pantheon/extensions/embed"
	openai "github.com/odysseythink/pantheon/providers/openai"
	"github.com/odysseythink/go-wren-ai-service/internal/config"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/provider"
	"github.com/odysseythink/go-wren-ai-service/internal/provider/docstore"
	"github.com/odysseythink/go-wren-ai-service/internal/provider/embedder"
	"github.com/odysseythink/go-wren-ai-service/internal/provider/engine"
	"github.com/odysseythink/go-wren-ai-service/internal/provider/llm"
)

// GenerateComponents creates per-pipeline PipelineComponent map from config.
func GenerateComponents(cfg *config.Config) (map[string]core.PipelineComponent, error) {
	if cfg.Raw == nil || len(cfg.Raw.Pipelines) == 0 {
		return generateFromEnv(cfg)
	}

	llms := map[string]core.LLMProvider{}
	for id, entry := range cfg.Raw.LLMs {
		f, ok := provider.GetLLMFactory(entry.Provider)
		if !ok {
			return nil, fmt.Errorf("unknown llm provider: %s", entry.Provider)
		}
		p, err := f(entry.Config)
		if err != nil {
			return nil, fmt.Errorf("init llm %s: %w", id, err)
		}
		llms[id] = p
	}

	embedders := map[string]core.EmbedderProvider{}
	for id, entry := range cfg.Raw.Embedders {
		f, ok := provider.GetEmbedderFactory(entry.Provider)
		if !ok {
			return nil, fmt.Errorf("unknown embedder provider: %s", entry.Provider)
		}
		p, err := f(entry.Config)
		if err != nil {
			return nil, fmt.Errorf("init embedder %s: %w", id, err)
		}
		embedders[id] = p
	}

	docStores := map[string]core.DocStoreProvider{}
	for id, entry := range cfg.Raw.DocumentStores {
		f, ok := provider.GetDocStoreFactory(entry.Provider)
		if !ok {
			return nil, fmt.Errorf("unknown docstore provider: %s", entry.Provider)
		}
		p, err := f(entry.Config)
		if err != nil {
			return nil, fmt.Errorf("init docstore %s: %w", id, err)
		}
		docStores[id] = p
	}

	engines := map[string]core.Engine{}
	for id, entry := range cfg.Raw.Engines {
		f, ok := provider.GetEngineFactory(entry.Provider)
		if !ok {
			return nil, fmt.Errorf("unknown engine provider: %s", entry.Provider)
		}
		p, err := f(entry.Config)
		if err != nil {
			return nil, fmt.Errorf("init engine %s: %w", id, err)
		}
		engines[id] = p
	}

	components := make(map[string]core.PipelineComponent)
	for name, pipe := range cfg.Raw.Pipelines {
		comp := core.PipelineComponent{}
		if pipe.LLM != "" {
			if p, ok := llms[pipe.LLM]; ok {
				comp.LLMProvider = p
			} else {
				return nil, fmt.Errorf("pipeline %q references unknown llm: %s", name, pipe.LLM)
			}
		}
		if pipe.Embedder != "" {
			if p, ok := embedders[pipe.Embedder]; ok {
				comp.EmbedderProvider = p
			} else {
				return nil, fmt.Errorf("pipeline %q references unknown embedder: %s", name, pipe.Embedder)
			}
		}
		if pipe.DocumentStore != "" {
			if p, ok := docStores[pipe.DocumentStore]; ok {
				comp.DocStoreProvider = p
			} else {
				return nil, fmt.Errorf("pipeline %q references unknown docstore: %s", name, pipe.DocumentStore)
			}
		}
		if pipe.Engine != "" {
			if p, ok := engines[pipe.Engine]; ok {
				comp.Engine = p
			} else {
				return nil, fmt.Errorf("pipeline %q references unknown engine: %s", name, pipe.Engine)
			}
		}
		components[name] = comp
	}

	return components, nil
}

func generateFromEnv(cfg *config.Config) (map[string]core.PipelineComponent, error) {
	openaiProvider, err := openai.New(cfg.LLMOpenAIAPIKey, openai.WithBaseURL(cfg.LLMOpenAIAPIBase))
	if err != nil {
		return nil, fmt.Errorf("create openai provider: %w", err)
	}

	llmProvider := llm.NewPantheonLLMProvider(openaiProvider, cfg.GenerationModel, map[string]any{
		"temperature":     0,
		"n":               1,
		"max_tokens":      4096,
		"response_format": map[string]any{"type": "json_object"},
	})

	embedProvider, ok := openaiProvider.(embed.Provider)
	if !ok {
		return nil, fmt.Errorf("pantheon provider does not support embedding")
	}
	embedderProvider := embedder.NewPantheonEmbedderProvider(embedProvider, cfg.EmbeddingModel, cfg.EmbeddingModelDim)

	docStoreProvider := docstore.NewQdrantProvider(
		fmt.Sprintf("http://%s:6333", cfg.QdrantHost),
		cfg.QdrantAPIKey,
		cfg.EmbeddingModelDim,
		cfg.QdrantTimeout,
	)

	var eng core.Engine
	switch cfg.Engine {
	case "wren_ibis":
		var connInfo map[string]any
		if cfg.WrenIbisConnInfo != "" {
			_ = json.Unmarshal([]byte(cfg.WrenIbisConnInfo), &connInfo)
		}
		eng = engine.NewWrenIbis(cfg.WrenIbisEndpoint, cfg.WrenIbisSource, cfg.WrenIbisManifest, connInfo)
	case "wren_engine":
		var manifest map[string]any
		if cfg.WrenEngineManifest != "" {
			_ = json.Unmarshal([]byte(cfg.WrenEngineManifest), &manifest)
		}
		eng = engine.NewWrenEngine(cfg.WrenEngineEndpoint, manifest)
	default:
		eng = engine.NewWrenUI(cfg.WrenUIEndpoint)
	}

	comp := core.PipelineComponent{
		LLMProvider:      llmProvider,
		EmbedderProvider: embedderProvider,
		DocStoreProvider: docStoreProvider,
		Engine:           eng,
	}

	pipelines := []string{
		"indexing", "retrieval", "historical_question",
		"sql_generation", "sql_correction", "followup_sql_generation",
		"sql_summary", "sql_answer", "sql_breakdown",
		"sql_expansion", "sql_explanation", "sql_regeneration",
		"semantics_description", "relationship_recommendation",
	}
	components := make(map[string]core.PipelineComponent)
	for _, name := range pipelines {
		components[name] = comp
	}
	return components, nil
}
