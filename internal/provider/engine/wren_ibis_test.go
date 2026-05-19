package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
)

func TestWrenIbis_ExecuteSQL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/connector/pg/query" {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["sql"] != "SELECT 1" {
				t.Fatalf("unexpected sql: %v", body["sql"])
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"rows": []any{}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	engine := NewWrenIbis(server.URL, "pg", "{}", nil)
	result, err := engine.ExecuteSQL(context.Background(), "SELECT 1", core.EngineOpts{DryRun: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
}
