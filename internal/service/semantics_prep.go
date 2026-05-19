package service

import (
	"context"

	"github.com/patrickmn/go-cache"
	"github.com/odysseythink/go-wren-ai-service/internal/model"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/indexing"
)

// SemanticsPreparationService handles semantics-preparations requests.
type SemanticsPreparationService struct {
	cache    *cache.Cache
	indexing *indexing.Indexing
}

// NewSemanticsPreparationService creates a new SemanticsPreparationService.
func NewSemanticsPreparationService(c *cache.Cache, indexing *indexing.Indexing) *SemanticsPreparationService {
	return &SemanticsPreparationService{cache: c, indexing: indexing}
}

// SemanticsPrepResult holds the result.
type SemanticsPrepResult struct {
	Status string
	Error  *model.AskError
}

// SemanticsPrep runs the indexing pipeline.
func (s *SemanticsPreparationService) SemanticsPrep(ctx context.Context, req *model.SemanticsPrepRequest) {
	s.setResult(req.MdlHash, &SemanticsPrepResult{Status: "indexing"})
	_, err := s.indexing.Run(ctx, &indexing.IndexingRequest{MDL: req.MDL, ProjectID: ptrStr(req.ProjectID)})
	if err != nil {
		s.setResult(req.MdlHash, &SemanticsPrepResult{Status: "failed", Error: &model.AskError{Code: "OTHERS", Message: err.Error()}})
		return
	}
	s.setResult(req.MdlHash, &SemanticsPrepResult{Status: "finished"})
}

func (s *SemanticsPreparationService) setResult(mdlHash string, result *SemanticsPrepResult) {
	s.cache.Set(mdlHash, result, cache.DefaultExpiration)
}

// GetResult retrieves the result.
func (s *SemanticsPreparationService) GetResult(mdlHash string) *SemanticsPrepResult {
	if r, found := s.cache.Get(mdlHash); found {
		return r.(*SemanticsPrepResult)
	}
	return &SemanticsPrepResult{Status: "failed", Error: &model.AskError{Code: "OTHERS", Message: "result not found"}}
}
