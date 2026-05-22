package mdl

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DDLCommand represents a single DDL indexing command.
type DDLCommand struct {
	Name    string `json:"name"`
	Payload string `json:"payload"`
}

// ViewDocument represents a view converted for vector storage.
type ViewDocument struct {
	Content string
	Meta    map[string]any
}

// TableDescription represents a table description for indexing.
type TableDescription struct {
	Name        string
	MDLType     string
	Description string
}

// ConvertToDDL converts MDL models, views, and metrics into DDL commands
// for vector indexing, replicating Python's DDLConverter.
func ConvertToDDL(mdl *MDL, columnBatchSize int) []DDLCommand {
	if columnBatchSize <= 0 {
		columnBatchSize = 50
	}
	var commands []DDLCommand
	commands = append(commands, convertModelsAndRelationships(mdl.Models, mdl.Relationships, columnBatchSize)...)
	commands = append(commands, convertViewsDDL(mdl.Views)...)
	commands = append(commands, convertMetricsDDL(mdl.Metrics)...)
	return commands
}

func convertModelsAndRelationships(models []Model, relationships []Relationship, batchSize int) []DDLCommand {
	pkMap := make(map[string]string)
	for _, m := range models {
		pkMap[m.Name] = m.PrimaryKey
	}

	var commands []DDLCommand

	for _, model := range models {
		// Build column DDL entries
		var columnsDDL []map[string]any
		for _, col := range model.Columns {
			if col.Relationship == "" {
				entry := map[string]any{
					"type":           "COLUMN",
					"name":           col.Name,
					"data_type":      col.Type,
					"is_primary_key": col.Name == model.PrimaryKey,
				}
				var comment string
				if len(col.Properties) > 0 {
					props := map[string]any{}
					if dn, ok := col.Properties["displayName"]; ok {
						props["alias"] = dn
					}
					if desc, ok := col.Properties["description"]; ok {
						props["description"] = desc
					}
					// Extract nested column properties (keys starting with "nested")
					nestedCols := map[string]any{}
					for k, v := range col.Properties {
						if strings.HasPrefix(k, "nested") {
							nestedCols[k] = v
						}
					}
					if len(nestedCols) > 0 {
						props["nested_columns"] = nestedCols
					}
					b, _ := json.Marshal(props)
					comment = fmt.Sprintf("-- %s\n  ", string(b))
				}
				if col.IsCalculated {
					comment += fmt.Sprintf("-- This column is a Calculated Field\n  -- column expression: %s\n  ", col.Expression)
				}
				entry["comment"] = comment
				columnsDDL = append(columnsDDL, entry)
			}
		}

		// Foreign keys from relationships
		for _, rel := range relationships {
			if len(rel.Models) != 2 {
				continue
			}
			condition := rel.Condition
			joinType := strings.ToUpper(rel.JoinType)
			relModels := rel.Models

			comment := fmt.Sprintf(`-- {"condition": %s, "joinType": %s}`+"\n  ", condition, rel.JoinType)
			shouldAddFK := false
			var fkConstraint string

			if model.Name == relModels[0] && joinType == "MANY_TO_ONE" {
				relatedTable := relModels[1]
				parts := strings.Split(condition, " = ")
				if len(parts) == 2 {
					fkColParts := strings.Split(parts[0], ".")
					if len(fkColParts) == 2 {
						fkColumn := fkColParts[1]
						fkConstraint = fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s(%s)", fkColumn, relatedTable, pkMap[relatedTable])
						shouldAddFK = true
					}
				}
			} else if model.Name == relModels[1] && joinType == "ONE_TO_MANY" {
				relatedTable := relModels[0]
				parts := strings.Split(condition, " = ")
				if len(parts) == 2 {
					fkColParts := strings.Split(parts[1], ".")
					if len(fkColParts) == 2 {
						fkColumn := fkColParts[1]
						fkConstraint = fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s(%s)", fkColumn, relatedTable, pkMap[relatedTable])
						shouldAddFK = true
					}
				}
			} else if joinType == "ONE_TO_ONE" {
				for idx, m := range relModels {
					if m == model.Name {
						var relatedTable string
						for _, rm := range relModels {
							if rm != model.Name {
								relatedTable = rm
								break
							}
						}
						parts := strings.Split(condition, " = ")
						if idx < len(parts) {
							fkColParts := strings.Split(parts[idx], ".")
							if len(fkColParts) == 2 {
								fkColumn := fkColParts[1]
								fkConstraint = fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s(%s)", fkColumn, relatedTable, pkMap[relatedTable])
								shouldAddFK = true
							}
						}
						break
					}
				}
			}

			if shouldAddFK {
				columnsDDL = append(columnsDDL, map[string]any{
					"type":       "FOREIGN_KEY",
					"comment":    comment,
					"constraint": fkConstraint,
					"tables":     relModels,
				})
			}
		}

		// TABLE command
		var modelComment string
		if len(model.Properties) > 0 {
			props := map[string]any{}
			if dn, ok := model.Properties["displayName"]; ok {
				props["alias"] = dn
			}
			if desc, ok := model.Properties["description"]; ok {
				props["description"] = desc
			}
			b, _ := json.Marshal(props)
			modelComment = fmt.Sprintf("\n/* %s */\n", string(b))
		}

		tablePayload, _ := json.Marshal(map[string]any{
			"type":    "TABLE",
			"comment": modelComment,
			"name":    model.Name,
		})
		commands = append(commands, DDLCommand{
			Name:    model.Name,
			Payload: string(tablePayload),
		})

		// TABLE_COLUMNS commands (batched)
		for i := 0; i < len(columnsDDL); i += batchSize {
			end := i + batchSize
			if end > len(columnsDDL) {
				end = len(columnsDDL)
			}
			batch := columnsDDL[i:end]
			colPayload, _ := json.Marshal(map[string]any{
				"type":    "TABLE_COLUMNS",
				"columns": batch,
			})
			commands = append(commands, DDLCommand{
				Name:    model.Name,
				Payload: string(colPayload),
			})
		}
	}

	return commands
}

