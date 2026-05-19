package generation

import (
	"context"
	"fmt"

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
	builder, _ := common.NewPromptBuilder("SQL: {{.sql}}\nDatabase Schema:\n{{range .documents}}    {{.}}\n{{end}}\nUser's input: {{.query}}")
	prompt, _ := builder.Build(map[string]any{"sql": req.SQL, "documents": req.Documents, "query": req.Query})
	gen, _ := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     "Expand the SQL by adding more columns or keywords.",
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	result, _ := gen.Run(ctx, prompt)
	if len(result.Replies) == 0 {
		return &common.SQLGenResult{}, nil
	}
	return p.postProc.Run(ctx, result.Replies[0], req.ProjectID)
}
