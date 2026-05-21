package config

import (
	"os"
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	cfg := Load()
	if cfg.Port != 8000 {
		t.Fatalf("expected default port 8000, got %d", cfg.Port)
	}
	if cfg.GenerationModel != "gpt-4o-mini" {
		t.Fatalf("expected default model gpt-4o-mini, got %s", cfg.GenerationModel)
	}
	if cfg.Engine != "wren_ui" {
		t.Fatalf("expected default engine wren_ui, got %s", cfg.Engine)
	}
	if cfg.EmbeddingModel != "text-embedding-3-large" {
		t.Fatalf("expected default embedding model, got %s", cfg.EmbeddingModel)
	}
	if cfg.EmbeddingModelDim != 3072 {
		t.Fatalf("expected default dim 3072, got %d", cfg.EmbeddingModelDim)
	}
}

func TestConfigFromEnv(t *testing.T) {
	os.Setenv("WREN_AI_SERVICE_PORT", "9000")
	os.Setenv("GENERATION_MODEL", "gpt-4o")
	defer os.Unsetenv("WREN_AI_SERVICE_PORT")
	defer os.Unsetenv("GENERATION_MODEL")

	cfg := Load()
	if cfg.Port != 9000 {
		t.Fatalf("expected port 9000 from env, got %d", cfg.Port)
	}
	if cfg.GenerationModel != "gpt-4o" {
		t.Fatalf("expected model gpt-4o from env, got %s", cfg.GenerationModel)
	}
}

func TestParseYAML_MultiDoc(t *testing.T) {
	yaml := `
type: llm
provider: openai_llm
models:
  - model: gpt-4o-mini
    kwargs:
      temperature: 0
api_key: sk-test

---
type: embedder
provider: openai_embedder
models:
  - model: text-embedding-3-large
    dimension: 3072
api_key: sk-test

---
type: pipeline
pipes:
  - name: sql_generation
    llm: openai_llm.gpt-4o-mini
    engine: wren_ui
`
	raw, err := ParseYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if raw == nil {
		t.Fatal("expected non-nil RawConfig")
	}
	if len(raw.LLMs) != 1 {
		t.Fatalf("expected 1 LLM, got %d", len(raw.LLMs))
	}
	if len(raw.Embedders) != 1 {
		t.Fatalf("expected 1 Embedder, got %d", len(raw.Embedders))
	}
	if len(raw.Pipelines) != 1 {
		t.Fatalf("expected 1 Pipeline, got %d", len(raw.Pipelines))
	}

	llm := raw.LLMs["openai_llm.gpt-4o-mini"]
	if llm.Provider != "openai_llm" {
		t.Fatalf("expected provider openai_llm, got %s", llm.Provider)
	}
	if llm.Config["api_key"] != "sk-test" {
		t.Fatalf("expected api_key sk-test, got %v", llm.Config["api_key"])
	}

	pipe := raw.Pipelines["sql_generation"]
	if pipe.LLM != "openai_llm.gpt-4o-mini" {
		t.Fatalf("expected llm openai_llm.gpt-4o-mini, got %s", pipe.LLM)
	}
}

func TestParseYAML_Empty(t *testing.T) {
	raw, err := ParseYAML([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw == nil {
		t.Fatal("expected non-nil RawConfig")
	}
	if len(raw.LLMs) != 0 || len(raw.Embedders) != 0 || len(raw.Pipelines) != 0 {
		t.Fatal("expected empty config")
	}
}

func TestParseYAML_MissingOptionalFields(t *testing.T) {
	yaml := `
type: llm
provider: openai_llm
models:
  - model: gpt-4o-mini
`
	raw, err := ParseYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(raw.LLMs) != 1 {
		t.Fatalf("expected 1 LLM, got %d", len(raw.LLMs))
	}
	llm := raw.LLMs["openai_llm.gpt-4o-mini"]
	if llm.Config["api_key"] != nil {
		t.Fatal("expected nil api_key")
	}
}

func TestParseYAML_EngineOnly(t *testing.T) {
	yaml := `
type: engine
provider: wren_ui
endpoint: http://localhost:3000
`
	raw, err := ParseYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(raw.Engines) != 1 {
		t.Fatalf("expected 1 engine, got %d", len(raw.Engines))
	}
	eng := raw.Engines["wren_ui"]
	if eng.Config["endpoint"] != "http://localhost:3000" {
		t.Fatalf("expected endpoint, got %v", eng.Config["endpoint"])
	}
}
