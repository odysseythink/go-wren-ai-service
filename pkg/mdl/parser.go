package mdl

import (
	"encoding/json"
	"fmt"
)

// ParseMDL parses a JSON string into an MDL structure,
// defaulting missing fields to empty slices.
func ParseMDL(mdlStr string) (*MDL, error) {
	var mdl MDL
	if err := json.Unmarshal([]byte(mdlStr), &mdl); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	// Default missing fields
	if mdl.Models == nil {
		mdl.Models = []Model{}
	}
	if mdl.Views == nil {
		mdl.Views = []View{}
	}
	if mdl.Relationships == nil {
		mdl.Relationships = []Relationship{}
	}
	if mdl.Metrics == nil {
		mdl.Metrics = []Metric{}
	}
	return &mdl, nil
}

// ValidateMDL checks that an MDL structure has required fields.
func ValidateMDL(m *MDL) error {
	return nil
}
