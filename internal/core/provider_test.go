package core

import (
	"context"
	"testing"
)

// --- Mocks for interface compilation ---

type mockLLMProvider struct {
	model string
}

func (m *mockLLMProvider) GetGenerator(ctx context.Context, opts GeneratorOpts) (Generator, error) {
	return nil, nil
}
func (m *mockLLMProvider) GetModel() string            { return m.model }
func (m *mockLLMProvider) GetModelKwargs() map[string]any { return nil }

type mockGenerator struct{}

func (m *mockGenerator) Run(ctx context.Context, prompt string) (*GenerateResult, error) {
	return &GenerateResult{Replies: []string{"test"}}, nil
}

type mockEmbedderProvider struct {
	model     string
	dimension int
}

func (m *mockEmbedderProvider) GetTextEmbedder(ctx context.Context) (TextEmbedder, error)     { return nil, nil }
func (m *mockEmbedderProvider) GetDocumentEmbedder(ctx context.Context) (DocumentEmbedder, error) { return nil, nil }
func (m *mockEmbedderProvider) GetModel() string  { return m.model }
func (m *mockEmbedderProvider) GetDimensions() int { return m.dimension }

type mockDocStoreProvider struct{}

func (m *mockDocStoreProvider) GetStore(opts StoreOpts) DocumentStore { return nil }
func (m *mockDocStoreProvider) GetRetriever(store DocumentStore, topK int) Retriever { return nil }

type mockEngine struct{}

func (m *mockEngine) ExecuteSQL(ctx context.Context, sql string, opts EngineOpts) (*EngineResult, error) {
	return &EngineResult{Success: true}, nil
}

func TestProviderInterfaces(t *testing.T) {
	var _ LLMProvider = &mockLLMProvider{}
	var _ Generator = &mockGenerator{}
	var _ EmbedderProvider = &mockEmbedderProvider{}
	var _ DocStoreProvider = &mockDocStoreProvider{}
	var _ Engine = &mockEngine{}
}

func TestGeneratorOpts(t *testing.T) {
	opts := GeneratorOpts{
		SystemPrompt:     "test prompt",
		GenerationKwargs: map[string]any{"temperature": 0},
	}
	if opts.SystemPrompt != "test prompt" {
		t.Fatalf("expected 'test prompt', got %s", opts.SystemPrompt)
	}
}

func TestGenerateResult(t *testing.T) {
	result := &GenerateResult{
		Replies: []string{"reply1", "reply2"},
		Meta:    []map[string]any{{"key": "val"}},
	}
	if len(result.Replies) != 2 {
		t.Fatalf("expected 2 replies, got %d", len(result.Replies))
	}
}

func TestEmbedResult(t *testing.T) {
	result := &EmbedResult{
		Embedding: []float32{0.1, 0.2, 0.3},
		Meta:      map[string]any{"model": "test"},
	}
	if len(result.Embedding) != 3 {
		t.Fatalf("expected 3 dims, got %d", len(result.Embedding))
	}
}

func TestStoreOpts(t *testing.T) {
	opts := StoreOpts{DatasetName: "test_collection", RecreateIndex: true}
	if opts.DatasetName != "test_collection" {
		t.Fatalf("expected test_collection, got %s", opts.DatasetName)
	}
}
