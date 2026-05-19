package generation

import (
	"context"
	"fmt"
	"time"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/model"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
)

// FollowUpSQLGeneration generates SQL for follow-up questions.
type FollowUpSQLGeneration struct {
	components core.PipelineComponent
	postProc   *common.SQLGenPostProcessor
}

// NewFollowUpSQLGeneration creates a new follow-up SQL generation pipeline.
func NewFollowUpSQLGeneration(components core.PipelineComponent) *FollowUpSQLGeneration {
	return &FollowUpSQLGeneration{
		components: components,
		postProc:   common.NewSQLGenPostProcessor(components.Engine),
	}
}

// FollowUpSQLRequest is the input.
type FollowUpSQLRequest struct {
	Query        string
	Documents    []string
	History      *model.AskHistory
	Instructions string
	ProjectID    string
}

// Run executes follow-up SQL generation.
func (p *FollowUpSQLGeneration) Run(ctx context.Context, input any) (any, error) {
	req, ok := input.(*FollowUpSQLRequest)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}
	builder, _ := common.NewPromptBuilder(followupSQLUserPrompt)
	prompt, _ := builder.Build(map[string]any{
		"query":        req.Query,
		"documents":    req.Documents,
		"history":      req.History,
		"alert":        common.TEXT_TO_SQL_RULES,
		"instructions": req.Instructions,
		"current_time": time.Now().Format("2006-01-02 15:04:05"),
	})
	gen, _ := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     sqlGenerationSystemPrompt,
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	result, _ := gen.Run(ctx, prompt)
	if len(result.Replies) == 0 {
		return &common.SQLGenResult{}, nil
	}
	return p.postProc.Run(ctx, result.Replies[0], req.ProjectID)
}

const followupSQLUserPrompt = `### TASK ###
Given user's follow-up question and previous SQL query and summary, generate at most 3 SQL queries.
### DATABASE SCHEMA ###
{{range .documents}}
    {{.}}
{{end}}
{{.alert}}
### QUESTION ###
Previous SQL Summary: {{.history.Summary}}
Previous SQL Query: {{.history.SQL}}
User's Follow-up Question: {{.query}}
Current Time: {{.current_time}}
{{if .instructions}}
Instructions: {{.instructions}}
{{end}}
`
