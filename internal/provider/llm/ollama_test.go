package llm

import (
	"testing"

	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

func TestOllamaLLMFactory(t *testing.T) {
	f, ok := provider.GetLLMFactory("ollama_llm")
	if !ok {
		t.Fatal("ollama_llm factory not registered")
	}

	p, err := f(map[string]any{
		"url":   "http://localhost:11434",
		"model": "gemma2:9b",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetModel() != "gemma2:9b" {
		t.Fatalf("expected model gemma2:9b, got %s", p.GetModel())
	}
}
