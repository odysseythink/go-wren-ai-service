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
	builder, _ := common.NewPromptBuilder(sqlSummaryUserPrompt)
	prompt, _ := builder.Build(map[string]any{
		"query":    req.Query,
		"sqls":     req.SQLs,
		"language": req.Language,
	})
	gen, _ := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     sqlSummarySystemPrompt,
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	result, _ := gen.Run(ctx, prompt)
	if len(result.Replies) == 0 {
		return &common.SQLSummaryResult{}, nil
	}
	postProc := &common.SQLSummaryPostProcessor{}
	return postProc.Run(result.Replies[0], req.SQLs)
}

const sqlSummarySystemPrompt = `
### TASK ###
You are a great data analyst. You are now given a task to summarize a list SQL queries in a human-readable format where each summary should be within 10-20 words.
You will be given a list of SQL queries and a user's question.

### INSTRUCTIONS ###
- SQL query summary must be within 10-20 words.
- SQL query summary must be human-readable and easy to understand.
- SQL query summary must be concise and to the point.
- SQL query summary must be in the same language user specified.

### OUTPUT FORMAT ###
Please return the result in the following JSON format:

{
    "sql_summary_results": [
        {"summary": <SQL_QUERY_SUMMARY_STRING_1>},
        {"summary": <SQL_QUERY_SUMMARY_STRING_2>},
        ...
    ]
}
`

const sqlSummaryUserPrompt = `
User's Question: {{.query}}
SQLs: {{.sqls}}
Language: {{.language}}

Please think step by step.
`