func convertViewsDDL(views []View) []DDLCommand {
	var commands []DDLCommand
	for _, v := range views {
		payload, _ := json.Marshal(map[string]any{
			"type":      "VIEW",
			"comment":   formatViewComment(v),
			"name":      v.Name,
			"statement": v.Statement,
		})
		commands = append(commands, DDLCommand{Name: v.Name, Payload: string(payload)})
	}
	return commands
}

func formatViewComment(v View) string {
	if len(v.Properties) > 0 {
		b, _ := json.Marshal(v.Properties)
		return fmt.Sprintf("/* %s */\n", string(b))
	}
	return ""
}

func convertMetricsDDL(metrics []Metric) []DDLCommand {
	var commands []DDLCommand
	for _, metric := range metrics {
		var columnsDDL []map[string]any
		for _, dim := range metric.Dimension {
			columnsDDL = append(columnsDDL, map[string]any{
				"type":      "COLUMN",
				"comment":   "-- This column is a dimension\n  ",
				"name":      dim.Name,
				"data_type": dim.Type,
			})
		}
		for _, meas := range metric.Measure {
			columnsDDL = append(columnsDDL, map[string]any{
				"type":      "COLUMN",
				"comment":   fmt.Sprintf("-- This column is a measure\n  -- expression: %s\n  ", meas.Expression),
				"name":      meas.Name,
				"data_type": meas.Type,
			})
		}
		comment := fmt.Sprintf("\n/* This table is a metric */\n/* Metric Base Object: %s */\n", metric.BaseObject)
		payload, _ := json.Marshal(map[string]any{
			"type":    "METRIC",
			"comment": comment,
			"name":    metric.Name,
			"columns": columnsDDL,
		})
		commands = append(commands, DDLCommand{Name: metric.Name, Payload: string(payload)})
	}
	return commands
}

// ConvertToTableDescriptions extracts table descriptions from MDL.
func ConvertToTableDescriptions(mdl *MDL) []TableDescription {
	var descs []TableDescription
	type entry struct {
		mdlType string
		payload []map[string]any
	}
	entries := []entry{
		{"MODEL", modelSliceToAny(mdl.Models)},
		{"METRIC", metricSliceToAny(mdl.Metrics)},
		{"VIEW", viewSliceToAny(mdl.Views)},
	}
	for _, e := range entries {
		for _, unit := range e.payload {
			name, _ := unit["name"].(string)
			desc := ""
			if props, ok := unit["properties"].(map[string]any); ok {
				if d, ok := props["description"].(string); ok {
					desc = d
				}
			}
			descs = append(descs, TableDescription{
				Name:        name,
				MDLType:     e.mdlType,
				Description: desc,
			})
		}
	}
	return descs
}

func modelSliceToAny(models []Model) []map[string]any {
	var result []map[string]any
	for _, m := range models {
		result = append(result, map[string]any{"name": m.Name, "properties": m.Properties})
	}
	return result
}

func metricSliceToAny(metrics []Metric) []map[string]any {
	var result []map[string]any
	for _, m := range metrics {
		result = append(result, map[string]any{"name": m.Name, "properties": m.Properties})
	}
	return result
}

func viewSliceToAny(views []View) []map[string]any {
	var result []map[string]any
	for _, v := range views {
		result = append(result, map[string]any{"name": v.Name, "properties": v.Properties})
	}
	return result
}

// ConvertViews converts MDL views into ViewDocuments for vector storage.
func ConvertViews(mdl *MDL) []ViewDocument {
	var docs []ViewDocument
	for _, v := range mdl.Views {
		props := v.Properties
		if props == nil {
			props = map[string]any{}
		}

		var histQueries []string
		if hq, ok := props["historical_queries"].([]any); ok {
			for _, q := range hq {
				if s, ok := q.(string); ok {
					histQueries = append(histQueries, s)
				}
			}
		}
		question, _ := props["question"].(string)
		parts := append(histQueries, question)
		content := strings.Join(parts, " ")

		meta := map[string]any{
			"summary":   props["summary"],
			"statement": v.Statement,
			"viewId":    props["viewId"],
		}

		docs = append(docs, ViewDocument{Content: content, Meta: meta})
	}
	return docs
}
