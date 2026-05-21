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

// WrenIbis implements core.Engine by calling wren-ibis-server.
type WrenIbis struct {
	endpoint       string
	source         string
	manifest       string
	connectionInfo map[string]any
	httpClient     *http.Client
}

// NewWrenIbis creates a new WrenIbis engine.
func NewWrenIbis(endpoint, source, manifest string, connectionInfo map[string]any) *WrenIbis {
	if connectionInfo == nil {
		connectionInfo = map[string]any{}
	}
	return &WrenIbis{
		endpoint:       endpoint,
		source:         source,
		manifest:       manifest,
		connectionInfo: connectionInfo,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

// ExecuteSQL executes or dry-runs SQL against wren-ibis.
func (e *WrenIbis) ExecuteSQL(ctx context.Context, sql string, opts core.EngineOpts) (*core.EngineResult, error) {
	sql = sqlutil.RemoveLimitStatement(sql)
	apiEndpoint := fmt.Sprintf("%s/v2/connector/%s/query", e.endpoint, e.source)
	if opts.DryRun {
		apiEndpoint += "?dryRun=true&limit=1"
	} else {
		limit := opts.Limit
		if limit == 0 {
			limit = 500
		}
		apiEndpoint += fmt.Sprintf("?limit=%d", limit)
	}

	body := map[string]any{
		"sql":            sql,
		"manifestStr":    e.manifest,
		"connectionInfo": e.connectionInfo,
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", apiEndpoint, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return &core.EngineResult{Success: true}, nil
	}

	var result any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if resp.StatusCode == 200 {
		return &core.EngineResult{Success: true, Data: map[string]any{"result": result}}, nil
	}

	return &core.EngineResult{Success: false, Error: fmt.Sprintf("status %d: %v", resp.StatusCode, result)}, nil
}

func init() {
	provider.RegisterEngine("wren_ibis", func(cfg map[string]any) (core.Engine, error) {
		endpoint, _ := cfg["endpoint"].(string)
		source, _ := cfg["source"].(string)
		manifest, _ := cfg["manifest"].(string)
		var connInfo map[string]any
		if v, ok := cfg["connection_info"].(map[string]any); ok {
			connInfo = v
		}
		return NewWrenIbis(endpoint, source, manifest, connInfo), nil
	})
}
