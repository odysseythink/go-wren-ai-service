package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/odysseythink/go-wren-ai-service/internal/model"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/generation"
	"github.com/odysseythink/go-wren-ai-service/pkg/sqlutil"
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
	Response [][]model.ExplanationItem
	Error    *model.AskError
}

// SQLExplanation runs the sql-explanation pipeline.
func (s *SQLExplanationService) SQLExplanation(ctx context.Context, req *model.SQLExplanationRequest) {
	queryID := fmt.Sprintf("sql-explanation-%d", time.Now().UnixNano())
	s.setResult(queryID, &SQLExplanationResult{Status: "understanding"})

	var results [][]model.ExplanationItem
	for _, step := range req.StepsWithAnalysisResults {
		s.setResult(queryID, &SQLExplanationResult{Status: "generating"})
		result, err := s.sqlExplanation.Run(ctx, &generation.SQLExplanationRequest{
			Question:         req.Question,
			SQL:              step.SQL,
			SQLSummary:       step.Summary,
			SQLAnalysisResult: map[string]any{"results": step.SQLAnalysisResults},
		})
		if err != nil {
			continue
		}
		res := result.(map[string]any)
		if exps, ok := res["explanations"].(string); ok {
			cleaned := sqlutil.CleanGenerationResult(exps)
			var parsed []model.ExplanationItem
			_ = json.Unmarshal([]byte(cleaned), &parsed)
			results = append(results, parsed)
		} else {
			results = append(results, []model.ExplanationItem{})
		}
	}
	if len(results) == 0 {
		s.setResult(queryID, &SQLExplanationResult{Status: "failed", Error: &model.AskError{Code: "OTHERS", Message: "No SQL explanation is found"}})
		return
	}
	s.setResult(queryID, &SQLExplanationResult{Status: "finished", Response: results})
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
