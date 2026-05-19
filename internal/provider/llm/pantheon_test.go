package llm

import (
	"context"
	"testing"

	pantheoncore "github.com/odysseythink/pantheon/core"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
)

type mockPantheonProvider struct {
	model pantheoncore.LanguageModel
}

func (m *mockPantheonProvider) Name() string { return "mock" }
func (m *mockPantheonProvider) Models(ctx context.Context) ([]pantheoncore.Model, error) {
	return nil, nil
}
func (m *mockPantheonProvider) LanguageModel(ctx context.Context, modelID string) (pantheoncore.LanguageModel, error) {
	return m.model, nil
}

type mockLanguageModel struct {
	reply string
}

func (m *mockLanguageModel) Generate(ctx context.Context, req *pantheoncore.Request) (*pantheoncore.Response, error) {
	return &pantheoncore.Response{
		Message: pantheoncore.NewTextMessage(pantheoncore.MESSAGE_ROLE_ASSISTANT, m.reply),
	}, nil
}
func (m *mockLanguageModel) Stream(ctx context.Context, req *pantheoncore.Request) (pantheoncore.StreamResponse, error) {
	return nil, nil
}
func (m *mockLanguageModel) GenerateObject(ctx context.Context, req *pantheoncore.ObjectRequest) (*pantheoncore.ObjectResponse, error) {
	return nil, nil
}
func (m *mockLanguageModel) Provider() string { return "mock" }
func (m *mockLanguageModel) Model() string    { return "mock-model" }

func TestPantheonLLMProvider_GetGenerator(t *testing.T) {
	mockModel := &mockLanguageModel{reply: "SELECT 1"}
	mockProvider := &mockPantheonProvider{model: mockModel}

	llmProvider := NewPantheonLLMProvider(mockProvider, "gpt-4o", map[string]any{"temperature": 0.0})
	if llmProvider.GetModel() != "gpt-4o" {
		t.Fatalf("expected model gpt-4o, got %s", llmProvider.GetModel())
	}

	gen, err := llmProvider.GetGenerator(context.Background(), core.GeneratorOpts{
		SystemPrompt: "You are a SQL generator",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := gen.Run(context.Background(), "show me orders")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Replies) != 1 || result.Replies[0] != "SELECT 1" {
		t.Fatalf("unexpected replies: %v", result.Replies)
	}
}

func TestPantheonLLMProvider_GetModelKwargs(t *testing.T) {
	llmProvider := NewPantheonLLMProvider(nil, "gpt-4o", map[string]any{"temperature": 0.5})
	kwargs := llmProvider.GetModelKwargs()
	if kwargs["temperature"] != 0.5 {
		t.Fatalf("expected temperature 0.5, got %v", kwargs["temperature"])
	}
}
