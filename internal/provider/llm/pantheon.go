package llm

import (
	"context"
	"fmt"

	pantheoncore "github.com/odysseythink/pantheon/core"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
)

// PantheonLLMProvider wraps a Pantheon core.Provider to implement core.LLMProvider.
type PantheonLLMProvider struct {
	provider pantheoncore.Provider
	modelID  string
	kwargs   map[string]any
}

// NewPantheonLLMProvider creates a new LLM provider backed by Pantheon.
func NewPantheonLLMProvider(provider pantheoncore.Provider, modelID string, kwargs map[string]any) *PantheonLLMProvider {
	return &PantheonLLMProvider{
		provider: provider,
		modelID:  modelID,
		kwargs:   kwargs,
	}
}

// GetModel returns the model identifier.
func (p *PantheonLLMProvider) GetModel() string {
	return p.modelID
}

// GetModelKwargs returns the default generation kwargs.
func (p *PantheonLLMProvider) GetModelKwargs() map[string]any {
	return p.kwargs
}

// GetGenerator creates a Generator for the given options.
func (p *PantheonLLMProvider) GetGenerator(ctx context.Context, opts core.GeneratorOpts) (core.Generator, error) {
	model, err := p.provider.LanguageModel(ctx, p.modelID)
	if err != nil {
		return nil, fmt.Errorf("get language model: %w", err)
	}
	return &PantheonGenerator{
		model:        model,
		systemPrompt: opts.SystemPrompt,
		generationKwargs: opts.GenerationKwargs,
		modelKwargs:  p.kwargs,
	}, nil
}

// PantheonGenerator implements core.Generator using Pantheon's LanguageModel.
type PantheonGenerator struct {
	model            pantheoncore.LanguageModel
	systemPrompt     string
	generationKwargs map[string]any
	modelKwargs      map[string]any
}

// Run executes a single LLM call.
func (g *PantheonGenerator) Run(ctx context.Context, prompt string) (*core.GenerateResult, error) {
	req := &pantheoncore.Request{
		Messages:     []pantheoncore.Message{pantheoncore.NewTextMessage(pantheoncore.MESSAGE_ROLE_USER, prompt)},
		SystemPrompt: g.systemPrompt,
	}

	// Merge kwargs: modelKwargs (defaults) + generationKwargs (per-call overrides)
	kwargs := mergeKwargs(g.modelKwargs, g.generationKwargs)
	applyKwargs(req, kwargs)

	resp, err := g.model.Generate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}

	return &core.GenerateResult{
		Replies: []string{resp.Message.Text()},
		Meta:    []map[string]any{{"model": resp.Model, "usage": resp.Usage}},
	}, nil
}

func mergeKwargs(defaults, overrides map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range defaults {
		result[k] = v
	}
	for k, v := range overrides {
		result[k] = v
	}
	return result
}

func applyKwargs(req *pantheoncore.Request, kwargs map[string]any) {
	if v, ok := kwargs["temperature"]; ok {
		if f, ok := v.(float64); ok {
			req.Temperature = &f
		}
		if i, ok := v.(int); ok {
			f := float64(i)
			req.Temperature = &f
		}
	}
	if v, ok := kwargs["max_tokens"]; ok {
		if i, ok := v.(int); ok {
			req.MaxTokens = &i
		}
		if f, ok := v.(float64); ok {
			i := int(f)
			req.MaxTokens = &i
		}
	}
	if v, ok := kwargs["top_p"]; ok {
		if f, ok := v.(float64); ok {
			req.TopP = &f
		}
	}
	if v, ok := kwargs["stop"]; ok {
		if s, ok := v.(string); ok {
			req.StopSequences = []string{s}
		}
		if arr, ok := v.([]string); ok {
			req.StopSequences = arr
		}
	}
	if v, ok := kwargs["response_format"]; ok {
		if m, ok := v.(map[string]any); ok {
			if t, ok := m["type"].(string); ok {
				switch t {
				case "json_object":
					req.ResponseFormat = &pantheoncore.ResponseFormat{
						Type: pantheoncore.ResponseFormatTypeJSON,
					}
				case "json_schema":
					req.ResponseFormat = &pantheoncore.ResponseFormat{
						Type:       pantheoncore.ResponseFormatTypeJSONSchema,
						JSONSchema: nil, // would need schema conversion
					}
				}
			}
		}
	}
}
