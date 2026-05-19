package generation

import (
	"context"
	"fmt"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
)

// SQLBreakdown decomposes SQL into CTE steps.
type SQLBreakdown struct {
	components core.PipelineComponent
	postProc   *common.SQLBreakdownGenPostProcessor
}

// NewSQLBreakdown creates a new SQL breakdown pipeline.
func NewSQLBreakdown(components core.PipelineComponent) *SQLBreakdown {
	return &SQLBreakdown{
		components: components,
		postProc:   common.NewSQLBreakdownGenPostProcessor(components.Engine),
	}
}

// SQLBreakdownRequest is the input.
type SQLBreakdownRequest struct {
	Query     string
	SQL       string
	Language  string
	ProjectID string
}

// Run executes SQL breakdown.
func (p *SQLBreakdown) Run(ctx context.Context, input any) (any, error) {
	req, ok := input.(*SQLBreakdownRequest)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}
	builder, _ := common.NewPromptBuilder("### INPUT ###\nUser's Question: {{.query}}\nSQL query: {{.sql}}\nLanguage: {{.language}}")
	prompt, _ := builder.Build(map[string]any{
		"query":    req.Query,
		"sql":      req.SQL,
		"language": req.Language,
	})
	gen, _ := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     "Break down the SQL query into 1-10 CTE steps. Each step should have sql, summary, and cte_name.",
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	result, _ := gen.Run(ctx, prompt)
	if len(result.Replies) == 0 {
		return &common.SQLBreakdownResult{}, nil
	}
	return p.postProc.Run(ctx, result.Replies[0], req.ProjectID)
}
