package common

import (
	"bytes"
	"fmt"
	"text/template"
)

// PromptBuilder wraps text/template for prompt construction.
type PromptBuilder struct {
	tmpl *template.Template
}

// NewPromptBuilder creates a PromptBuilder from a template string.
func NewPromptBuilder(tmplStr string) (*PromptBuilder, error) {
	tmpl, err := template.New("prompt").Parse(tmplStr)
	if err != nil {
		return nil, fmt.Errorf("parse prompt template: %w", err)
	}
	return &PromptBuilder{tmpl: tmpl}, nil
}

// Build executes the template with the given data map.
func (b *PromptBuilder) Build(data map[string]any) (string, error) {
	var buf bytes.Buffer
	if err := b.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute prompt template: %w", err)
	}
	return buf.String(), nil
}
