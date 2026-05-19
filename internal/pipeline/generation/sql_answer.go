package generation

import (
	"context"
	"fmt"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
)

// SQLAnswer generates natural language answers from SQL results.
type SQLAnswer struct {
	components core.PipelineComponent
}

// NewSQLAnswer creates a new SQL answer pipeline.
func NewSQLAnswer(components core.PipelineComponent) *SQLAnswer {
	return &SQLAnswer{components: components}
}

// SQLAnswerRequest is the input.
type SQLAnswerRequest struct {
	Query      string
	SQL        string
	SQLSummary string
	SQLData    any
}

// Run executes SQL answer generation.
func (p *SQLAnswer) Run(ctx context.Context, input any) (any, error) {
	req, ok := input.(*SQLAnswerRequest)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}
	builder, _ := common.NewPromptBuilder("User's question: {{.query}}\nSQL: {{.sql}}\nSQL summary: {{.sql_summary}}\nData: {{.sql_data}}")
	prompt, _ := builder.Build(map[string]any{
		"query":       req.Query,
		"sql":         req.SQL,
		"sql_summary": req.SQLSummary,
		"sql_data":    req.SQLData,
	})
	gen, _ := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     "Generate a natural language answer and reasoning based on the SQL and its result data.",
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	result, _ := gen.Run(ctx, prompt)
	if len(result.Replies) == 0 {
		return nil, fmt.Errorf("no reply")
	}
	postProc := &common.SQLAnswerPostProcessor{}
	return postProc.Run(result.Replies[0])
}
