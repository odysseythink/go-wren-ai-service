package generation

import (
	"context"
	"fmt"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
)

// SQLExplanation explains SQL analysis results.
type SQLExplanation struct {
	components core.PipelineComponent
}

// NewSQLExplanation creates a new SQL explanation pipeline.
func NewSQLExplanation(components core.PipelineComponent) *SQLExplanation {
	return &SQLExplanation{components: components}
}

// SQLExplanationRequest is the input.
type SQLExplanationRequest struct {
	Question         string
	SQL              string
	SQLSummary       string
	SQLAnalysisResult map[string]any
}

// Run executes SQL explanation.
func (p *SQLExplanation) Run(ctx context.Context, input any) (any, error) {
	req, ok := input.(*SQLExplanationRequest)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}
	builder, _ := common.NewPromptBuilder("Question: {{.question}}\nSQL query: {{.sql}}\nSQL query summary: {{.sql_summary}}\nSQL query analysis: {{.sql_analysis_result}}")
	prompt, _ := builder.Build(map[string]any{
		"question":             req.Question,
		"sql":                  req.SQL,
		"sql_summary":          req.SQLSummary,
		"sql_analysis_result":  req.SQLAnalysisResult,
	})
	gen, _ := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     "Explain each SQL analysis result in layman terms (<20 words).",
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	result, _ := gen.Run(ctx, prompt)
	if len(result.Replies) == 0 {
		return nil, fmt.Errorf("no reply")
	}
	return map[string]any{"explanations": result.Replies[0]}, nil
}
