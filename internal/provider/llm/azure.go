package llm

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/odysseythink/pantheon/providers/azure"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

func init() {
	provider.RegisterLLM("azure_openai_llm", func(cfg map[string]any) (core.LLMProvider, error) {
		apiKey, _ := cfg["api_key"].(string)
		apiBase, _ := cfg["api_base"].(string)
		apiVersion, _ := cfg["api_version"].(string)
		model, _ := cfg["model"].(string)
		if model == "" {
			model = "gpt-4-turbo"
		}

		kwargs, _ := cfg["kwargs"].(map[string]any)
		if kwargs == nil {
			kwargs = map[string]any{
				"temperature":     0,
				"n":               1,
				"max_tokens":      1000,
				"response_format": map[string]any{"type": "json_object"},
			}
		}

		resourceName, deployment := "", ""
		if apiBase != "" {
			resourceName, deployment = extractAzureInfo(apiBase)
		}
		if resourceName == "" {
			resourceName, _ = cfg["resource_name"].(string)
		}
		if deployment == "" {
			deployment, _ = cfg["deployment"].(string)
		}
		if resourceName == "" || deployment == "" {
			return nil, fmt.Errorf("azure_openai_llm: need api_base or resource_name+deployment")
		}

		var opts []azure.Option
		if apiBase != "" {
			opts = append(opts, azure.WithBaseURL(apiBase))
		}
		if apiVersion != "" {
			opts = append(opts, azure.WithAPIVersion(apiVersion))
		}

		pantheonProvider, err := azure.New(apiKey, resourceName, deployment, opts...)
		if err != nil {
			return nil, err
		}
		return NewPantheonLLMProvider(pantheonProvider, model, kwargs), nil
	})
}

// extractAzureInfo parses https://{resource}.openai.azure.com/openai/deployments/{deployment}
func extractAzureInfo(apiBase string) (resourceName, deployment string) {
	u, err := url.Parse(apiBase)
	if err != nil {
		return "", ""
	}
	parts := strings.Split(u.Path, "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "deployments" && i+1 < len(parts) {
			deployment = parts[i+1]
			break
		}
	}
	hostParts := strings.Split(u.Hostname(), ".")
	if len(hostParts) > 0 {
		resourceName = hostParts[0]
	}
	return resourceName, deployment
}
