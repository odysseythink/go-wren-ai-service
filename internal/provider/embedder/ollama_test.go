package embedder

import (
	"testing"

	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

func TestOllamaEmbedderFactory(t *testing.T) {
	f, ok := provider.GetEmbedderFactory("ollama_embedder")
	if !ok {
		t.Fatal("ollama_embedder factory not registered")
	}

	p, err := f(map[string]any{
		"url":       "http://localhost:11434",
		"model":     "nomic-embed-text:latest",
		"dimension": 768,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetModel() != "nomic-embed-text:latest" {
		t.Fatalf("expected model nomic-embed-text:latest, got %s", p.GetModel())
	}
	if p.GetDimensions() != 768 {
		t.Fatalf("expected dimension 768, got %d", p.GetDimensions())
	}
}
