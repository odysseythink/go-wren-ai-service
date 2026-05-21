package embedder

import (
	"fmt"
	"strings"

	"github.com/odysseythink/pantheon/extensions/embed"
	openai "github.com/odysseythink/pantheon/providers/openai"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

func init() {
	provider.RegisterEmbedder("ollama_embedder", func(cfg map[string]any) (core.EmbedderProvider, error) {
		url, _ := cfg["url"].(string)
		if url == "" {
			url = "http://localhost:11434"
		}
		model, _ := cfg["model"].(string)
		if model == "" {
			model = "nomic-embed-text:latest"
		}
		dimension := 768
		if d, ok := cfg["dimension"].(int); ok {
			dimension = d
		}
		if d, ok := cfg["dimension"].(float64); ok {
			dimension = int(d)
		}

		if !strings.HasSuffix(url, "/v1") {
			url = strings.TrimSuffix(url, "/") + "/v1"
		}

		pantheonProvider, err := openai.New("", openai.WithBaseURL(url))
		if err != nil {
			return nil, err
		}
		embedProvider, ok := pantheonProvider.(embed.Provider)
		if !ok {
			return nil, fmt.Errorf("ollama provider does not support embedding")
		}
		return NewPantheonEmbedderProvider(embedProvider, model, dimension), nil
	})
}
