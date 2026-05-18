package model

import (
	"encoding/json"
	"testing"
)

func TestAskErrorJSON(t *testing.T) {
	e := AskError{Code: "NO_RELEVANT_DATA", Message: "No relevant data"}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	want := `{"code":"NO_RELEVANT_DATA","message":"No relevant data"}`
	if string(b) != want {
		t.Fatalf("expected %s, got %s", want, string(b))
	}
}

func TestQueryStatus(t *testing.T) {
	statuses := []string{"understanding", "searching", "generating", "finished", "failed", "stopped"}
	for _, s := range statuses {
		if s == "" {
			t.Fatal("status should not be empty")
		}
	}
}
