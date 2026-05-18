package sqlutil

import (
	"regexp"
	"strings"
)

// CleanGenerationResult normalizes LLM output by removing markdown
// artifacts and normalizing whitespace.
func CleanGenerationResult(result string) string {
	s := strings.ReplaceAll(result, "\\n", " ")
	s = strings.ReplaceAll(s, "```sql", "")
	s = strings.ReplaceAll(s, "```", "")
	s = strings.ReplaceAll(s, `"""`, "")
	s = strings.ReplaceAll(s, "'''", "")
	s = strings.ReplaceAll(s, ";", "")
	// normalize whitespace
	ws := regexp.MustCompile(`\s+`)
	s = ws.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
