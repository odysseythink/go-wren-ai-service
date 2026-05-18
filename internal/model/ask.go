package model

// AskConfigurations holds optional query configuration.
type AskConfigurations struct {
	FiscalYear *FiscalYear `json:"fiscal_year,omitempty"`
	Language   string      `json:"language"`
}

// FiscalYear defines a custom fiscal year range.
type FiscalYear struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// AskHistory represents prior query context.
type AskHistory struct {
	SQL     string         `json:"sql"`
	Summary string         `json:"summary"`
	Steps   []SQLBreakdown `json:"steps"`
}

// AskRequest is the POST /v1/asks request body.
type AskRequest struct {
	Query          string            `json:"query"`
	ProjectID      *string           `json:"project_id,omitempty"`
	MdlHash        *string           `json:"mdl_hash,omitempty"`
	ThreadID       *string           `json:"thread_id,omitempty"`
	UserID         *string           `json:"user_id,omitempty"`
	History        *AskHistory       `json:"history,omitempty"`
	Configurations AskConfigurations `json:"configurations"`
}

// AskResponse is the POST /v1/asks response.
type AskResponse struct {
	QueryID string `json:"query_id"`
}

// StopAskRequest is the PATCH /v1/asks/{query_id} request.
type StopAskRequest struct {
	Status string `json:"status"`
}

// StopAskResponse is the PATCH /v1/asks/{query_id} response.
type StopAskResponse struct {
	QueryID string `json:"query_id"`
}

// AskResult is a single SQL result in an ask response.
type AskResult struct {
	SQL     string  `json:"sql"`
	Summary string  `json:"summary"`
	Type    string  `json:"type"`
	ViewID  *string `json:"viewId,omitempty"`
}

// AskResultResponse is the GET /v1/asks/{query_id}/result response.
type AskResultResponse struct {
	Status   string      `json:"status"`
	Response []AskResult `json:"response,omitempty"`
	Error    *AskError   `json:"error,omitempty"`
}

// AskDetailsRequest is the POST /v1/ask-details request.
type AskDetailsRequest struct {
	Query     string  `json:"query"`
	SQL       string  `json:"sql"`
	Summary   string  `json:"summary"`
	MdlHash   *string `json:"mdl_hash,omitempty"`
	ThreadID  *string `json:"thread_id,omitempty"`
	ProjectID *string `json:"project_id,omitempty"`
	UserID    *string `json:"user_id,omitempty"`
}

// AskDetailsResponse is the POST /v1/ask-details response.
type AskDetailsResponse struct {
	QueryID string `json:"query_id"`
}

// AskDetailsResultResponse is the GET /v1/ask-details/{query_id}/result response.
type AskDetailsResultResponse struct {
	Status   string                `json:"status"`
	Response *AskDetailsResultData `json:"response,omitempty"`
	Error    *AskError             `json:"error,omitempty"`
}

// AskDetailsResultData holds the ask-details result content.
type AskDetailsResultData struct {
	Description string         `json:"description"`
	Steps       []SQLBreakdown `json:"steps"`
}

// SQLBreakdown represents a step in a SQL breakdown.
type SQLBreakdown struct {
	SQL     string `json:"sql"`
	Summary string `json:"summary"`
	CTEName string `json:"cte_name,omitempty"`
}
