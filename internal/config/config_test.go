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
