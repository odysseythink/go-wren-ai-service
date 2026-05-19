package service

import (
	"context"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/odysseythink/go-wren-ai-service/internal/model"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/generation"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/retrieval"
)

// SQLExpansionService handles sql-expansions requests.
type SQLExpansionService struct {
	cache         *cache.Cache
	retrieval     *retrieval.Retrieval
	sqlExpansion  *generation.SQLExpansion
	sqlCorrection *generation.SQLCorrection
	sqlSummary    *generation.SQLSummary
}

// NewSQLExpansionService creates a new SQLExpansionService.
func NewSQLExpansionService(c *cache.Cache, retrieval *retrieval.Retrieval, expansion *generation.SQLExpansion, correction *generation.SQLCorrection, summary *generation.SQLSummary) *SQLExpansionService {
	return &SQLExpansionService{cache: c, retrieval: retrieval, sqlExpansion: expansion, sqlCorrection: correction, sqlSummary: summary}
}

// SQLExpansionResult holds the result.
type SQLExpansionResult struct {
	Status   string
	Response *model.SQLExpansionResultData
	Error    *model.AskError
}

// SQLExpansion runs the sql-expansion pipeline.
func (s *SQLExpansionService) SQLExpansion(ctx context.Context, req *model.SQLExpansionRequest) {
	queryID := fmt.Sprintf("sql-expansion-%d", time.Now().UnixNano())
	s.setResult(queryID, &SQLExpansionResult{Status: "understanding"})

	query := req.Query
	if req.History != nil && req.History.Summary != "" {
		query = req.History.Summary + " " + query
	}

	s.setResult(queryID, &SQLExpansionResult{Status: "searching"})
	retResult, err := s.retrieval.Run(ctx, &retrieval.RetrievalRequest{Query: query, ProjectID: ptrStr(req.ProjectID)})
	if err != nil {
		s.setResult(queryID, &SQLExpansionResult{Status: "failed", Error: &model.AskError{Code: "NO_RELEVANT_DATA", Message: err.Error()}})
		return
	}
	retDocs := retResult.(*retrieval.RetrievalResult)

	s.setResult(queryID, &SQLExpansionResult{Status: "generating"})
	genResult, err := s.sqlExpansion.Run(ctx, &generation.SQLExpansionRequest{
		SQL:       req.SQL,
		Documents: retDocs.Documents,
		Query:     req.Query,
		ProjectID: ptrStr(req.ProjectID),
	})
	if err != nil {
		s.setResult(queryID, &SQLExpansionResult{Status: "failed", Error: &model.AskError{Code: "NO_RELEVANT_SQL", Message: err.Error()}})
		return
	}
	gen := genResult.(*common.SQLGenResult)
	if len(gen.ValidGenerationResults) == 0 {
		s.setResult(queryID, &SQLExpansionResult{Status: "failed", Error: &model.AskError{Code: "NO_RELEVANT_SQL", Message: "no valid SQL generated"}})
		return
	}

	sql, _ := gen.ValidGenerationResults[0]["sql"].(string)
	s.setResult(queryID, &SQLExpansionResult{
		Status:   "finished",
		Response: &model.SQLExpansionResultData{Description: query, Steps: []model.SQLBreakdown{{SQL: sql, Summary: "", CTEName: ""}}},
	})
}

func (s *SQLExpansionService) setResult(queryID string, result *SQLExpansionResult) {
	s.cache.Set(queryID, result, cache.DefaultExpiration)
}

// GetResult retrieves the result.
func (s *SQLExpansionService) GetResult(queryID string) *SQLExpansionResult {
	if r, found := s.cache.Get(queryID); found {
		return r.(*SQLExpansionResult)
	}
	return &SQLExpansionResult{Status: "failed", Error: &model.AskError{Code: "OTHERS", Message: "result not found"}}
}

// StopSQLExpansion marks a query as stopped.
func (s *SQLExpansionService) StopSQLExpansion(queryID string) {
	s.setResult(queryID, &SQLExpansionResult{Status: "stopped"})
}
