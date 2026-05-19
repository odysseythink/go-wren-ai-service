package generation

import (
	"context"
	"fmt"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
)

// SQLCorrection fixes invalid SQL queries.
type SQLCorrection struct {
	components core.PipelineComponent
	postProc   *common.SQLGenPostProcessor
}

// NewSQLCorrection creates a new SQL correction pipeline.
func NewSQLCorrection(components core.PipelineComponent) *SQLCorrection {
	return &SQLCorrection{
		components: components,
		postProc:   common.NewSQLGenPostProcessor(components.Engine),
	}
}

// SQLCorrectionRequest is the input.
type SQLCorrectionRequest struct {
	Documents                  []string
	InvalidGenerationResults   []map[string]any
	ProjectID                  string
}

// Run executes SQL correction.
func (p *SQLCorrection) Run(ctx context.Context, input any) (any, error) {
	req, ok := input.(*SQLCorrectionRequest)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}
	builder, _ := common.NewPromptBuilder(sqlCorrectionUserPrompt)
	prompt, _ := builder.Build(map[string]any{
		"documents":                    req.Documents,
		"invalid_generation_results":   req.InvalidGenerationResults,
		"alert":                        common.TEXT_TO_SQL_RULES,
	})
	gen, _ := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     "Correct the invalid SQL queries.",
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	result, _ := gen.Run(ctx, prompt)
	if len(result.Replies) == 0 {
		return &common.SQLGenResult{}, nil
	}
	return p.postProc.Run(ctx, result.Replies[0], req.ProjectID)
}

const sqlCorrectionUserPrompt = `### DATABASE SCHEMA ###
{{range .documents}}
    {{.}}
{{end}}
### FINAL ANSWER FORMAT ###
{ "results": [{"sql": "<CORRECTED>", "summary": "<ORIGINAL_SUMMARY>"}] }
{{.alert}}
### QUESTION ###
{{range .invalid_generation_results}}
    sql: {{.sql}}
    summary: {{.summary}}
    error: {{.error}}
{{end}}
`
