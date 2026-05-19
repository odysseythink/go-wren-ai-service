package service

import (
	"context"

	"github.com/patrickmn/go-cache"
	"github.com/odysseythink/go-wren-ai-service/internal/model"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/generation"
)

// RelationshipRecommendationService handles relationship-recommendations requests.
type RelationshipRecommendationService struct {
	cache                  *cache.Cache
	relationshipRecommendation *generation.RelationshipRecommendation
}

// NewRelationshipRecommendationService creates a new RelationshipRecommendationService.
func NewRelationshipRecommendationService(c *cache.Cache, relationshipRecommendation *generation.RelationshipRecommendation) *RelationshipRecommendationService {
	return &RelationshipRecommendationService{cache: c, relationshipRecommendation: relationshipRecommendation}
}

// RelationshipRecResult holds the result.
type RelationshipRecResult struct {
	Status   string
	Response any
	Error    *model.AskError
}

// RelationshipRec runs the relationship recommendation pipeline.
func (s *RelationshipRecommendationService) RelationshipRec(ctx context.Context, id string, req *model.RelationshipRecRequest) {
	s.setResult(id, &RelationshipRecResult{Status: "generating"})
	result, err := s.relationshipRecommendation.Run(ctx, &generation.RelationshipRecRequest{MDL: req.MDL})
	if err != nil {
		s.setResult(id, &RelationshipRecResult{Status: "failed", Error: &model.AskError{Code: "OTHERS", Message: err.Error()}})
		return
	}
	res := result.(map[string]any)
	s.setResult(id, &RelationshipRecResult{Status: "finished", Response: res})
}

func (s *RelationshipRecommendationService) setResult(id string, result *RelationshipRecResult) {
	s.cache.Set(id, result, cache.DefaultExpiration)
}

// GetResult retrieves the result.
func (s *RelationshipRecommendationService) GetResult(id string) *RelationshipRecResult {
	if r, found := s.cache.Get(id); found {
		return r.(*RelationshipRecResult)
	}
	return &RelationshipRecResult{Status: "failed", Error: &model.AskError{Code: "OTHERS", Message: "result not found"}}
}
