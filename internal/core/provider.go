package core

import "context"

// GeneratorOpts configures a generator instance.
type GeneratorOpts struct {
	SystemPrompt     string
	GenerationKwargs map[string]any
}

// GenerateResult holds LLM generation output.
type GenerateResult struct {
	Replies []string
	Meta    []map[string]any
}

// EmbedResult holds a single text embedding result.
type EmbedResult struct {
	Embedding []float32
	Meta      map[string]any
}

// DocEmbedResult holds batch document embedding output.
type DocEmbedResult struct {
	Documents []Document
	Meta      map[string]any
}

// Document represents a vector-store document.
type Document struct {
	ID        string
	Content   string
	Meta      map[string]any
	Embedding []float32
	Score     float32
}

// StoreOpts configures document store creation.
type StoreOpts struct {
	DatasetName   string
	RecreateIndex bool
}

// WritePolicy controls duplicate document handling.
type WritePolicy int

const (
	WritePolicyFail WritePolicy = iota
	WritePolicyOverwrite
	WritePolicySkip
)

// RetrievalResult holds vector search results.
type RetrievalResult struct {
	Documents []Document
}

// LLMProvider creates generators for LLM calls.
type LLMProvider interface {
	GetGenerator(ctx context.Context, opts GeneratorOpts) (Generator, error)
	GetModel() string
	GetModelKwargs() map[string]any
}

// Generator performs a single LLM call.
type Generator interface {
	Run(ctx context.Context, prompt string) (*GenerateResult, error)
}

// EmbedderProvider creates embedder instances.
type EmbedderProvider interface {
	GetTextEmbedder(ctx context.Context) (TextEmbedder, error)
	GetDocumentEmbedder(ctx context.Context) (DocumentEmbedder, error)
	GetModel() string
	GetDimensions() int
}

// TextEmbedder embeds a single text string.
type TextEmbedder interface {
	Run(ctx context.Context, text string) (*EmbedResult, error)
}

// DocumentEmbedder embeds a batch of documents.
type DocumentEmbedder interface {
	Run(ctx context.Context, docs []Document) (*DocEmbedResult, error)
}

// DocStoreProvider creates document store and retriever instances.
type DocStoreProvider interface {
	GetStore(opts StoreOpts) DocumentStore
	GetRetriever(store DocumentStore, topK int) Retriever
}

// DocumentStore provides vector document storage.
type DocumentStore interface {
	WriteDocuments(ctx context.Context, docs []Document, policy WritePolicy) (int, error)
	DeleteDocuments(ctx context.Context, filters map[string]any) error
	QueryByEmbedding(ctx context.Context, embedding []float32, filters map[string]any, topK int) ([]Document, error)
}

// Retriever performs vector similarity search.
type Retriever interface {
	Run(ctx context.Context, queryEmbedding []float32, filters map[string]any) (*RetrievalResult, error)
}

// Engine executes SQL against a backend.
type Engine interface {
	ExecuteSQL(ctx context.Context, sql string, opts EngineOpts) (*EngineResult, error)
}
