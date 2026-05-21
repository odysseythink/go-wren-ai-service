package embedder

import (
	"testing"

	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

func TestOpenAIEmbedderFactory(t *testing.T) {
	f, ok := provider.GetEmbedderFactory("openai_embedder")
	if !ok {
		t.Fatal("openai_embedder factory not registered")
	}

	p, err := f(map[string]any{
		"api_key":   "test-key",
		"model":     "text-embedding-3-small",
		"dimension": 1536,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetModel() != "text-embedding-3-small" {
		t.Fatalf("expected model text-embedding-3-small, got %s", p.GetModel())
	}
	if p.GetDimensions() != 1536 {
		t.Fatalf("expected dimension 1536, got %d", p.GetDimensions())
	}
}
