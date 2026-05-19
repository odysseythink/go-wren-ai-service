package embedder

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
)

// OpenAIEmbedderProvider wraps the OpenAI Embedding API to implement core.EmbedderProvider.
type OpenAIEmbedderProvider struct {
	client    *openai.Client
	model     string
	dimension int
}

// NewOpenAIEmbedderProvider creates a new OpenAI embedder provider.
func NewOpenAIEmbedderProvider(apiKey, apiBase, model string, dimension int, timeout time.Duration) *OpenAIEmbedderProvider {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = strings.TrimSuffix(apiBase, "/")
	if timeout > 0 {
		config.HTTPClient = &http.Client{Timeout: timeout}
	}
	return &OpenAIEmbedderProvider{
		client:    openai.NewClientWithConfig(config),
		model:     model,
		dimension: dimension,
	}
}

// GetModel returns the embedding model identifier.
func (p *OpenAIEmbedderProvider) GetModel() string {
	return p.model
}

// GetDimensions returns the embedding dimension.
func (p *OpenAIEmbedderProvider) GetDimensions() int {
	return p.dimension
}

// GetTextEmbedder creates a text embedder.
func (p *OpenAIEmbedderProvider) GetTextEmbedder(ctx context.Context) (core.TextEmbedder, error) {
	return &OpenAITextEmbedder{
		model:     p.model,
		embedFunc: p.embed,
	}, nil
}

// GetDocumentEmbedder creates a document embedder.
func (p *OpenAIEmbedderProvider) GetDocumentEmbedder(ctx context.Context) (core.DocumentEmbedder, error) {
	return &OpenAIDocumentEmbedder{
		model:     p.model,
		embedFunc: p.embed,
	}, nil
}

func (p *OpenAIEmbedderProvider) embed(ctx context.Context, texts []string) ([][]float32, error) {
	resp, err := p.client.CreateEmbeddings(ctx, openai.EmbeddingRequestStrings{
		Input: texts,
		Model: openai.EmbeddingModel(p.model),
	})
	if err != nil {
		return nil, fmt.Errorf("create embeddings: %w", err)
	}
	var result [][]float32
	for _, data := range resp.Data {
		emb := make([]float32, len(data.Embedding))
		for i, v := range data.Embedding {
			emb[i] = float32(v)
		}
		result = append(result, emb)
	}
	return result, nil
}

// OpenAITextEmbedder embeds a single text string.
type OpenAITextEmbedder struct {
	model     string
	embedFunc func(ctx context.Context, texts []string) ([][]float32, error)
}

// Run embeds a single text.
func (e *OpenAITextEmbedder) Run(ctx context.Context, text string) (*core.EmbedResult, error) {
	text = strings.ReplaceAll(text, "\n", " ")
	embeddings, err := e.embedFunc(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return &core.EmbedResult{
		Embedding: embeddings[0],
		Meta:      map[string]any{"model": e.model},
	}, nil
}

// OpenAIDocumentEmbedder embeds a batch of documents.
type OpenAIDocumentEmbedder struct {
	model     string
	embedFunc func(ctx context.Context, texts []string) ([][]float32, error)
}

// Run embeds a batch of documents.
func (e *OpenAIDocumentEmbedder) Run(ctx context.Context, docs []core.Document) (*core.DocEmbedResult, error) {
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = strings.ReplaceAll(doc.Content, "\n", " ")
	}
	embeddings, err := e.embedFunc(ctx, texts)
	if err != nil {
		return nil, err
	}
	for i := range docs {
		docs[i].Embedding = embeddings[i]
	}
	return &core.DocEmbedResult{
		Documents: docs,
		Meta:      map[string]any{"model": e.model},
	}, nil
}
