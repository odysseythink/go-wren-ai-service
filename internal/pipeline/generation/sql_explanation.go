package generation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/model"
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

// SQLExplanationRequest is the input for one step.
type SQLExplanationRequest struct {
	Question           string
	SQL                string
	SQLSummary         string
	SQLAnalysisResults []map[string]any
}

// Run executes SQL explanation for a single step.
func (p *SQLExplanation) Run(ctx context.Context, input any) (any, error) {
	req, ok := input.(*SQLExplanationRequest)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	// Preprocess raw analysis results
	preprocessor := &SQLExplanationPreProcessor{}
	preprocessed := preprocessor.Run(req.SQLAnalysisResults)
	if len(preprocessed) == 0 {
		return []model.SQLExplanationItem{}, nil
	}

	// Generate explanation for each preprocessed result
	var explanations []map[string]any
	for _, item := range preprocessed {
		builder, _ := common.NewPromptBuilder(sqlExplanationUserPrompt)
		prompt, _ := builder.Build(map[string]any{
			"question":            req.Question,
			"sql":                 req.SQL,
			"sql_summary":         req.SQLSummary,
			"sql_analysis_result": item,
		})
		gen, _ := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
			SystemPrompt:     sqlExplanationSystemPrompt,
			GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
		})
		result, _ := gen.Run(ctx, prompt)
		if len(result.Replies) == 0 {
			continue
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(result.Replies[0]), &parsed); err != nil {
			continue
		}
		explanations = append(explanations, parsed)
	}

	// Postprocess to zip preprocessed data with LLM outputs
	postprocessor := &SQLExplanationPostProcessor{}
	return postprocessor.Run(preprocessed, explanations), nil
}

// SQLExplanationPreProcessor transforms raw sql_analysis_results into composed expressions.
type SQLExplanationPreProcessor struct{}

// Run preprocesses the raw analysis results.
func (p *SQLExplanationPreProcessor) Run(sqlAnalysisResults []map[string]any) []map[string]any {
	var results []map[string]any
	for _, analysis := range sqlAnalysisResults {
		if isSubqueryOrCte(analysis) {
			continue
		}
		if filter, ok := analysis["filter"]; ok && filter != nil {
			results = append(results, map[string]any{
				"filter": composeFilterExpression(filter.(map[string]any)),
			})
		}
		if groupByKeys, ok := analysis["groupByKeys"]; ok && groupByKeys != nil {
			results = append(results, map[string]any{
				"groupByKeys": composeGroupByKeys(groupByKeys.([]any)),
			})
		}
		if relation, ok := analysis["relation"]; ok && relation != nil {
			results = append(results, map[string]any{
				"relation": composeRelation(relation.(map[string]any)),
			})
		}
		if selectItems, ok := analysis["selectItems"]; ok && selectItems != nil {
			results = append(results, map[string]any{
				"selectItems": composeSelectItems(selectItems.([]any)),
			})
		}
		if sortings, ok := analysis["sortings"]; ok && sortings != nil {
			results = append(results, map[string]any{
				"sortings": composeSortings(sortings.([]any)),
			})
		}
	}
	return results
}

func isSubqueryOrCte(analysis map[string]any) bool {
	if v, ok := analysis["isSubqueryOrCte"].(bool); ok {
		return v
	}
	return false
}

func composeFilterExpression(filter map[string]any) map[string]any {
	typ, _ := filter["type"].(string)
	switch typ {
	case "EXPR":
		return map[string]any{
			"values": filter["node"],
			"id":   getString(filter, "id"),
		}
	case "AND", "OR":
		left := composeFilterExpression(filter["left"].(map[string]any))
		right := composeFilterExpression(filter["right"].(map[string]any))
		return map[string]any{
			"values": fmt.Sprintf("%s %s %s", left["values"], typ, right["values"]),
			"id":   getString(filter, "id"),
		}
	default:
		return map[string]any{"values": "", "id": ""}
	}
}

func composeGroupByKeys(groupByKeys []any) []map[string]any {
	var results []map[string]any
	for _, group := range groupByKeys {
		groupList, ok := group.([]any)
		if !ok {
			continue
		}
		for _, g := range groupList {
			gm, ok := g.(map[string]any)
			if !ok {
				continue
			}
			results = append(results, map[string]any{
				"values": getString(gm, "expression"),
				"id":   getString(gm, "id"),
			})
		}
	}
	return results
}

