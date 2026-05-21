package generation

import (
	"context"
	"fmt"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
)

// SQLBreakdown decomposes SQL into CTE steps.
type SQLBreakdown struct {
	components core.PipelineComponent
	postProc   *common.SQLBreakdownGenPostProcessor
}

// NewSQLBreakdown creates a new SQL breakdown pipeline.
func NewSQLBreakdown(components core.PipelineComponent) *SQLBreakdown {
	return &SQLBreakdown{
		components: components,
		postProc:   common.NewSQLBreakdownGenPostProcessor(components.Engine),
	}
}

// SQLBreakdownRequest is the input.
type SQLBreakdownRequest struct {
	Query     string
	SQL       string
	Language  string
	ProjectID string
}

// Run executes SQL breakdown.
func (p *SQLBreakdown) Run(ctx context.Context, input any) (any, error) {
	req, ok := input.(*SQLBreakdownRequest)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}
	builder, _ := common.NewPromptBuilder(sqlBreakdownUserPrompt)
	prompt, _ := builder.Build(map[string]any{
		"query":    req.Query,
		"sql":      req.SQL,
		"language": req.Language,
	})
	gen, _ := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     sqlBreakdownSystemPrompt,
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	result, _ := gen.Run(ctx, prompt)
	if len(result.Replies) == 0 {
		return &common.SQLBreakdownResult{}, nil
	}
	return p.postProc.Run(ctx, result.Replies[0], req.ProjectID)
}

const sqlBreakdownSystemPrompt = `
### TASK ###
You are a Trino SQL expert with exceptional logical thinking skills.
You are going to break a complex SQL query into 1 to 10 steps to make it easier to understand for end users.
Each step should have a SQL query part, a summary explaining the purpose of that query, and a CTE name to link the queries.
Also, you need to give a short description describing the purpose of the original SQL query.
Description and summary in each step MUST BE in the same language as user specified.

### SQL QUERY BREAKDOWN INSTRUCTIONS ###
- YOU MUST BREAK DOWN any SQL query into small steps if there is JOIN operations or sub-queries.
- ONLY USE the tables and columns mentioned in the original sql query.
- ONLY CHOOSE columns belong to the tables mentioned in the database schema.
- ALWAYS USE alias for tables and referenced CTEs.
- ALWAYS SHOW alias for columns and tables such as SELECT [column_name] AS [alias_column_name].
- MUST USE alias from the original SQL query.

### SUMMARY AND DESCRIPTION INSTRUCTIONS ###
- SUMMARY AND DESCRIPTION MUST BE the same language as the user speficied.
- SUMMARY AND DESCRIPTION MUST BE human-readable and easy to understand.
- SUMMARY AND DESCRIPTION MUST BE concise and to the point.

### EXAMPLES ###
Example 1:
Original SQL Query:

SELECT product_id, SUM(sales) AS total_sales
FROM sales_data
GROUP BY product_id
HAVING SUM(sales) > 10000;

Results:

- Description: The breakdown simplifies the process of aggregating sales data by product and filtering for top-selling products.
- Step 1:
    - sql: SELECT product_id, sales FROM sales_data
    - summary: Selects product IDs and their corresponding sales from the sales_data table.
    - cte_name: basic_sales_data
- Step 2:
    - sql: SELECT product_id, SUM(sales) AS total_sales FROM basic_sales_data GROUP BY product_id
    - summary: Aggregates sales by product, summing up sales for each product ID.
    - cte_name: aggregated_sales
- Step 3:
    - sql: SELECT product_id, total_sales FROM aggregated_sales WHERE total_sales > 10000
    - summary: Filters the aggregated sales data to only include products whose total sales exceed 10,000.
    - cte_name: <empty_string>

Example 2:
Original SQL Query:

SELECT product_id FROM sales_data

Results:

- Description: The breakdown simplifies the process of selecting product IDs from the sales_data table.
- Step 1:
    - sql: SELECT product_id FROM sales_data
    - summary: Selects product IDs from the sales_data table.
    - cte_name: <empty_string>

### FINAL ANSWER FORMAT ###
The final answer must be a valid JSON format as following:

{
    "description": <SHORT_SQL_QUERY_DESCRIPTION_STRING>,
    "steps": [
        {"sql": <SQL_QUERY_STRING_1>, "summary": <SUMMARY_STRING_1>, "cte_name": <CTE_NAME_STRING_1>},
        ...
    ]
}
`

const sqlBreakdownUserPrompt = `
### INPUT ###
User's Question: {{.query}}
SQL query: {{.sql}}
Language: {{.language}}

Let's think step by step.
`
