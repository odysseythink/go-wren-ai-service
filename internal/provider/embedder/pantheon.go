package embedder

import (
	"context"
	"fmt"
	"strings"

	"github.com/odysseythink/pantheon/extensions/embed"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
)

// PantheonEmbedderProvider wraps a Pantheon embed.Provider to implement core.EmbedderProvider.
type PantheonEmbedderProvider struct {
	provider  embed.Provider
	model     string
	dimension int
}

// NewPantheonEmbedderProvider creates a new Pantheon embedder provider.
func NewPantheonEmbedderProvider(provider embed.Provider, model string, dimension int) *PantheonEmbedderProvider {
	return &PantheonEmbedderProvider{
		provider:  provider,
		model:     model,
		dimension: dimension,
	}
}

// GetModel returns the embedding model identifier.
func (p *PantheonEmbedderProvider) GetModel() string {
	return p.model
}

// GetDimensions returns the embedding dimension.
func (p *PantheonEmbedderProvider) GetDimensions() int {
	return p.dimension
}

// GetTextEmbedder creates a text embedder.
func (p *PantheonEmbedderProvider) GetTextEmbedder(ctx context.Context) (core.TextEmbedder, error) {
	model, err := p.provider.EmbeddingModel(ctx, p.model)
	if err != nil {
		return nil, fmt.Errorf("get embedding model: %w", err)
	}
	return &PantheonTextEmbedder{
		model: model,
		name:  p.model,
	}, nil
}

// GetDocumentEmbedder creates a document embedder.
func (p *PantheonEmbedderProvider) GetDocumentEmbedder(ctx context.Context) (core.DocumentEmbedder, error) {
	model, err := p.provider.EmbeddingModel(ctx, p.model)
	if err != nil {
		return nil, fmt.Errorf("get embedding model: %w", err)
	}
	return &PantheonDocumentEmbedder{
		model: model,
		name:  p.model,
	}, nil
}

// PantheonTextEmbedder embeds a single text string.
type PantheonTextEmbedder struct {
	model embed.EmbeddingModel
	name  string
}

// Run embeds a single text.
func (e *PantheonTextEmbedder) Run(ctx context.Context, text string) (*core.EmbedResult, error) {
	text = strings.ReplaceAll(text, "\n", " ")
	resp, err := e.model.Embed(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("embed text: %w", err)
	}
	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return &core.EmbedResult{
		Embedding: resp.Embeddings[0],
		Meta:      map[string]any{"model": e.name},
	}, nil
}

// PantheonDocumentEmbedder embeds a batch of documents.
type PantheonDocumentEmbedder struct {
	model embed.EmbeddingModel
	name  string
}

// Run embeds a batch of documents.
func (e *PantheonDocumentEmbedder) Run(ctx context.Context, docs []core.Document) (*core.DocEmbedResult, error) {
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = strings.ReplaceAll(doc.Content, "\n", " ")
	}
	resp, err := e.model.Embed(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("embed documents: %w", err)
	}
	for i := range docs {
		docs[i].Embedding = resp.Embeddings[i]
	}
	return &core.DocEmbedResult{
		Documents: docs,
		Meta:      map[string]any{"model": e.name},
	}, nil
}
