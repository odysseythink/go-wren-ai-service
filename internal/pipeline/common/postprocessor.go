package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/pkg/sqlutil"
)

// SQLGenPostProcessor cleans LLM output, quotes SQLs, and validates via dry-run.
type SQLGenPostProcessor struct {
	engine core.Engine
}

// NewSQLGenPostProcessor creates a new SQL generation postprocessor.
func NewSQLGenPostProcessor(engine core.Engine) *SQLGenPostProcessor {
	return &SQLGenPostProcessor{engine: engine}
}

// Result holds valid and invalid generation results.
type SQLGenResult struct {
	ValidGenerationResults   []map[string]any
	InvalidGenerationResults []map[string]any
}

// Run parses the LLM reply and classifies each generated SQL.
func (p *SQLGenPostProcessor) Run(ctx context.Context, reply string, projectID string) (*SQLGenResult, error) {
	cleaned := sqlutil.CleanGenerationResult(reply)
	var parsed struct {
		Results []struct {
			SQL string `json:"sql"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return &SQLGenResult{}, nil
	}

	var valid, invalid []map[string]any
	for _, r := range parsed.Results {
		quoted, ok := sqlutil.AddQuotes(r.SQL)
		if !ok {
			invalid = append(invalid, map[string]any{
				"sql":   r.SQL,
				"type":  "ADD_QUOTES",
				"error": "add_quotes failed",
			})
			continue
		}
		result, err := p.engine.ExecuteSQL(ctx, quoted, core.EngineOpts{ProjectID: projectID, DryRun: true})
		if err != nil || !result.Success {
			errMsg := "dry-run failed"
			if result != nil && result.Error != "" {
				errMsg = result.Error
			}
			invalid = append(invalid, map[string]any{
				"sql":   quoted,
				"type":  "DRY_RUN",
				"error": errMsg,
			})
			continue
		}
		valid = append(valid, map[string]any{"sql": quoted})
	}
	return &SQLGenResult{ValidGenerationResults: valid, InvalidGenerationResults: invalid}, nil
}

// SQLBreakdownResult holds a SQL breakdown output.
type SQLBreakdownResult struct {
	Description string                 `json:"description"`
	Steps       []map[string]any       `json:"steps"`
}

// SQLBreakdownGenPostProcessor processes SQL breakdown results.
type SQLBreakdownGenPostProcessor struct {
	engine core.Engine
}

// NewSQLBreakdownGenPostProcessor creates a new breakdown postprocessor.
func NewSQLBreakdownGenPostProcessor(engine core.Engine) *SQLBreakdownGenPostProcessor {
	return &SQLBreakdownGenPostProcessor{engine: engine}
}

// Run parses the LLM reply, validates steps, and builds CTE query.
func (p *SQLBreakdownGenPostProcessor) Run(ctx context.Context, reply string, projectID string) (*SQLBreakdownResult, error) {
	cleaned := sqlutil.CleanGenerationResult(reply)
	var result struct {
		Description string                 `json:"description"`
		Steps       []map[string]any       `json:"steps"`
	}
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, err
	}
	if len(result.Steps) == 0 {
		return &SQLBreakdownResult{Description: result.Description, Steps: []map[string]any{}}, nil
	}

	// Ensure last step has empty cte_name
	last := result.Steps[len(result.Steps)-1]
	last["cte_name"] = ""

	for _, step := range result.Steps {
		sql, _ := step["sql"].(string)
		quoted, ok := sqlutil.AddQuotes(sql)
		if !ok {
			return &SQLBreakdownResult{Description: result.Description, Steps: []map[string]any{}}, nil
		}
		step["sql"] = quoted
	}

	cteSQL := buildCTEQuery(result.Steps)
	r, err := p.engine.ExecuteSQL(ctx, cteSQL, core.EngineOpts{ProjectID: projectID, DryRun: true})
	if err != nil || !r.Success {
		return &SQLBreakdownResult{Description: result.Description, Steps: []map[string]any{}}, nil
	}

	return &SQLBreakdownResult{Description: result.Description, Steps: result.Steps}, nil
}

func buildCTEQuery(steps []map[string]any) string {
	var ctes []string
	for _, step := range steps {
		name, _ := step["cte_name"].(string)
		if name == "" {
			continue
		}
		sql, _ := step["sql"].(string)
		ctes = append(ctes, fmt.Sprintf("%s AS (%s)", name, sql))
	}
	lastSQL, _ := steps[len(steps)-1]["sql"].(string)
	if len(ctes) > 0 {
		return "WITH " + strings.Join(ctes, ",\n") + "\n" + lastSQL
	}
	return lastSQL
}

// SQLSummaryPostProcessor zips SQLs with summaries.
type SQLSummaryPostProcessor struct{}

// Result holds zipped sql/summary pairs.
type SQLSummaryResult struct {
	Results []map[string]string `json:"results"`
}

// Run parses summaries and pairs them with input SQLs.
func (p *SQLSummaryPostProcessor) Run(reply string, sqls []string) (*SQLSummaryResult, error) {
	cleaned := sqlutil.CleanGenerationResult(reply)
	var parsed struct {
		SQLSummaryResults []struct {
			Summary string `json:"summary"`
		} `json:"sql_summary_results"`
	}
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return nil, err
	}
	var results []map[string]string
	for i, s := range parsed.SQLSummaryResults {
		if i >= len(sqls) {
			break
		}
		results = append(results, map[string]string{"sql": sqls[i], "summary": s.Summary})
	}
	return &SQLSummaryResult{Results: results}, nil
}

// SQLAnswerPostProcessor parses the sql-answer LLM output.
type SQLAnswerPostProcessor struct{}

// SQLAnswerResult holds the answer output.
type SQLAnswerResult struct {
	Answer    string `json:"answer"`
	Reasoning string `json:"reasoning"`
}

// Run parses the LLM reply into answer and reasoning.
func (p *SQLAnswerPostProcessor) Run(reply string) (*SQLAnswerResult, error) {
	cleaned := sqlutil.CleanGenerationResult(reply)
	var result SQLAnswerResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// TEXT_TO_SQL_RULES is injected as alert in SQL generation prompts.
const TEXT_TO_SQL_RULES = `
### ALERT ###
- ONLY USE SELECT statements, NO DELETE, UPDATE OR INSERT etc. statements that might change the data in the database.
- ONLY USE the tables and columns mentioned in the database schema.
- ONLY USE "*" if the user query asks for all the columns of a table.
- ONLY CHOOSE columns belong to the tables mentioned in the database schema.
- YOU MUST USE "JOIN" if you choose columns from multiple tables!
- YOU MUST USE "lower(<column_name>) = lower(<value>)" function for case-insensitive comparison!
- DON'T USE "DATE_ADD" or "DATE_SUB" functions for date operations, instead use syntax like this "current_date - INTERVAL '7' DAY"!
- ALWAYS ADD "timestamp" to the front of the timestamp literal, ex. "timestamp '2024-02-20 12:00:00'"
- USE THE VIEW TO SIMPLIFY THE QUERY.
- DON'T MISUSE THE VIEW NAME. THE ACTUAL NAME IS FOLLOWING THE CREATE VIEW STATEMENT.
- MUST USE the value of alias from the comment section of the corresponding table or column in the DATABASE SCHEMA section for the column/table alias.
  - EXAMPLE
    DATABASE SCHEMA
    /* {"displayName":"_orders","description":"A model representing the orders data."} */
    CREATE TABLE orders (
      -- {"description":"A column that represents the timestamp when the order was approved.","alias":"_timestamp"}
      ApprovedTimestamp TIMESTAMP
    }

    SQL
    SELECT ApprovedTimestamp AS _timestamp FROM orders AS _orders;
`
