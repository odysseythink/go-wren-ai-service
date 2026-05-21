package service

import (
	"context"

	"github.com/patrickmn/go-cache"
	"github.com/odysseythink/go-wren-ai-service/internal/model"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/generation"
)

// SQLAnswerService handles sql-answers requests.
type SQLAnswerService struct {
	cache     *cache.Cache
	sqlAnswer *generation.SQLAnswer
}

// NewSQLAnswerService creates a new SQLAnswerService.
func NewSQLAnswerService(c *cache.Cache, sqlAnswer *generation.SQLAnswer) *SQLAnswerService {
	return &SQLAnswerService{cache: c, sqlAnswer: sqlAnswer}
}

// SQLAnswerResult holds the result.
type SQLAnswerResult struct {
	Status   string
	Response *common.SQLAnswerResult
	Error    *model.AskError
}

// SQLAnswer runs the sql-answer pipeline.
func (s *SQLAnswerService) SQLAnswer(ctx context.Context, queryID string, req *model.SQLAnswerRequest) {
	s.setResult(queryID, &SQLAnswerResult{Status: "understanding"})

	result, err := s.sqlAnswer.Run(ctx, &generation.SQLAnswerRequest{
		Query:      req.Query,
		SQL:        req.SQL,
		SQLSummary: req.SQLSummary,
		SQLData:    map[string]any{"rows": []any{}},
	})
	if err != nil {
		s.setResult(queryID, &SQLAnswerResult{Status: "failed", Error: &model.AskError{Code: "OTHERS", Message: err.Error()}})
		return
	}
	answer := result.(*common.SQLAnswerResult)
	s.setResult(queryID, &SQLAnswerResult{Status: "finished", Response: answer})
}

func (s *SQLAnswerService) setResult(queryID string, result *SQLAnswerResult) {
	s.cache.Set(queryID, result, cache.DefaultExpiration)
}

// GetResult retrieves the result.
func (s *SQLAnswerService) GetResult(queryID string) *SQLAnswerResult {
	if r, found := s.cache.Get(queryID); found {
		return r.(*SQLAnswerResult)
	}
	return &SQLAnswerResult{Status: "failed", Error: &model.AskError{Code: "OTHERS", Message: "result not found"}}
}
