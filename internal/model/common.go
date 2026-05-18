package model

// AskError represents an error returned in API responses.
type AskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