func composeRelation(relation map[string]any) []map[string]any {
	var results []map[string]any
	var collect func(r map[string]any, topLevel bool)
	collect = func(r map[string]any, topLevel bool) {
		if isSubqueryOrHasSubqueryChild(r) {
			return
		}
		typ, _ := r["type"].(string)
		if typ == "TABLE" && topLevel {
			results = append(results, map[string]any{
				"values": map[string]any{
					"type":      typ,
					"tableName": getString(r, "tableName"),
				},
				"id": getString(r, "id"),
			})
		} else if strings.HasSuffix(typ, "_JOIN") {
			exprSources := []map[string]any{}
			if es, ok := r["exprSources"].([]any); ok {
				for _, e := range es {
					em, _ := e.(map[string]any)
					exprSources = append(exprSources, map[string]any{
						"expression":    getString(em, "expression"),
						"sourceDataset": getString(em, "sourceDataset"),
					})
				}
			}
			results = append(results, map[string]any{
				"values": map[string]any{
					"type":        typ,
					"criteria":    getString(r, "criteria"),
					"exprSources": exprSources,
				},
				"id": getString(r, "id"),
			})
			if left, ok := r["left"].(map[string]any); ok {
				collect(left, false)
			}
			if right, ok := r["right"].(map[string]any); ok {
				collect(right, false)
			}
		}
	}
	collect(relation, true)
	return results
}

func isSubqueryOrHasSubqueryChild(relation map[string]any) bool {
	typ, _ := relation["type"].(string)
	if typ == "SUBQUERY" {
		return true
	}
	if strings.HasSuffix(typ, "_JOIN") {
		if left, ok := relation["left"].(map[string]any); ok {
			if lt, _ := left["type"].(string); lt == "SUBQUERY" {
				return true
			}
		}
		if right, ok := relation["right"].(map[string]any); ok {
			if rt, _ := right["type"].(string); rt == "SUBQUERY" {
				return true
			}
		}
	}
	return false
}

func composeSelectItems(selectItems []any) map[string]any {
	result := map[string]any{
		"withFunctionCallOrMathematicalOperation":    []map[string]any{},
		"withoutFunctionCallOrMathematicalOperation": []map[string]any{},
	}
	for _, item := range selectItems {
		si, ok := item.(map[string]any)
		if !ok {
			continue
		}
		props, _ := si["properties"].(map[string]any)
		hasFunc := getString(props, "includeFunctionCall") == "true"
		hasMath := getString(props, "includeMathematicalOperation") == "true"
		entry := map[string]any{
			"values": map[string]any{
				"alias":      getString(si, "alias"),
				"expression": getString(si, "expression"),
			},
			"id": getString(si, "id"),
		}
		if hasFunc || hasMath {
			result["withFunctionCallOrMathematicalOperation"] = append(
				result["withFunctionCallOrMathematicalOperation"].([]map[string]any), entry)
		} else {
			result["withoutFunctionCallOrMathematicalOperation"] = append(
				result["withoutFunctionCallOrMathematicalOperation"].([]map[string]any), entry)
		}
	}
	return result
}

func composeSortings(sortings []any) []map[string]any {
	var results []map[string]any
	for _, s := range sortings {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		results = append(results, map[string]any{
			"values": fmt.Sprintf("%s %s", getString(sm, "expression"), getString(sm, "ordering")),
			"id":   getString(sm, "id"),
		})
	}
	return results
}

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// SQLExplanationPostProcessor zips preprocessed analysis results with LLM explanations.
type SQLExplanationPostProcessor struct{}

