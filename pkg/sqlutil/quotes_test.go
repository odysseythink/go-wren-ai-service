package sqlutil

import "testing"

func TestAddQuotes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantOK  bool
		wantHas string // substring that must appear in output
	}{
		{
			"simple select",
			"SELECT id FROM orders",
			true,
			`"id"`,
		},
		{
			"already quoted",
			`SELECT "id" FROM "orders"`,
			true,
			`"id"`,
		},
		{
			"keyword not quoted",
			"SELECT id FROM orders WHERE status = 'active'",
			true,
			`"status"`,
		},
		{
			"join",
			"SELECT o.id FROM orders o JOIN customers c ON o.cid = c.id",
			true,
			`"o"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := AddQuotes(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("AddQuotes(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if ok && !contains(got, tt.wantHas) {
				t.Fatalf("AddQuotes(%q) = %q, want substring %q", tt.input, got, tt.wantHas)
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		anyMatch(s, sub))
}

func anyMatch(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
