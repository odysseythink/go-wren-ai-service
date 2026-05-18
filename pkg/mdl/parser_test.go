package mdl

import "testing"

func TestParseMDL(t *testing.T) {
	input := `{"models": [], "views": [], "relationships": [], "metrics": []}`
	m, err := ParseMDL(input)
	if err != nil {
		t.Fatalf("ParseMDL error: %v", err)
	}
	if len(m.Models) != 0 {
		t.Fatal("expected empty models")
	}
}

func TestParseMDLWithModel(t *testing.T) {
	input := `{"models": [{"name": "orders", "primaryKey": "id", "columns": []}], "views": [], "relationships": [], "metrics": []}`
	m, err := ParseMDL(input)
	if err != nil {
		t.Fatalf("ParseMDL error: %v", err)
	}
	if len(m.Models) != 1 || m.Models[0].Name != "orders" {
		t.Fatal("model not parsed correctly")
	}
}

func TestParseMDLInvalidJSON(t *testing.T) {
	_, err := ParseMDL("{invalid}")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestValidateMDL(t *testing.T) {
	m := &MDL{}
	err := ValidateMDL(m)
	if err != nil {
		t.Fatalf("empty MDL should be valid: %v", err)
	}
}
