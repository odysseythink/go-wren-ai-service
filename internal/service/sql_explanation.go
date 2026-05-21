package service

import (
	"context"

	"github.com/patrickmn/go-cache"
	"github.com/odysseythink/go-wren-ai-service/internal/model"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/generation"
)

// SQLExplanationService handles sql-explanations requests.
type SQLExplanationService struct {
	cache          *cache.Cache
	sqlExplanation *generation.SQLExplanation
}

// NewSQLExplanationService creates a new SQLExplanationService.
func NewSQLExplanationService(c *cache.Cache, sqlExplanation *generation.SQLExplanation) *SQLExplanationService {
	return &SQLExplanationService{cache: c, sqlExplanation: sqlExplanation}
}

// SQLExplanationResult holds the result.
type SQLExplanationResult struct {
	Status   string
	Response [][]model.SQLExplanationItem
	Error    *model.AskError
}

// SQLExplanation runs the sql-explanation pipeline.
func (s *SQLExplanationService) SQLExplanation(ctx context.Context, queryID string, req *model.SQLExplanationRequest) {
	s.setResult(queryID, &SQLExplanationResult{Status: "understanding"})

	var results [][]model.SQLExplanationItem
	for _, step := range req.StepsWithAnalysisResults {
		s.setResult(queryID, &SQLExplanationResult{Status: "generating"})
		result, err := s.sqlExplanation.Run(ctx, &generation.SQLExplanationRequest{
			Question:           req.Question,
			SQL:                step.SQL,
			SQLSummary:         step.Summary,
			SQLAnalysisResults: toMapSlice(step.SQLAnalysisResults),
		})
		if err != nil {
			continue
		}
		items := result.([]model.SQLExplanationItem)
		results = append(results, items)
	}
	if len(results) == 0 {
		s.setResult(queryID, &SQLExplanationResult{Status: "failed", Error: &model.AskError{Code: "OTHERS", Message: "No SQL explanation is found"}})
		return
	}
	s.setResult(queryID, &SQLExplanationResult{Status: "finished", Response: results})
}

func toMapSlice(items []any) []map[string]any {
	var results []map[string]any
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			results = append(results, m)
		}
	}
	return results
}

func (s *SQLExplanationService) setResult(queryID string, result *SQLExplanationResult) {
	s.cache.Set(queryID, result, cache.DefaultExpiration)
}

// GetResult retrieves the result.
func (s *SQLExplanationService) GetResult(queryID string) *SQLExplanationResult {
	if r, found := s.cache.Get(queryID); found {
		return r.(*SQLExplanationResult)
	}
	return &SQLExplanationResult{Status: "failed", Error: &model.AskError{Code: "OTHERS", Message: "result not found"}}
}
