package llm

import (
	"testing"

	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

func TestAzureOpenAILLMFactory(t *testing.T) {
	f, ok := provider.GetLLMFactory("azure_openai_llm")
	if !ok {
		t.Fatal("azure_openai_llm factory not registered")
	}

	p, err := f(map[string]any{
		"api_key":    "test-key",
		"api_base":   "https://my-resource.openai.azure.com/openai/deployments/my-deployment",
		"api_version": "2024-06-01",
		"model":      "gpt-4o",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetModel() != "gpt-4o" {
		t.Fatalf("expected model gpt-4o, got %s", p.GetModel())
	}
}

func TestExtractAzureInfo(t *testing.T) {
	rn, dep := extractAzureInfo("https://my-resource.openai.azure.com/openai/deployments/my-deployment")
	if rn != "my-resource" {
		t.Fatalf("expected resource my-resource, got %s", rn)
	}
	if dep != "my-deployment" {
		t.Fatalf("expected deployment my-deployment, got %s", dep)
	}
}
