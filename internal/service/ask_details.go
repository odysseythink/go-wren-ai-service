package service

import (
	"context"
	"fmt"

	"github.com/patrickmn/go-cache"
	"github.com/odysseythink/go-wren-ai-service/internal/model"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/generation"
)

// AskDetailsService handles ask-details requests.
type AskDetailsService struct {
	cache       *cache.Cache
	sqlBreakdown *generation.SQLBreakdown
}

// NewAskDetailsService creates a new AskDetailsService.
func NewAskDetailsService(c *cache.Cache, sqlBreakdown *generation.SQLBreakdown) *AskDetailsService {
	return &AskDetailsService{cache: c, sqlBreakdown: sqlBreakdown}
}

// AskDetailsResult holds the result.
type AskDetailsResult struct {
	Status   string
	Response *model.AskDetailsResultData
	Error    *model.AskError
}

// AskDetails runs the ask-details pipeline.
func (s *AskDetailsService) AskDetails(ctx context.Context, queryID string, req *model.AskDetailsRequest) {
	s.setResult(queryID, &AskDetailsResult{Status: "understanding"})

	result, err := s.sqlBreakdown.Run(ctx, &generation.SQLBreakdownRequest{
		Query:     req.Query,
		SQL:       req.SQL,
		Language:  "English",
		ProjectID: ptrStr(req.ProjectID),
	})
	if err != nil {
		s.setResult(queryID, &AskDetailsResult{Status: "failed", Error: &model.AskError{Code: "NO_RELEVANT_SQL", Message: err.Error()}})
		return
	}
	breakdown := result.(*common.SQLBreakdownResult)
	var steps []model.SQLBreakdown
	for _, step := range breakdown.Steps {
		steps = append(steps, model.SQLBreakdown{
			SQL:     fmt.Sprintf("%v", step["sql"]),
			Summary: fmt.Sprintf("%v", step["summary"]),
			CTEName: fmt.Sprintf("%v", step["cte_name"]),
		})
	}
	if len(steps) == 0 {
		steps = append(steps, model.SQLBreakdown{SQL: req.SQL, Summary: req.Summary, CTEName: ""})
	}
	s.setResult(queryID, &AskDetailsResult{
		Status:   "finished",
		Response: &model.AskDetailsResultData{Description: breakdown.Description, Steps: steps},
	})
}

func (s *AskDetailsService) setResult(queryID string, result *AskDetailsResult) {
	s.cache.Set(queryID, result, cache.DefaultExpiration)
}

// GetResult retrieves the result.
func (s *AskDetailsService) GetResult(queryID string) *AskDetailsResult {
	if r, found := s.cache.Get(queryID); found {
		return r.(*AskDetailsResult)
	}
	return &AskDetailsResult{Status: "failed", Error: &model.AskError{Code: "OTHERS", Message: "result not found"}}
}
