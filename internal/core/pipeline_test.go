package core

import (
	"context"
	"testing"
)

type mockPipeline struct{}

func (m *mockPipeline) Run(ctx context.Context, input any) (any, error) {
	return "result", nil
}

func TestPipelineInterface(t *testing.T) {
	var p Pipeline = &mockPipeline{}
	ctx := context.Background()
	result, err := p.Run(ctx, "input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "result" {
		t.Fatalf("expected 'result', got %v", result)
	}
}

func TestPipelineComponent_Fields(t *testing.T) {
	pc := PipelineComponent{}
	if pc.LLMProvider != nil {
		t.Fatal("expected nil LLMProvider")
	}
	if pc.EmbedderProvider != nil {
		t.Fatal("expected nil EmbedderProvider")
	}
	if pc.DocStoreProvider != nil {
		t.Fatal("expected nil DocStoreProvider")
	}
	if pc.Engine != nil {
		t.Fatal("expected nil Engine")
	}
}
