package embedder

import (
	"fmt"

	"github.com/odysseythink/pantheon/extensions/embed"
	openai "github.com/odysseythink/pantheon/providers/openai"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

func init() {
	provider.RegisterEmbedder("openai_embedder", func(cfg map[string]any) (core.EmbedderProvider, error) {
		apiKey, _ := cfg["api_key"].(string)
		apiBase, _ := cfg["api_base"].(string)
		model, _ := cfg["model"].(string)
		if model == "" {
			model = "text-embedding-3-large"
		}
		dimension := 3072
		if d, ok := cfg["dimension"].(int); ok {
			dimension = d
		}
		if d, ok := cfg["dimension"].(float64); ok {
			dimension = int(d)
		}

		var opts []openai.Option
		if apiBase != "" {
			opts = append(opts, openai.WithBaseURL(apiBase))
		}

		pantheonProvider, err := openai.New(apiKey, opts...)
		if err != nil {
			return nil, err
		}
		embedProvider, ok := pantheonProvider.(embed.Provider)
		if !ok {
			return nil, fmt.Errorf("openai provider does not support embedding")
		}
		return NewPantheonEmbedderProvider(embedProvider, model, dimension), nil
	})
}
