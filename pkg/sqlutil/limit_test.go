package sqlutil

import "testing"

func TestRemoveLimitStatement(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "SELECT 1 LIMIT 10", "SELECT 1"},
		{"with comment", "SELECT 1 LIMIT 10; -- comment", "SELECT 1"},
		{"no limit", "SELECT 1", "SELECT 1"},
		{"limit in string", "SELECT 'LIMIT 10'", "SELECT 'LIMIT 10'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveLimitStatement(tt.input)
			if got != tt.want {
				t.Fatalf("RemoveLimitStatement(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
