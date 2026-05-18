package core

// EngineOpts configures SQL execution.
type EngineOpts struct {
	ProjectID string
	DryRun    bool
	Limit     int
}

// EngineResult holds the outcome of SQL execution.
type EngineResult struct {
	Success bool
	Data    map[string]any
	Error   string
}
