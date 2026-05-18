package sqlutil

import "testing"

func TestCleanGenerationResult(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"remove code fence", "```sql\nSELECT 1\n```", "SELECT 1"},
		{"remove triple backticks", "```\nSELECT 1\n```", "SELECT 1"},
		{"remove semicolon", "SELECT 1;", "SELECT 1"},
		{"remove triple double quotes", "\"\"\"SELECT 1\"\"\"", "SELECT 1"},
		{"remove triple single quotes", "'''SELECT 1'''", "SELECT 1"},
		{"remove backslash n", "SELECT\\n1", "SELECT 1"},
		{"normalize whitespace", "SELECT   1   FROM   t", "SELECT 1 FROM t"},
		{"combined", "```sql\\nSELECT 1;\\n```", "SELECT 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanGenerationResult(tt.input)
			if got != tt.want {
				t.Fatalf("CleanGenerationResult(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
