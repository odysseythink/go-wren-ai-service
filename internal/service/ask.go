package service

import (
	"context"

	"github.com/patrickmn/go-cache"
	"github.com/odysseythink/go-wren-ai-service/internal/model"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/generation"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/retrieval"
)

// AskService handles ask requests.
type AskService struct {
	cache             *cache.Cache
	retrieval         *retrieval.Retrieval
	historical        *retrieval.HistoricalQuestion
	sqlGeneration     *generation.SQLGeneration
	sqlCorrection     *generation.SQLCorrection
	sqlSummary        *generation.SQLSummary
	followupSQL       *generation.FollowUpSQLGeneration
}

// NewAskService creates a new AskService.
func NewAskService(
	c *cache.Cache,
	retrieval *retrieval.Retrieval,
	historical *retrieval.HistoricalQuestion,
	sqlGen *generation.SQLGeneration,
	sqlCorr *generation.SQLCorrection,
	sqlSum *generation.SQLSummary,
	followup *generation.FollowUpSQLGeneration,
) *AskService {
	return &AskService{
		cache:         c,
		retrieval:     retrieval,
		historical:    historical,
		sqlGeneration: sqlGen,
		sqlCorrection: sqlCorr,
		sqlSummary:    sqlSum,
		followupSQL:   followup,
	}
}

// AskResult holds the result for a query.
type AskResult struct {
	Status   string
	Response []model.AskResult
	Error    *model.AskError
}

// Ask runs the full ask pipeline asynchronously.
func (s *AskService) Ask(ctx context.Context, queryID string, req *model.AskRequest) {
	s.setResult(queryID, &AskResult{Status: "understanding"})

	query := req.Query
	if req.History != nil && req.History.Summary != "" {
		query = req.History.Summary + " " + query
	}

	// Retrieval
	s.setResult(queryID, &AskResult{Status: "searching"})
	retResult, err := s.retrieval.Run(ctx, &retrieval.RetrievalRequest{Query: query, ProjectID: ptrStr(req.ProjectID)})
	if err != nil {
		s.setResult(queryID, &AskResult{Status: "failed", Error: &model.AskError{Code: "NO_RELEVANT_DATA", Message: err.Error()}})
		return
	}
	retDocs := retResult.(*retrieval.RetrievalResult)
	if len(retDocs.Documents) == 0 {
		s.setResult(queryID, &AskResult{Status: "failed", Error: &model.AskError{Code: "NO_RELEVANT_DATA", Message: "no relevant data"}})
		return
	}

	// Historical questions
	s.setResult(queryID, &AskResult{Status: "generating"})
	histResult, _ := s.historical.Run(ctx, query, ptrStr(req.ProjectID))
	var viewResult *model.AskResult
	if histResult != nil {
		hist := histResult
		if len(hist.Questions) > 0 {
			viewResult = &model.AskResult{
				SQL:     hist.Questions[0].Statement,
				Summary: hist.Questions[0].Summary,
				Type:    "view",
				ViewID:  &hist.Questions[0].ViewID,
			}
		}
	}

	// SQL Generation
	var genInput any
	if req.History != nil {
		genInput = &generation.FollowUpSQLRequest{
			Query:     req.Query,
			Documents: retDocs.Documents,
			History:   req.History,
			ProjectID: ptrStr(req.ProjectID),
		}
	} else {
		genInput = &generation.SQLGenerationRequest{
			Query:     req.Query,
			Documents: retDocs.Documents,
			ProjectID: ptrStr(req.ProjectID),
		}
	}
	genResult, err := s.sqlGeneration.Run(ctx, genInput)
	if err != nil {
		s.setResult(queryID, &AskResult{Status: "failed", Error: &model.AskError{Code: "OTHERS", Message: err.Error()}})
		return
	}
	gen := genResult.(*common.SQLGenResult)

	// SQL Summary for valid results
	var validSQLs []string
	for _, v := range gen.ValidGenerationResults {
		if sql, ok := v["sql"].(string); ok {
			validSQLs = append(validSQLs, sql)
		}
	}
	var summaries []map[string]string
	if len(validSQLs) > 0 {
		sumResult, _ := s.sqlSummary.Run(ctx, &generation.SQLSummaryRequest{
			Query:    req.Query,
			SQLs:     validSQLs,
			Language: req.Configurations.Language,
		})
		if sumResult != nil {
			sums := sumResult.(*common.SQLSummaryResult)
			summaries = sums.Results
		}
	}

	// SQL Correction for invalid dry-run results
	var corrected []map[string]any
	var invalidForCorrection []map[string]any
	for _, inv := range gen.InvalidGenerationResults {
		if inv["type"] == "DRY_RUN" {
			invalidForCorrection = append(invalidForCorrection, inv)
		}
	}
	if len(invalidForCorrection) > 0 {
		corrResult, _ := s.sqlCorrection.Run(ctx, &generation.SQLCorrectionRequest{
			Documents:                retDocs.Documents,
			InvalidGenerationResults: invalidForCorrection,
			ProjectID:                ptrStr(req.ProjectID),
		})
		if corrResult != nil {
			corr := corrResult.(*common.SQLGenResult)
			corrected = corr.ValidGenerationResults
		}
	}

	// Build final results
	var results []model.AskResult
	if viewResult != nil {
		results = append(results, *viewResult)
	}
	for i, v := range gen.ValidGenerationResults {
		sql, _ := v["sql"].(string)
		summary := ""
		if i < len(summaries) {
			summary = summaries[i]["summary"]
		}
		results = append(results, model.AskResult{SQL: sql, Summary: summary, Type: "llm"})
	}
	for _, c := range corrected {
		sql, _ := c["sql"].(string)
		results = append(results, model.AskResult{SQL: sql, Summary: "", Type: "llm"})
	}

	if len(results) == 0 {
		s.setResult(queryID, &AskResult{Status: "failed", Error: &model.AskError{Code: "NO_RELEVANT_SQL", Message: "no valid SQL generated"}})
		return
	}
	if len(results) > 3 {
		results = results[:3]
	}
	s.setResult(queryID, &AskResult{Status: "finished", Response: results})
}

func (s *AskService) setResult(queryID string, result *AskResult) {
	s.cache.Set(queryID, result, cache.DefaultExpiration)
}

// GetResult retrieves the result for a query.
func (s *AskService) GetResult(queryID string) *AskResult {
	if r, found := s.cache.Get(queryID); found {
		return r.(*AskResult)
	}
	return &AskResult{Status: "failed", Error: &model.AskError{Code: "OTHERS", Message: "result not found"}}
}

// StopAsk marks a query as stopped.
func (s *AskService) StopAsk(queryID string) {
	s.setResult(queryID, &AskResult{Status: "stopped"})
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
