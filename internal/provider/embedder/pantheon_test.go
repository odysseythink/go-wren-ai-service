package embedder

import (
	"context"
	"testing"

	pantheonCore "github.com/odysseythink/pantheon/core"
	"github.com/odysseythink/pantheon/extensions/embed"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
)

type fakeEmbedProvider struct{}

func (f *fakeEmbedProvider) Name() string { return "fake" }
func (f *fakeEmbedProvider) Models(ctx context.Context) ([]pantheonCore.Model, error) { return nil, nil }
func (f *fakeEmbedProvider) LanguageModel(ctx context.Context, modelID string) (pantheonCore.LanguageModel, error) { return nil, nil }
func (f *fakeEmbedProvider) EmbeddingModel(ctx context.Context, modelID string) (embed.EmbeddingModel, error) {
	return &fakeEmbeddingModel{}, nil
}

type fakeEmbeddingModel struct{}

func (f *fakeEmbeddingModel) Embed(ctx context.Context, texts []string) (*embed.EmbeddingResponse, error) {
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		embeddings[i] = []float32{0.1, 0.2, 0.3}
	}
	return &embed.EmbeddingResponse{Embeddings: embeddings}, nil
}

func TestPantheonEmbedderProvider_ModelAndDim(t *testing.T) {
	p := NewPantheonEmbedderProvider(&fakeEmbedProvider{}, "text-embedding-3-large", 3072)
	if p.GetModel() != "text-embedding-3-large" {
		t.Fatalf("unexpected model: %s", p.GetModel())
	}
	if p.GetDimensions() != 3072 {
		t.Fatalf("unexpected dim: %d", p.GetDimensions())
	}
}

func TestPantheonTextEmbedder(t *testing.T) {
	provider := NewPantheonEmbedderProvider(&fakeEmbedProvider{}, "text-embedding-3-large", 3072)
	embedder, err := provider.GetTextEmbedder(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := embedder.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Embedding) != 3 {
		t.Fatalf("expected 3 dims, got %d", len(result.Embedding))
	}
}

func TestPantheonDocumentEmbedder(t *testing.T) {
	provider := NewPantheonEmbedderProvider(&fakeEmbedProvider{}, "text-embedding-3-large", 3072)
	embedder, err := provider.GetDocumentEmbedder(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
	if len(result.Documents[0].Embedding) != 3 {
		t.Fatalf("expected 3 dims for doc1, got %d", len(result.Documents[0].Embedding))
	}
}
