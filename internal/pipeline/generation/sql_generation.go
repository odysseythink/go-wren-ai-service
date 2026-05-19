package generation

import (
	"context"
	"fmt"
	"time"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
)

// SQLGeneration generates SQL queries from natural language.
type SQLGeneration struct {
	components core.PipelineComponent
	postProc   *common.SQLGenPostProcessor
}

// NewSQLGeneration creates a new SQL generation pipeline.
func NewSQLGeneration(components core.PipelineComponent) *SQLGeneration {
	return &SQLGeneration{
		components: components,
		postProc:   common.NewSQLGenPostProcessor(components.Engine),
	}
}

// SQLGenerationRequest is the input.
type SQLGenerationRequest struct {
	Query        string
	Documents    []string
	Exclude      []string
	Instructions string
	Samples      []map[string]string
	ProjectID    string
}

// SQLGenerationResult is the output.
type SQLGenerationResult struct {
	ValidGenerationResults   []map[string]any
	InvalidGenerationResults []map[string]any
}

// Run executes SQL generation.
func (p *SQLGeneration) Run(ctx context.Context, input any) (any, error) {
	req, ok := input.(*SQLGenerationRequest)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	prompt, err := buildSQLGenerationPrompt(req)
	if err != nil {
		return nil, err
	}

	gen, err := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     sqlGenerationSystemPrompt,
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	if err != nil {
		return nil, err
	}

	result, err := gen.Run(ctx, prompt)
	if err != nil {
		return nil, err
	}
	if len(result.Replies) == 0 {
		return &SQLGenerationResult{}, nil
	}

	return p.postProc.Run(ctx, result.Replies[0], req.ProjectID)
}

func buildSQLGenerationPrompt(req *SQLGenerationRequest) (string, error) {
	builder, err := common.NewPromptBuilder(sqlGenerationUserPrompt)
	if err != nil {
		return "", err
	}
	return builder.Build(map[string]any{
		"query":        req.Query,
		"documents":    req.Documents,
		"exclude":      req.Exclude,
		"alert":        common.TEXT_TO_SQL_RULES,
		"instructions": req.Instructions,
		"samples":      req.Samples,
		"current_time": time.Now().Format("2006-01-02 15:04:05"),
	})
}

const sqlGenerationSystemPrompt = `You are an expert SQL generator. Given a user's question and database schema, generate up to 3 SQL queries that answer the question. Return ONLY a JSON object with a "results" array containing {"sql": "..."} objects.`

const sqlGenerationUserPrompt = `### TASK ###
Given a user query that is ambiguous in nature, generate three SQL statements that could potentially answer the question.
### DATABASE SCHEMA ###
{{range .documents}}
    {{.}}
{{end}}
{{if .exclude}}
### EXCLUDED STATEMENTS ###
{{range .exclude}}
    {{.}}
{{end}}
{{end}}
{{.alert}}
{{if .instructions}}
{{.instructions}}
{{end}}
### FINAL ANSWER FORMAT ###
{ "results": [{"sql": "..."}, {"sql": "..."}, {"sql": "..."}] }
{{if .samples}}
### SAMPLES ###
{{range .samples}}
Question: {{.question}}
SQL: {{.sql}}
{{end}}
{{end}}
### QUESTION ###
User's Question: {{.query}}
Current Time: {{.current_time}}
`
