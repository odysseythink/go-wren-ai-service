package service

import (
	"context"

	"github.com/patrickmn/go-cache"
	"github.com/odysseythink/go-wren-ai-service/internal/model"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/generation"
)

// SemanticsDescriptionService handles semantics-descriptions requests.
type SemanticsDescriptionService struct {
	cache              *cache.Cache
	semanticsDescription *generation.SemanticsDescription
}

// NewSemanticsDescriptionService creates a new SemanticsDescriptionService.
func NewSemanticsDescriptionService(c *cache.Cache, semanticsDescription *generation.SemanticsDescription) *SemanticsDescriptionService {
	return &SemanticsDescriptionService{cache: c, semanticsDescription: semanticsDescription}
}

// SemanticsDescResult holds the result.
type SemanticsDescResult struct {
	Status   string
	Response []model.ModelDescItem
	Error    *model.AskError
}

// SemanticsDesc runs the semantics description pipeline.
func (s *SemanticsDescriptionService) SemanticsDesc(ctx context.Context, id string, req *model.SemanticsDescRequest) {
	s.setResult(id, &SemanticsDescResult{Status: "generating"})
	result, err := s.semanticsDescription.Run(ctx, &generation.SemanticsDescriptionRequest{
		UserPrompt:     req.UserPrompt,
		SelectedModels: req.SelectedModels,
		MDL:            req.MDL,
	})
	if err != nil {
		s.setResult(id, &SemanticsDescResult{Status: "failed", Error: &model.AskError{Code: "OTHERS", Message: err.Error()}})
		return
	}
	res := result.(map[string]any)
	s.setResult(id, &SemanticsDescResult{Status: "finished", Response: s.formatResponse(res)})
}

func (s *SemanticsDescriptionService) formatResponse(res map[string]any) []model.ModelDescItem {
	return []model.ModelDescItem{}
}

func (s *SemanticsDescriptionService) setResult(id string, result *SemanticsDescResult) {
	s.cache.Set(id, result, cache.DefaultExpiration)
}

// GetResult retrieves the result.
func (s *SemanticsDescriptionService) GetResult(id string) *SemanticsDescResult {
	if r, found := s.cache.Get(id); found {
		return r.(*SemanticsDescResult)
	}
	return &SemanticsDescResult{Status: "failed", Error: &model.AskError{Code: "OTHERS", Message: "result not found"}}
}
