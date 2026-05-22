package mdl

import (
	"encoding/json"
	"strings"
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

func TestConvertToDDLWithNestedColumns(t *testing.T) {
	mdl := &MDL{
		Models: []Model{
			{
				Name:       "orders",
				PrimaryKey: "id",
				Columns: []Column{
					{Name: "id", Type: "VARCHAR"},
					{
						Name: "items",
						Type: "ARRAY",
						Properties: map[string]any{
							"displayName":  "订单项",
							"description":  "订单中的商品列表",
							"nestedName":   "item_name",
							"nestedPrice":  99.9,
							"nestedQty":    2,
						},
					},
				},
			},
		},
		Relationships: []Relationship{},
		Views:         []View{},
		Metrics:       []Metric{},
	}
	commands := ConvertToDDL(mdl, 50)
	var foundNested bool
	for _, cmd := range commands {
		if cmd.Name != "orders" {
			continue
		}
		var content map[string]any
		json.Unmarshal([]byte(cmd.Payload), &content)
		if content["type"] != "TABLE_COLUMNS" {
			continue
		}
		cols, _ := content["columns"].([]any)
		for _, c := range cols {
			col, _ := c.(map[string]any)
			if col["name"] != "items" {
				continue
			}
			comment, _ := col["comment"].(string)
			if strings.Contains(comment, "nested_columns") {
				foundNested = true
				// Verify all nested keys are present
				if !strings.Contains(comment, "nestedName") {
					t.Errorf("expected nestedName in comment, got: %s", comment)
				}
				if !strings.Contains(comment, "nestedPrice") {
					t.Errorf("expected nestedPrice in comment, got: %s", comment)
				}
				if !strings.Contains(comment, "nestedQty") {
					t.Errorf("expected nestedQty in comment, got: %s", comment)
				}
			}
		}
	}
	if !foundNested {
		t.Fatal("expected nested_columns in column comment")
	}
}

func TestConvertToDDLWithForeignKeys(t *testing.T) {
	mdl := &MDL{
		Models: []Model{
			{
				Name:       "orders",
				PrimaryKey: "id",
				Columns:    []Column{{Name: "id", Type: "VARCHAR"}, {Name: "customer_id", Type: "VARCHAR"}},
			},
			{
				Name:       "customers",
				PrimaryKey: "id",
				Columns:    []Column{{Name: "id", Type: "VARCHAR"}},
			},
		},
		Relationships: []Relationship{
			{
				Condition: "orders.customer_id = customers.id",
				JoinType:  "MANY_TO_ONE",
				Models:    []string{"orders", "customers"},
			},
		},
		Views:   []View{},
		Metrics: []Metric{},
	}
	commands := ConvertToDDL(mdl, 50)
	var foundFK bool
	for _, cmd := range commands {
		if cmd.Name != "orders" {
			continue
		}
		var content map[string]any
		json.Unmarshal([]byte(cmd.Payload), &content)
		if content["type"] != "TABLE_COLUMNS" {
			continue
		}
		cols, _ := content["columns"].([]any)
		for _, c := range cols {
			col, _ := c.(map[string]any)
			if col["type"] != "FOREIGN_KEY" {
				continue
			}
			foundFK = true
			constraint, _ := col["constraint"].(string)
			if !strings.Contains(constraint, "FOREIGN KEY (customer_id) REFERENCES customers(id)") {
				t.Errorf("unexpected FK constraint: %s", constraint)
			}
		}
	}
	if !foundFK {
		t.Fatal("expected FOREIGN_KEY entry in columns DDL")
	}
}

func TestConvertToDDLWithOneToOneFK(t *testing.T) {
	mdl := &MDL{
		Models: []Model{
			{
				Name:       "users",
				PrimaryKey: "id",
				Columns:    []Column{{Name: "id", Type: "VARCHAR"}, {Name: "profile_id", Type: "VARCHAR"}},
			},
			{
				Name:       "profiles",
				PrimaryKey: "id",
				Columns:    []Column{{Name: "id", Type: "VARCHAR"}},
			},
		},
		Relationships: []Relationship{
			{
				Condition: "users.profile_id = profiles.id",
				JoinType:  "ONE_TO_ONE",
				Models:    []string{"users", "profiles"},
			},
		},
		Views:   []View{},
		Metrics: []Metric{},
	}
	commands := ConvertToDDL(mdl, 50)
	var foundFK bool
	for _, cmd := range commands {
		if cmd.Name != "users" {
			continue
		}
		var content map[string]any
		json.Unmarshal([]byte(cmd.Payload), &content)
		if content["type"] != "TABLE_COLUMNS" {
			continue
		}
		cols, _ := content["columns"].([]any)
		for _, c := range cols {
			col, _ := c.(map[string]any)
			if col["type"] != "FOREIGN_KEY" {
				continue
			}
			foundFK = true
			constraint, _ := col["constraint"].(string)
			if !strings.Contains(constraint, "REFERENCES profiles(id)") {
				t.Errorf("unexpected FK constraint: %s", constraint)
			}
		}
	}
	if !foundFK {
		t.Fatal("expected FOREIGN_KEY entry for ONE_TO_ONE relationship")
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
