package sqlutil

import (
	"fmt"
	"strings"
	"unicode"
)

// SQL keywords that should not be quoted.
var sqlKeywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "AND": true, "OR": true,
	"NOT": true, "IN": true, "IS": true, "NULL": true, "AS": true,
	"ON": true, "JOIN": true, "INNER": true, "LEFT": true, "RIGHT": true,
	"OUTER": true, "CROSS": true, "FULL": true, "GROUP": true, "BY": true,
	"ORDER": true, "ASC": true, "DESC": true, "HAVING": true, "LIMIT": true,
	"OFFSET": true, "UNION": true, "ALL": true, "DISTINCT": true,
	"INSERT": true, "INTO": true, "VALUES": true, "UPDATE": true, "SET": true,
	"DELETE": true, "CREATE": true, "TABLE": true, "VIEW": true, "DROP": true,
	"ALTER": true, "ADD": true, "COLUMN": true, "INDEX": true, "IF": true,
	"EXISTS": true, "CASE": true, "WHEN": true, "THEN": true, "ELSE": true,
	"END": true, "LIKE": true, "BETWEEN": true, "TRUE": true, "FALSE": true,
	"CAST": true, "WITH": true, "RECURSIVE": true, "OVER": true,
	"PARTITION": true, "ROWS": true, "RANGE": true, "UNBOUNDED": true,
	"PRECEDING": true, "FOLLOWING": true, "CURRENT": true, "ROW": true,
	"FETCH": true, "NEXT": true, "ONLY": true, "FOR": true,
	"PRIMARY": true, "KEY": true, "FOREIGN": true, "REFERENCES": true,
	"CONSTRAINT": true, "UNIQUE": true, "CHECK": true, "DEFAULT": true,
	"NO": true, "ACTION": true, "CASCADE": true, "RESTRICT": true,
	"USING": true, "NATURAL": true, "INTERVAL": true, "EXTRACT": true,
	"COUNT": true, "SUM": true, "AVG": true, "MIN": true, "MAX": true,
	"COALESCE": true, "NULLIF": true, "TYPE": true, "INTEGER": true,
	"BIGINT": true, "VARCHAR": true, "DOUBLE": true, "BOOLEAN": true,
	"TIMESTAMP": true, "DATE": true, "TIME": true, "TEXT": true,
	"FLOAT": true, "REAL": true, "DECIMAL": true, "NUMERIC": true,
	"CHAR": true, "CHARACTER": true, "VARYING": true, "ARRAY": true,
	"MAP": true, "JSON": true, "JSONB": true,
	"MANY_TO_ONE": true, "ONE_TO_MANY": true, "ONE_TO_ONE": true,
}

type tokenKind int

const (
	tokenKeyword tokenKind = iota
	tokenIdentifier
	tokenString
	tokenNumber
	tokenPunctuation
	tokenWhitespace
	tokenComment
)

type token struct {
	kind tokenKind
	val  string
}

// AddQuotes tokenizes a Trino SQL statement and wraps unquoted
// identifiers in double quotes, then reassembles the SQL.
// Returns the quoted SQL and whether tokenization succeeded.
func AddQuotes(sql string) (string, bool) {
	tokens, err := tokenize(sql)
	if err != nil {
		return "", false
	}

	var b strings.Builder
	for _, tok := range tokens {
		switch tok.kind {
		case tokenIdentifier:
			upper := strings.ToUpper(tok.val)
			if sqlKeywords[upper] {
				b.WriteString(tok.val)
			} else {
				b.WriteString(fmt.Sprintf(`"%s"`, tok.val))
			}
		default:
			b.WriteString(tok.val)
		}
	}
	return b.String(), true
}

func tokenize(sql string) ([]token, error) {
	var tokens []token
	i := 0
	for i < len(sql) {
		ch := rune(sql[i])

		// Whitespace
		if unicode.IsSpace(ch) {
			start := i
			for i < len(sql) && unicode.IsSpace(rune(sql[i])) {
				i++
			}
			tokens = append(tokens, token{tokenWhitespace, sql[start:i]})
			continue
		}

		// Single-line comment
		if i+1 < len(sql) && sql[i] == '-' && sql[i+1] == '-' {
			start := i
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
			tokens = append(tokens, token{tokenComment, sql[start:i]})
			continue
		}

		// Multi-line comment
		if i+1 < len(sql) && sql[i] == '/' && sql[i+1] == '*' {
			start := i
			i += 2
			for i+1 < len(sql) && !(sql[i] == '*' && sql[i+1] == '/') {
				i++
			}
			if i+1 < len(sql) {
				i += 2
			}
			tokens = append(tokens, token{tokenComment, sql[start:i]})
			continue
		}

		// String literal (single-quoted)
		if ch == '\'' {
			start := i
			i++
			for i < len(sql) {
				if sql[i] == '\'' {
					i++
					if i < len(sql) && sql[i] == '\'' {
						i++ // escaped quote
						continue
					}
					break
				}
				i++
			}
			tokens = append(tokens, token{tokenString, sql[start:i]})
			continue
		}

		// Quoted identifier (double-quoted)
		if ch == '"' {
			start := i
			i++
			for i < len(sql) && sql[i] != '"' {
				i++
			}
			if i < len(sql) {
				i++ // closing quote
			}
			tokens = append(tokens, token{tokenIdentifier, sql[start:i]})
			continue
		}

		// Number
		if unicode.IsDigit(ch) {
			start := i
			for i < len(sql) && (unicode.IsDigit(rune(sql[i])) || sql[i] == '.') {
				i++
			}
			tokens = append(tokens, token{tokenNumber, sql[start:i]})
			continue
		}

		// Identifier or keyword
		if unicode.IsLetter(ch) || ch == '_' {
			start := i
			for i < len(sql) && (unicode.IsLetter(rune(sql[i])) || unicode.IsDigit(rune(sql[i])) || sql[i] == '_') {
				i++
			}
			word := sql[start:i]
			upper := strings.ToUpper(word)
			if sqlKeywords[upper] {
				tokens = append(tokens, token{tokenKeyword, word})
			} else {
				tokens = append(tokens, token{tokenIdentifier, word})
			}
			continue
		}

		// Punctuation and operators
		tokens = append(tokens, token{tokenPunctuation, string(ch)})
		i++
	}
	return tokens, nil
}
