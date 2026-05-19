package generation

import (
	"context"
	"fmt"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
)

// SQLRegeneration regenerates SQL from breakdown with corrections.
type SQLRegeneration struct {
	components core.PipelineComponent
	postProc   *common.SQLBreakdownGenPostProcessor
}

// NewSQLRegeneration creates a new SQL regeneration pipeline.
func NewSQLRegeneration(components core.PipelineComponent) *SQLRegeneration {
	return &SQLRegeneration{
		components: components,
		postProc:   common.NewSQLBreakdownGenPostProcessor(components.Engine),
	}
}

// SQLRegenerationRequest is the input.
type SQLRegenerationRequest struct {
	Description string
	Steps       []map[string]any
	ProjectID   string
}

// Run executes SQL regeneration.
func (p *SQLRegeneration) Run(ctx context.Context, input any) (any, error) {
	req, ok := input.(*SQLRegenerationRequest)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}
	builder, _ := common.NewPromptBuilder("inputs: {{.results}}")
	prompt, _ := builder.Build(map[string]any{"results": map[string]any{"description": req.Description, "steps": req.Steps}})
	gen, _ := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     "Given user corrections, regenerate the SQL query.",
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	result, _ := gen.Run(ctx, prompt)
	if len(result.Replies) == 0 {
		return &common.SQLBreakdownResult{}, nil
	}
	return p.postProc.Run(ctx, result.Replies[0], req.ProjectID)
}
