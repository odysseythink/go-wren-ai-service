package llm

import (
	openai "github.com/odysseythink/pantheon/providers/openai"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

func init() {
	provider.RegisterLLM("openai_llm", func(cfg map[string]any) (core.LLMProvider, error) {
		apiKey, _ := cfg["api_key"].(string)
		apiBase, _ := cfg["api_base"].(string)
		model, _ := cfg["model"].(string)
		if model == "" {
			model = "gpt-4o-mini"
		}

		kwargs, _ := cfg["kwargs"].(map[string]any)
		if kwargs == nil {
			kwargs = map[string]any{
				"temperature":     0,
				"n":               1,
				"max_tokens":      4096,
				"response_format": map[string]any{"type": "json_object"},
			}
		}

		var opts []openai.Option
		if apiBase != "" {
			opts = append(opts, openai.WithBaseURL(apiBase))
		}

		pantheonProvider, err := openai.New(apiKey, opts...)
		if err != nil {
			return nil, err
		}
		return NewPantheonLLMProvider(pantheonProvider, model, kwargs), nil
	})
}
