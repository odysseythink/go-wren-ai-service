package generation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
)

// RelationshipRecommendation suggests relationships between models.
type RelationshipRecommendation struct {
	components core.PipelineComponent
}

// NewRelationshipRecommendation creates a new relationship recommendation pipeline.
func NewRelationshipRecommendation(components core.PipelineComponent) *RelationshipRecommendation {
	return &RelationshipRecommendation{components: components}
}

// RelationshipRecRequest is the input.
type RelationshipRecRequest struct {
	MDL string
}

// Run executes relationship recommendation.
func (p *RelationshipRecommendation) Run(ctx context.Context, input any) (any, error) {
	req, ok := input.(*RelationshipRecRequest)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}
	var mdlObj map[string]any
	json.Unmarshal([]byte(req.MDL), &mdlObj)
	builder, _ := common.NewPromptBuilder("Here is my data model's relationship specification:\n{{.models}}\n**Please review these models and provide recommendations...**")
	prompt, _ := builder.Build(map[string]any{"models": mdlObj["models"]})
	gen, _ := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     "Analyze models and suggest relationships with name, models, joinType, condition, reason.",
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	result, _ := gen.Run(ctx, prompt)
	if len(result.Replies) == 0 {
		return nil, fmt.Errorf("no reply")
	}
	return map[string]any{"validated": result.Replies[0]}, nil
}
