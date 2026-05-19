package embedder

import (
	"context"
	"testing"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
)

func TestOpenAIEmbedderProvider_ModelAndDim(t *testing.T) {
	p := NewOpenAIEmbedderProvider("fake-key", "https://api.openai.com/v1", "text-embedding-3-large", 3072, 0)
	if p.GetModel() != "text-embedding-3-large" {
		t.Fatalf("unexpected model: %s", p.GetModel())
	}
	if p.GetDimensions() != 3072 {
		t.Fatalf("unexpected dim: %d", p.GetDimensions())
	}
}

func TestOpenAITextEmbedder(t *testing.T) {
	embedder := &OpenAITextEmbedder{
		model: "text-embedding-3-large",
		embedFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			if len(texts) != 1 || texts[0] != "hello" {
				t.Fatalf("unexpected texts: %v", texts)
			}
			return [][]float32{{0.1, 0.2, 0.3}}, nil
		},
	}
	result, err := embedder.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Embedding) != 3 {
		t.Fatalf("expected 3 dims, got %d", len(result.Embedding))
	}
}

func TestOpenAIDocumentEmbedder(t *testing.T) {
	embedder := &OpenAIDocumentEmbedder{
		model: "text-embedding-3-large",
		embedFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			if len(texts) != 2 {
				t.Fatalf("expected 2 texts, got %d", len(texts))
			}
			return [][]float32{{0.1, 0.2}, {0.3, 0.4}}, nil
		},
	}
	docs := []core.Document{
		{ID: "1", Content: "doc1"},
		{ID: "2", Content: "doc2"},
	}
	result, err := embedder.Run(context.Background(), docs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Documents) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(result.Documents))
	}
	if len(result.Documents[0].Embedding) != 2 {
		t.Fatalf("expected 2 dims for doc1, got %d", len(result.Documents[0].Embedding))
	}
}
