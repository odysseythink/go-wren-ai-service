package model

// SQLAnswerRequest is the POST /v1/sql-answers request.
type SQLAnswerRequest struct {
	Query      string  `json:"query"`
	SQL        string  `json:"sql"`
	SQLSummary string  `json:"sql_summary"`
	ThreadID   *string `json:"thread_id,omitempty"`
	UserID     *string `json:"user_id,omitempty"`
}

// SQLAnswerResponse is the POST /v1/sql-answers response.
type SQLAnswerResponse struct {
	QueryID string `json:"query_id"`
}

// SQLAnswerResultResponse is the GET /v1/sql-answers/{query_id}/result response.
type SQLAnswerResultResponse struct {
	Status   string              `json:"status"`
	Response *SQLAnswerResultData `json:"response,omitempty"`
	Error    *AskError           `json:"error,omitempty"`
}

// SQLAnswerResultData holds the sql-answer result.
type SQLAnswerResultData struct {
	Answer    string `json:"answer"`
	Reasoning string `json:"reasoning"`
}

// SQLExplanationRequest is the POST /v1/sql-explanations request.
type SQLExplanationRequest struct {
	Question                 string                   `json:"question"`
	StepsWithAnalysisResults []StepWithAnalysisResult `json:"steps_with_analysis_results"`
	MdlHash                  *string                  `json:"mdl_hash,omitempty"`
	ThreadID                 *string                  `json:"thread_id,omitempty"`
	ProjectID                *string                  `json:"project_id,omitempty"`
	UserID                   *string                  `json:"user_id,omitempty"`
}

// StepWithAnalysisResult pairs a SQL step with its analysis.
type StepWithAnalysisResult struct {
	SQL                string `json:"sql"`
	Summary            string `json:"summary"`
	SQLAnalysisResults []any  `json:"sql_analysis_results"`
}

// SQLExplanationResponse is the POST /v1/sql-explanations response.
type SQLExplanationResponse struct {
	QueryID string `json:"query_id"`
}

// SQLExplanationResultResponse is the GET /v1/sql-explanations/{query_id}/result response.
type SQLExplanationResultResponse struct {
	Status   string                `json:"status"`
	Response [][]SQLExplanationItem `json:"response,omitempty"`
	Error    *AskError             `json:"error,omitempty"`
}

// SQLExplanationItem is a single typed explanation for a SQL analysis result.
type SQLExplanationItem struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

// SQLExpansionRequest is the POST /v1/sql-expansions request.
type SQLExpansionRequest struct {
	Query     string      `json:"query"`
	SQL       string      `json:"sql"`
	Summary   string      `json:"summary"`
	History   *AskHistory `json:"history,omitempty"`
	ProjectID *string     `json:"project_id,omitempty"`
	MdlHash   *string     `json:"mdl_hash,omitempty"`
	ThreadID  *string     `json:"thread_id,omitempty"`
	UserID    *string     `json:"user_id,omitempty"`
}

// SQLExpansionResponse is the POST /v1/sql-expansions response.
type SQLExpansionResponse struct {
	QueryID string `json:"query_id"`
}

// StopSQLExpansionRequest is the PATCH /v1/sql-expansions/{query_id} request.
type StopSQLExpansionRequest struct {
	Status string `json:"status"`
}

// StopSQLExpansionResponse is the PATCH response.
type StopSQLExpansionResponse struct {
	QueryID string `json:"query_id"`
}

// SQLExpansionResultResponse is the GET response.
type SQLExpansionResultResponse struct {
	Status   string                  `json:"status"`
	Response *SQLExpansionResultData `json:"response,omitempty"`
	Error    *AskError               `json:"error,omitempty"`
}

// SQLExpansionResultData holds expansion result.
type SQLExpansionResultData struct {
	Description string         `json:"description"`
	Steps       []SQLBreakdown `json:"steps"`
}

// SQLRegenerationRequest is the POST /v1/sql-regenerations request.
type SQLRegenerationRequest struct {
	Description string         `json:"description"`
	Steps       []SQLBreakdown `json:"steps"`
	MdlHash     *string        `json:"mdl_hash,omitempty"`
	ThreadID    *string        `json:"thread_id,omitempty"`
	ProjectID   *string        `json:"project_id,omitempty"`
	UserID      *string        `json:"user_id,omitempty"`
}

// SQLRegenerationResponse is the POST response.
type SQLRegenerationResponse struct {
	QueryID string `json:"query_id"`
}

// SQLRegenerationResultResponse is the GET response.
type SQLRegenerationResultResponse struct {
	Status   string                     `json:"status"`
	Response *SQLRegenerationResultData `json:"response,omitempty"`
	Error    *AskError                  `json:"error,omitempty"`
}

// SQLRegenerationResultData holds regeneration result.
type SQLRegenerationResultData struct {
	Description string         `json:"description"`
	Steps       []SQLBreakdown `json:"steps"`
}
