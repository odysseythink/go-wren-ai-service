package generation

import (
	"context"
	"fmt"
	"time"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
)

// SQLExpansion expands SQL with additional context.
type SQLExpansion struct {
	components core.PipelineComponent
	postProc   *common.SQLGenPostProcessor
}

// NewSQLExpansion creates a new SQL expansion pipeline.
func NewSQLExpansion(components core.PipelineComponent) *SQLExpansion {
	return &SQLExpansion{
		components: components,
		postProc:   common.NewSQLGenPostProcessor(components.Engine),
	}
}

// SQLExpansionRequest is the input.
type SQLExpansionRequest struct {
	SQL       string
	Documents []string
	Query     string
	ProjectID string
}

// Run executes SQL expansion.
func (p *SQLExpansion) Run(ctx context.Context, input any) (any, error) {
	req, ok := input.(*SQLExpansionRequest)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}
	builder, _ := common.NewPromptBuilder(sqlExpansionUserPrompt)
	prompt, _ := builder.Build(map[string]any{
		"sql":         req.SQL,
		"documents":   req.Documents,
		"query":       req.Query,
		"current_time": time.Now().Format("2006-01-02 15:04:05"),
	})
	gen, _ := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     sqlExpansionSystemPrompt,
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	result, _ := gen.Run(ctx, prompt)
	if len(result.Replies) == 0 {
		return &common.SQLGenResult{}, nil
	}
	return p.postProc.Run(ctx, result.Replies[0], req.ProjectID)
}

const sqlExpansionSystemPrompt = `
### TASK ###
You are a great data analyst. You are now given a task to expand original SQL by adding more columns or add more keywords such as DISTINCT.

### INSTRUCTIONS ###
- Columns are given from the user's input
- Columns to be added must belong to the given database schema; if no such column exists, keep SQL_QUERY_STRING empty

### OUTPUT FORMAT ###
Please return the result in the following JSON format:

{
    "results": [
        {"sql": <SQL_QUERY_STRING>}
    ]
}
`

const sqlExpansionUserPrompt = `
SQL: {{.sql}}

Database Schema:
{{range .documents}}
    {{.}}
{{end}}

User's input: {{.query}}
Current Time: {{.current_time}}
`