// Run combines preprocessed data with LLM outputs into typed explanation items.
func (p *SQLExplanationPostProcessor) Run(preprocessed []map[string]any, explanations []map[string]any) []model.SQLExplanationItem {
	var results []model.SQLExplanationItem
	if len(preprocessed) == 0 || len(explanations) == 0 {
		return results
	}

	// Match each preprocessed item with its corresponding explanation
	for i, preproc := range preprocessed {
		if i >= len(explanations) {
			break
		}
		expResult := explanations[i]
		resultsMap, _ := expResult["results"].(map[string]any)
		if resultsMap == nil {
			continue
		}

		// Determine which type this preprocessed item is
		for key, preprocValue := range preproc {
			switch key {
			case "filter":
				if filterExp, ok := preprocValue.(map[string]any); ok {
					if filterResults, ok := resultsMap["filter"].([]any); ok && len(filterResults) > 0 {
						results = append(results, model.SQLExplanationItem{
							Type: "filter",
							Payload: map[string]any{
								"id":         getString(filterExp, "id"),
								"expression": filterExp["values"],
								"explanation": extractToStr(filterResults[0]),
							},
						})
					}
				}
			case "groupByKeys":
				if groupKeys, ok := preprocValue.([]map[string]any); ok {
					if groupResults, ok := resultsMap["groupByKeys"].([]any); ok {
						for j, gk := range groupKeys {
							if j >= len(groupResults) {
								break
							}
							results = append(results, model.SQLExplanationItem{
								Type: "groupByKeys",
								Payload: map[string]any{
									"id":         getString(gk, "id"),
									"expression": gk["values"],
									"explanation": extractToStr(groupResults[j]),
								},
							})
						}
					}
				}
			case "relation":
				if relations, ok := preprocValue.([]map[string]any); ok {
					if relResults, ok := resultsMap["relation"].([]any); ok {
						for j, rel := range relations {
							if j >= len(relResults) {
								break
							}
							payload := map[string]any{
								"id":          getString(rel, "id"),
								"explanation": extractToStr(relResults[j]),
							}
							if vals, ok := rel["values"].(map[string]any); ok {
								for vk, vv := range vals {
									payload[vk] = vv
								}
							}
							results = append(results, model.SQLExplanationItem{
								Type:    "relation",
								Payload: payload,
							})
						}
					}
				}
			case "selectItems":
				if selectItems, ok := preprocValue.(map[string]any); ok {
					if siResults, ok := resultsMap["selectItems"].(map[string]any); ok {
						withFunc := selectItems["withFunctionCallOrMathematicalOperation"].([]map[string]any)
						withoutFunc := selectItems["withoutFunctionCallOrMathematicalOperation"].([]map[string]any)
						withResults := []any{}
						if wr, ok := siResults["withFunctionCallOrMathematicalOperation"].([]any); ok {
							withResults = wr
						}
						withoutResults := []any{}
						if wr, ok := siResults["withoutFunctionCallOrMathematicalOperation"].([]any); ok {
							withoutResults = wr
						}
						for j, si := range withFunc {
							if j >= len(withResults) {
								break
							}
							if vals, ok := si["values"].(map[string]any); ok {
								results = append(results, model.SQLExplanationItem{
									Type: "selectItems",
									Payload: map[string]any{
										"id":     getString(si, "id"),
										"alias":  getString(vals, "alias"),
										"expression": vals["expression"],
										"isFunctionCallOrMathematicalOperation": true,
										"explanation": extractToStr(withResults[j]),
									},
								})
							}
						}
						for j, si := range withoutFunc {
							if j >= len(withoutResults) {
								break
							}
							if vals, ok := si["values"].(map[string]any); ok {
								results = append(results, model.SQLExplanationItem{
									Type: "selectItems",
									Payload: map[string]any{
										"id":     getString(si, "id"),
										"alias":  getString(vals, "alias"),
										"expression": vals["expression"],
										"isFunctionCallOrMathematicalOperation": false,
										"explanation": extractToStr(withoutResults[j]),
									},
								})
							}
						}
					}
				}
			case "sortings":
				if sortings, ok := preprocValue.([]map[string]any); ok {
					if sortResults, ok := resultsMap["sortings"].([]any); ok {
						for j, s := range sortings {
							if j >= len(sortResults) {
								break
							}
							results = append(results, model.SQLExplanationItem{
								Type: "sortings",
								Payload: map[string]any{
									"id":         getString(s, "id"),
									"expression": s["values"],
									"explanation": extractToStr(sortResults[j]),
								},
							})
						}
					}
				}
			}
		}
	}
	return results
}

func extractToStr(data any) string {
	if data == nil {
		return ""
	}
	if s, ok := data.(string); ok {
		return s
	}
	if arr, ok := data.([]any); ok && len(arr) > 0 {
		if s, ok := arr[0].(string); ok {
			return s
		}
	}
	return ""
}

const sqlExplanationSystemPrompt = `
### INSTRUCTIONS ###
Given the question, sql query, sql analysis result to the sql query, sql query summary for reference,
please explain sql analysis result within 20 words in layman term based on sql query:
1. how does the expression work
2. why this expression is given based on the question
3. why can it answer user's question
The sql analysis will be one of the types: selectItems, relation, filter, groupByKeys, sortings

### ALERT ###
1. There must be only one type of sql analysis result in the input(sql analysis result) and output(sql explanation)
2. The number of the sql explanation must be the same as the number of the <expression_string> in the input

### INPUT STRUCTURE ###
{
  "selectItems": {
    "withFunctionCallOrMathematicalOperation": [
      {"alias": "<alias_string>", "expression": "<expression_string>"}
    ],
    "withoutFunctionCallOrMathematicalOperation": [
      {"alias": "<alias_string>", "expression": "<expression_string>"}
    ]
  }
} | {
  "relation": [
    {"type": "INNER_JOIN" | "LEFT_JOIN" | "RIGHT_JOIN" | "FULL_JOIN" | "CROSS_JOIN" | "IMPLICIT_JOIN", "criteria": <criteria_string>, "exprSources": [{"expression": <expression_string>, "sourceDataset": <sourceDataset_string>}]}
    | {"type": "TABLE", "alias": "<alias_string>", "tableName": "<expression_string>"}
  ]
} | {
  "filter": <expression_string>
} | {
  "groupByKeys": [<expression_string>, ...]
} | {
  "sortings": [<expression_string>, ...]
}

### OUTPUT STRUCTURE ###
Please generate the output with the following JSON format depending on the type of the sql analysis result:

{
  "results": {
    "selectItems": {
      "withFunctionCallOrMathematicalOperation": [<explanation1_string>, <explanation2_string>],
      "withoutFunctionCallOrMathematicalOperation": [<explanation1_string>, <explanation2_string>]
    }
  }
} | {
  "results": {
    "groupByKeys|sortings|relation|filter": [<explanation1_string>, <explanation2_string>, ...]
  }
}
`

const sqlExplanationUserPrompt = `
Question: {{.question}}
SQL query: {{.sql}}
SQL query summary: {{.sql_summary}}
SQL query analysis: {{.sql_analysis_result}}

Let's think step by step.
`
