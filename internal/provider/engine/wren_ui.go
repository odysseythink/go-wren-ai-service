package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/pkg/sqlutil"
)

// WrenUI implements core.Engine by calling wren-ui's GraphQL previewSql mutation.
type WrenUI struct {
	endpoint   string
	httpClient *http.Client
}

// NewWrenUI creates a new WrenUI engine.
func NewWrenUI(endpoint string) *WrenUI {
	return &WrenUI{
		endpoint:   endpoint,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ExecuteSQL executes or dry-runs SQL against wren-ui.
func (e *WrenUI) ExecuteSQL(ctx context.Context, sql string, opts core.EngineOpts) (*core.EngineResult, error) {
	sql = sqlutil.RemoveLimitStatement(sql)
	data := map[string]any{
		"sql":       sql,
		"projectId": opts.ProjectID,
	}
	if opts.DryRun {
		data["dryRun"] = true
		data["limit"] = 1
	} else {
		data["limit"] = opts.Limit
		if data["limit"] == 0 {
			data["limit"] = 500
		}
	}

	body := map[string]any{
		"query":     "mutation PreviewSql($data: PreviewSQLDataInput) { previewSql(data: $data) }",
		"variables": map[string]any{"data": data},
	}

	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", e.endpoint+"/api/graphql", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if data, ok := result["data"]; ok {
		return &core.EngineResult{Success: true, Data: map[string]any{"data": data}}, nil
	}

	errors := result["errors"].([]any)
	msg := "Unknown error"
	if len(errors) > 0 {
		if errMap, ok := errors[0].(map[string]any); ok {
			if m, ok := errMap["message"].(string); ok {
				msg = m
			}
		}
	}
	return &core.EngineResult{Success: false, Error: msg}, nil
}
