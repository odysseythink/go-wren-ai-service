package sqlutil

import "regexp"

var limitPattern = regexp.MustCompile(`(?i)\s*LIMIT\s+\d+(\s*;?\s*--.*)*$`)

// RemoveLimitStatement removes trailing LIMIT clauses from SQL.
func RemoveLimitStatement(sql string) string {
	return limitPattern.ReplaceAllString(sql, "")
}
