package model

import (
	"encoding/json"
	"testing"
)

func TestAskRequestJSON(t *testing.T) {
	req := AskRequest{
		Query:          "show me orders",
		ProjectID:      strPtr("proj1"),
		MdlHash:        strPtr("hash1"),
		Configurations: AskConfigurations{Language: "English"},
	}
	b, _ := json.Marshal(req)
	var m map[string]any
	json.Unmarshal(b, &m)
	if m["query"] != "show me orders" {
		t.Fatal("query not serialized")
	}
	if m["project_id"] != "proj1" {
		t.Fatal("project_id not serialized")
	}
}

func TestAskResultResponseJSON(t *testing.T) {
	resp := AskResultResponse{Status: "finished"}
	resp.Response = []AskResult{
		{SQL: "SELECT 1", Summary: "test", Type: "llm"},
	}
	b, _ := json.Marshal(resp)
	var m map[string]any
	json.Unmarshal(b, &m)
	if m["status"] != "finished" {
		t.Fatal("status not serialized")
	}
	arr := m["response"].([]any)
	if len(arr) != 1 {
		t.Fatal("expected 1 result")
	}
}

func TestSQLExpansionResultResponse(t *testing.T) {
	resp := SQLExpansionResultResponse{Status: "generating"}
	b, _ := json.Marshal(resp)
	var m map[string]any
	json.Unmarshal(b, &m)
	if m["status"] != "generating" {
		t.Fatal("status wrong")
	}
}

func TestSemanticsPrepStatusResponse(t *testing.T) {
	resp := SemanticsPrepStatusResponse{Status: "indexing"}
	b, _ := json.Marshal(resp)
	var m map[string]any
	json.Unmarshal(b, &m)
	if m["status"] != "indexing" {
		t.Fatal("status wrong")
	}
}

func strPtr(s string) *string { return &s }
