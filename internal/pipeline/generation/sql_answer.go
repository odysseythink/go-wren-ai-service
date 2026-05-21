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
	builder, _ := common.NewPromptBuilder(sqlAnswerUserPrompt)
	prompt, _ := builder.Build(map[string]any{
		"query":       req.Query,
		"sql":         req.SQL,
		"sql_summary": req.SQLSummary,
		"sql_data":    req.SQLData,
	})
	gen, _ := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     sqlAnswerSystemPrompt,
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	result, _ := gen.Run(ctx, prompt)
	if len(result.Replies) == 0 {
		return nil, fmt.Errorf("no reply")
	}
	postProc := &common.SQLAnswerPostProcessor{}
	return postProc.Run(result.Replies[0])
}

const sqlAnswerSystemPrompt = `
### TASK ###
You are a data analyst that great at answering user's questions based on the data, sql and sql summary. Please answer the user's question in concise and clear manner.

### INSTRUCTIONS ###
1. Read the user's question and understand the user's intention.
2. Read the sql summary and understand the data.
3. Read the sql and understand the data.
4. Generate an answer in string format and a reasoning process in string format to the user's question based on the data, sql and sql summary.

### OUTPUT FORMAT ###
Return the output in the following JSON format:

{
    "reasoning": "<STRING>",
    "answer": "<STRING>"
}
`

const sqlAnswerUserPrompt = `
### Input ###
User's question: {{.query}}
SQL: {{.sql}}
SQL summary: {{.sql_summary}}
Data: {{.sql_data}}

Please think step by step and answer the user's question.
`
