package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/provider"
	"github.com/odysseythink/go-wren-ai-service/pkg/sqlutil"
)

// WrenEngine implements core.Engine by calling wren-engine.
type WrenEngine struct {
	endpoint   string
	manifest   map[string]any
	httpClient *http.Client
}

// NewWrenEngine creates a new WrenEngine.
func NewWrenEngine(endpoint string, manifest map[string]any) *WrenEngine {
	if manifest == nil {
		manifest = map[string]any{}
	}
	return &WrenEngine{
		endpoint:   endpoint,
		manifest:   manifest,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ExecuteSQL executes or dry-runs SQL against wren-engine.
func (e *WrenEngine) ExecuteSQL(ctx context.Context, sql string, opts core.EngineOpts) (*core.EngineResult, error) {
	sql = sqlutil.RemoveLimitStatement(sql)
	apiEndpoint := e.endpoint + "/v1/mdl/dry-run"
	if !opts.DryRun {
		apiEndpoint = e.endpoint + "/v1/mdl/preview"
	}

	limit := opts.Limit
	if limit == 0 {
		limit = 1
		if !opts.DryRun {
			limit = 500
		}
	}

	body := map[string]any{
		"manifest": e.manifest,
		"sql":      sql,
		"limit":    limit,
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "GET", apiEndpoint, bytes.NewReader(b))
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

	if resp.StatusCode == 200 {
		return &core.EngineResult{Success: true, Data: result}, nil
	}

	return &core.EngineResult{Success: false, Error: fmt.Sprintf("status %d: %v", resp.StatusCode, result)}, nil
}

func init() {
	provider.RegisterEngine("wren_engine", func(cfg map[string]any) (core.Engine, error) {
		endpoint, _ := cfg["endpoint"].(string)
		var manifest map[string]any
		if v, ok := cfg["manifest"].(map[string]any); ok {
			manifest = v
		}
		return NewWrenEngine(endpoint, manifest), nil
	})
}
