package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
)

func TestWrenUI_ExecuteSQL_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/graphql" {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			vars := body["variables"].(map[string]any)
			data := vars["data"].(map[string]any)
			if data["sql"] != "SELECT 1" {
				t.Fatalf("unexpected sql: %v", data["sql"])
			}
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"previewSql": "ok"}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	engine := NewWrenUI(server.URL)
	result, err := engine.ExecuteSQL(context.Background(), "SELECT 1", core.EngineOpts{ProjectID: "proj1", DryRun: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatal("expected success")
	}
}

func TestWrenUI_ExecuteSQL_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]any{{"message": "bad sql"}}})
	}))
	defer server.Close()

	engine := NewWrenUI(server.URL)
	result, err := engine.ExecuteSQL(context.Background(), "SELECT 1", core.EngineOpts{DryRun: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatal("expected failure")
	}
	if result.Error != "bad sql" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
}
