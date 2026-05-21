package provider

import (
	"context"
	"testing"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
)

func TestRegisterLLM(t *testing.T) {
	RegisterLLM("test_llm", func(cfg map[string]any) (core.LLMProvider, error) {
		return &mockLLM{}, nil
	})

	f, ok := GetLLMFactory("test_llm")
	if !ok {
		t.Fatal("expected factory to be registered")
	}

	p, err := f(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

type mockLLM struct{}

func (m *mockLLM) GetGenerator(ctx context.Context, opts core.GeneratorOpts) (core.Generator, error) {
	return nil, nil
}
func (m *mockLLM) GetModel() string                 { return "" }
func (m *mockLLM) GetModelKwargs() map[string]any   { return nil }

func TestRegisterEmbedder(t *testing.T) {
	RegisterEmbedder("test_embedder", func(cfg map[string]any) (core.EmbedderProvider, error) {
		return &mockEmbedder{}, nil
	})

	f, ok := GetEmbedderFactory("test_embedder")
	if !ok {
		t.Fatal("expected factory to be registered")
	}

	p, err := f(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

type mockEmbedder struct{}

func (m *mockEmbedder) GetTextEmbedder(ctx context.Context) (core.TextEmbedder, error)     { return nil, nil }
func (m *mockEmbedder) GetDocumentEmbedder(ctx context.Context) (core.DocumentEmbedder, error) { return nil, nil }
func (m *mockEmbedder) GetModel() string                                                     { return "" }
func (m *mockEmbedder) GetDimensions() int                                                   { return 0 }

func TestRegisterEngine(t *testing.T) {
	RegisterEngine("test_engine", func(cfg map[string]any) (core.Engine, error) {
		return &mockEngine{}, nil
	})

	f, ok := GetEngineFactory("test_engine")
	if !ok {
		t.Fatal("expected factory to be registered")
	}

	p, err := f(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

type mockEngine struct{}

func (m *mockEngine) ExecuteSQL(ctx context.Context, sql string, opts core.EngineOpts) (*core.EngineResult, error) {
	return nil, nil
}

func TestRegisterDocStore(t *testing.T) {
	RegisterDocStore("test_docstore", func(cfg map[string]any) (core.DocStoreProvider, error) {
		return &mockDocStore{}, nil
	})

	f, ok := GetDocStoreFactory("test_docstore")
	if !ok {
		t.Fatal("expected factory to be registered")
	}

	p, err := f(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

type mockDocStore struct{}

func (m *mockDocStore) GetStore(opts core.StoreOpts) core.DocumentStore { return nil }
func (m *mockDocStore) GetRetriever(store core.DocumentStore, topK int) core.Retriever {
	return nil
}
