package mdl

import "testing"

func TestMDLStruct(t *testing.T) {
	m := &MDL{
		Models:        []Model{{Name: "orders", PrimaryKey: "id"}},
		Relationships: []Relationship{{Condition: "a = b", JoinType: "MANY_TO_ONE"}},
		Views:         []View{{Name: "v1", Statement: "SELECT 1"}},
		Metrics:       []Metric{{Name: "revenue", BaseObject: "orders"}},
	}
	if len(m.Models) != 1 || m.Models[0].Name != "orders" {
		t.Fatal("Models not set correctly")
	}
	if m.Models[0].PrimaryKey != "id" {
		t.Fatal("PrimaryKey not set correctly")
	}
	if len(m.Relationships) != 1 {
		t.Fatal("Relationships not set correctly")
	}
}
