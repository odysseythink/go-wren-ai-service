package generation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
)

// SemanticsDescription generates model and column descriptions.
type SemanticsDescription struct {
	components core.PipelineComponent
}

// NewSemanticsDescription creates a new semantics description pipeline.
func NewSemanticsDescription(components core.PipelineComponent) *SemanticsDescription {
	return &SemanticsDescription{components: components}
}

// SemanticsDescriptionRequest is the input.
type SemanticsDescriptionRequest struct {
	UserPrompt     string
	SelectedModels []string
	MDL            string
}

// Run executes semantics description generation.
func (p *SemanticsDescription) Run(ctx context.Context, input any) (any, error) {
	req, ok := input.(*SemanticsDescriptionRequest)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}
	var mdlObj map[string]any
	json.Unmarshal([]byte(req.MDL), &mdlObj)
	models, _ := mdlObj["models"].([]any)
	var picked []map[string]any
	for _, m := range models {
		modelMap, _ := m.(map[string]any)
		name, _ := modelMap["name"].(string)
		for _, sel := range req.SelectedModels {
			if sel == name {
				picked = append(picked, modelMap)
				break
			}
		}
	}

	builder, _ := common.NewPromptBuilder("### Input:\nUser's prompt: {{.user_prompt}}\nPicked models: {{.picked_models}}")
	prompt, _ := builder.Build(map[string]any{
		"user_prompt":    req.UserPrompt,
		"picked_models":  picked,
	})
	gen, _ := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     "Add description field inside properties for each model and each column. Preserve original format.",
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	result, _ := gen.Run(ctx, prompt)
	if len(result.Replies) == 0 {
		return nil, fmt.Errorf("no reply")
	}
	return map[string]any{"normalize": result.Replies[0]}, nil
}
