package model

// SemanticsPrepRequest is the POST /v1/semantics-preparations request.
type SemanticsPrepRequest struct {
	MDL       string  `json:"mdl"`
	MdlHash   string  `json:"mdl_hash"`
	ProjectID *string `json:"project_id,omitempty"`
	UserID    *string `json:"user_id,omitempty"`
}

// SemanticsPrepResponse is the POST response.
type SemanticsPrepResponse struct {
	MdlHash string `json:"mdl_hash"`
}

// SemanticsPrepStatusResponse is the GET response.
type SemanticsPrepStatusResponse struct {
	Status string    `json:"status"`
	Error  *AskError `json:"error,omitempty"`
}

// SemanticsDescRequest is the POST /v1/semantics-descriptions request.
type SemanticsDescRequest struct {
	SelectedModels []string `json:"selected_models"`
	UserPrompt     string   `json:"user_prompt"`
	MDL            string   `json:"mdl"`
}

// SemanticsDescResponse is the POST response.
type SemanticsDescResponse struct {
	ID string `json:"id"`
}

// SemanticsDescGetResponse is the GET response.
type SemanticsDescGetResponse struct {
	ID       string          `json:"id"`
	Status   string          `json:"status"`
	Response []ModelDescItem `json:"response,omitempty"`
	Error    *AskError       `json:"error,omitempty"`
}

// ModelDescItem describes a model with its columns.
type ModelDescItem struct {
	Name        string       `json:"name"`
	Columns     []ColumnDesc `json:"columns"`
	Description string       `json:"description"`
}

// ColumnDesc describes a column.
type ColumnDesc struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// RelationshipRecRequest is the POST /v1/relationship-recommendations request.
type RelationshipRecRequest struct {
	MDL string `json:"mdl"`
}

// RelationshipRecResponse is the POST response.
type RelationshipRecResponse struct {
	ID string `json:"id"`
}

// RelationshipRecGetResponse is the GET response.
type RelationshipRecGetResponse struct {
	ID       string    `json:"id"`
	Status   string    `json:"status"`
	Response any       `json:"response,omitempty"`
	Error    *AskError `json:"error,omitempty"`
}
