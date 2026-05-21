package generation

import (
	"context"
	"fmt"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
)

// SQLRegeneration regenerates SQL from breakdown with corrections.
type SQLRegeneration struct {
	components core.PipelineComponent
	postProc   *common.SQLBreakdownGenPostProcessor
}

// NewSQLRegeneration creates a new SQL regeneration pipeline.
func NewSQLRegeneration(components core.PipelineComponent) *SQLRegeneration {
	return &SQLRegeneration{
		components: components,
		postProc:   common.NewSQLBreakdownGenPostProcessor(components.Engine),
	}
}

// SQLRegenerationRequest is the input.
type SQLRegenerationRequest struct {
	Description string
	Steps       []map[string]any
	ProjectID   string
}

// Run executes SQL regeneration.
func (p *SQLRegeneration) Run(ctx context.Context, input any) (any, error) {
	req, ok := input.(*SQLRegenerationRequest)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}
	builder, _ := common.NewPromptBuilder(sqlRegenerationUserPrompt)
	prompt, _ := builder.Build(map[string]any{"results": map[string]any{"description": req.Description, "steps": req.Steps}})
	gen, _ := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     sqlRegenerationSystemPrompt,
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	result, _ := gen.Run(ctx, prompt)
	if len(result.Replies) == 0 {
		return &common.SQLBreakdownResult{}, nil
	}
	return p.postProc.Run(ctx, result.Replies[0], req.ProjectID)
}

const sqlRegenerationSystemPrompt = `
### Instructions ###
- Given a list of user corrections, regenerate the corresponding SQL query.
- For each modified SQL query, update the corresponding SQL summary, CTE name.
- If subsequent steps are dependent on the corrected step, make sure to update the SQL query, SQL summary and CTE name in subsequent steps if needed.
- Regenerate the description after correcting all of the steps.

### INPUT STRUCTURE ###
{
    "description": "<original_description_string>",
    "steps": [
        {
            "summary": "<original_sql_summary_string>",
            "sql": "<original_sql_string>",
            "cte_name": "<original_cte_name_string>",
            "corrections": [
                {
                    "before": {
                        "type": "<filter/selectItems/relation/groupByKeys/sortings>",
                        "value": "<original_value_string>"
                    },
                    "after": {
                        "type": "<sql_expression/nl_expression>",
                        "value": "<new_value_string>"
                    }
                }
            ]
        }
    ]
}

### OUTPUT STRUCTURE ###
Generate modified results according to the following in JSON format:

{
    "description": "<modified_description_string>",
    "steps": [
        {
            "summary": "<modified_sql_summary_string>",
            "sql": "<modified_sql_string>",
            "cte_name": "<modified_cte_name_string>"
        }
    ]
}
`

const sqlRegenerationUserPrompt = `
inputs: {{.results}}

Let's think step by step.
`
