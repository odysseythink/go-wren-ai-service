package mdl

import (
	"encoding/json"
	"testing"
)

func TestConvertToDDL(t *testing.T) {
	mdl := &MDL{
		Models: []Model{
			{
				Name:       "orders",
				PrimaryKey: "id",
				Columns: []Column{
					{Name: "id", Type: "VARCHAR"},
					{Name: "status", Type: "VARCHAR", Properties: map[string]any{"description": "order status", "displayName": "_status"}},
				},
			},
		},
		Relationships: []Relationship{},
		Views:         []View{},
		Metrics:       []Metric{},
	}
	commands := ConvertToDDL(mdl, 50)
	if len(commands) == 0 {
		t.Fatal("expected DDL commands")
	}
	// Should contain a TABLE command and a TABLE_COLUMNS command
	hasTable := false
	hasColumns := false
	for _, cmd := range commands {
		var content map[string]any
		json.Unmarshal([]byte(cmd.Payload), &content)
		if content["type"] == "TABLE" {
			hasTable = true
		}
		if content["type"] == "TABLE_COLUMNS" {
			hasColumns = true
		}
	}
	if !hasTable {
		t.Fatal("expected TABLE command")
	}
	if !hasColumns {
		t.Fatal("expected TABLE_COLUMNS command")
	}
}

func TestConvertToTableDescriptions(t *testing.T) {
	mdl := &MDL{
		Models: []Model{
			{Name: "orders", Properties: map[string]any{"description": "Orders table"}},
		},
	}
	descs := ConvertToTableDescriptions(mdl)
	if len(descs) != 1 {
		t.Fatalf("expected 1 description, got %d", len(descs))
	}
}

func TestConvertViews(t *testing.T) {
	mdl := &MDL{
		Views: []View{
			{
				Name:      "v1",
				Statement: "SELECT 1",
				Properties: map[string]any{
					"summary": "test view",
					"question": "what is 1?",
					"historical_queries": []string{},
					"viewId": "view-1",
				},
			},
		},
	}
	docs := ConvertViews(mdl)
	if len(docs) != 1 {
		t.Fatalf("expected 1 view doc, got %d", len(docs))
	}
	if docs[0].Content != "what is 1?" {
		t.Fatalf("unexpected content: %s", docs[0].Content)
	}
}
