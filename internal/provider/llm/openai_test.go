package llm

import (
	"testing"

	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

func TestOpenAILLMFactory(t *testing.T) {
	f, ok := provider.GetLLMFactory("openai_llm")
	if !ok {
		t.Fatal("openai_llm factory not registered")
	}

	p, err := f(map[string]any{
		"api_key": "test-key",
		"model":   "gpt-4o",
		"kwargs": map[string]any{
			"temperature": 0.5,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetModel() != "gpt-4o" {
		t.Fatalf("expected model gpt-4o, got %s", p.GetModel())
	}
}
