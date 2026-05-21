package service

import (
	"context"
	"fmt"

	"github.com/patrickmn/go-cache"
	"github.com/odysseythink/go-wren-ai-service/internal/model"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/generation"
)

// SQLRegenerationService handles sql-regenerations requests.
type SQLRegenerationService struct {
	cache           *cache.Cache
	sqlRegeneration *generation.SQLRegeneration
}

// NewSQLRegenerationService creates a new SQLRegenerationService.
func NewSQLRegenerationService(c *cache.Cache, sqlRegeneration *generation.SQLRegeneration) *SQLRegenerationService {
	return &SQLRegenerationService{cache: c, sqlRegeneration: sqlRegeneration}
}

// SQLRegenerationResult holds the result.
type SQLRegenerationResult struct {
	Status   string
	Response *model.SQLRegenerationResultData
	Error    *model.AskError
}

// SQLRegeneration runs the sql-regeneration pipeline.
func (s *SQLRegenerationService) SQLRegeneration(ctx context.Context, queryID string, req *model.SQLRegenerationRequest) {
	s.setResult(queryID, &SQLRegenerationResult{Status: "understanding"})

	var steps []map[string]any
	for _, step := range req.Steps {
		steps = append(steps, map[string]any{"sql": step.SQL, "summary": step.Summary, "cte_name": step.CTEName})
	}

	s.setResult(queryID, &SQLRegenerationResult{Status: "generating"})
	result, err := s.sqlRegeneration.Run(ctx, &generation.SQLRegenerationRequest{
		Description: req.Description,
		Steps:       steps,
		ProjectID:   ptrStr(req.ProjectID),
	})
	if err != nil {
		s.setResult(queryID, &SQLRegenerationResult{Status: "failed", Error: &model.AskError{Code: "NO_RELEVANT_SQL", Message: "SQL is not executable"}})
		return
	}
	breakdown := result.(*common.SQLBreakdownResult)
	if len(breakdown.Steps) == 0 {
		s.setResult(queryID, &SQLRegenerationResult{Status: "failed", Error: &model.AskError{Code: "NO_RELEVANT_SQL", Message: "SQL is not executable"}})
		return
	}
	var outSteps []model.SQLBreakdown
	for _, step := range breakdown.Steps {
		outSteps = append(outSteps, model.SQLBreakdown{
			SQL:     fmt.Sprintf("%v", step["sql"]),
			Summary: fmt.Sprintf("%v", step["summary"]),
			CTEName: fmt.Sprintf("%v", step["cte_name"]),
		})
	}
	s.setResult(queryID, &SQLRegenerationResult{
		Status:   "finished",
		Response: &model.SQLRegenerationResultData{Description: breakdown.Description, Steps: outSteps},
	})
}

func (s *SQLRegenerationService) setResult(queryID string, result *SQLRegenerationResult) {
	s.cache.Set(queryID, result, cache.DefaultExpiration)
}

// GetResult retrieves the result.
func (s *SQLRegenerationService) GetResult(queryID string) *SQLRegenerationResult {
	if r, found := s.cache.Get(queryID); found {
		return r.(*SQLRegenerationResult)
	}
	return &SQLRegenerationResult{Status: "failed", Error: &model.AskError{Code: "OTHERS", Message: "result not found"}}
}
