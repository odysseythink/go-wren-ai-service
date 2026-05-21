package llm

import (
	"strings"

	"github.com/odysseythink/pantheon/providers/ollama"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

func init() {
	provider.RegisterLLM("ollama_llm", func(cfg map[string]any) (core.LLMProvider, error) {
		url, _ := cfg["url"].(string)
		if url == "" {
			url = "http://localhost:11434"
		}
		model, _ := cfg["model"].(string)
		if model == "" {
			model = "gemma2:9b"
		}

		kwargs, _ := cfg["kwargs"].(map[string]any)
		if kwargs == nil {
			kwargs = map[string]any{"temperature": 0}
		}

		if !strings.HasSuffix(url, "/v1") {
			url = strings.TrimSuffix(url, "/") + "/v1"
		}

		pantheonProvider, err := ollama.New("", ollama.WithBaseURL(url))
		if err != nil {
			return nil, err
		}
		return NewPantheonLLMProvider(pantheonProvider, model, kwargs), nil
	})
}
