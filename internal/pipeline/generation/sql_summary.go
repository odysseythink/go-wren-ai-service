package generation

import (
	"context"
	"fmt"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
)

// SQLSummary summarizes SQL queries.
type SQLSummary struct {
	components core.PipelineComponent
}

// NewSQLSummary creates a new SQL summary pipeline.
func NewSQLSummary(components core.PipelineComponent) *SQLSummary {
	return &SQLSummary{components: components}
}

// SQLSummaryRequest is the input.
type SQLSummaryRequest struct {
	Query    string
	SQLs     []string
	Language string
}

// Run executes SQL summarization.
func (p *SQLSummary) Run(ctx context.Context, input any) (any, error) {
	req, ok := input.(*SQLSummaryRequest)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}
	builder, _ := common.NewPromptBuilder("User's Question: {{.query}}\nSQLs: {{.sqls}}\nLanguage: {{.language}}")
	prompt, _ := builder.Build(map[string]any{
		"query":    req.Query,
		"sqls":     req.SQLs,
		"language": req.Language,
	})
	gen, _ := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     "Summarize each SQL query in 10-20 words.",
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	result, _ := gen.Run(ctx, prompt)
	if len(result.Replies) == 0 {
		return &common.SQLSummaryResult{}, nil
	}
	postProc := &common.SQLSummaryPostProcessor{}
	return postProc.Run(result.Replies[0], req.SQLs)
}
