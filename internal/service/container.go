package service

import (
	"github.com/patrickmn/go-cache"
	"github.com/odysseythink/go-wren-ai-service/internal/config"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/generation"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/indexing"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/retrieval"
)

// Container holds all services and their pipeline dependencies.
type Container struct {
	AskService                   *AskService
	AskDetailsService            *AskDetailsService
	SQLAnswerService             *SQLAnswerService
	SQLExpansionService          *SQLExpansionService
	SQLExplanationService        *SQLExplanationService
	SQLRegenerationService       *SQLRegenerationService
	SemanticsPreparationService  *SemanticsPreparationService
	SemanticsDescriptionService  *SemanticsDescriptionService
	RelationshipRecommendationService *RelationshipRecommendationService
}

// NewContainer initializes all services with pipeline dependencies.
func NewContainer(components core.PipelineComponent, cfg *config.Config) *Container {
	queryCache := cache.New(cache.DefaultExpiration, cache.NoExpiration)

	indexingPipe := indexing.NewIndexing(components, cfg.ColumnIndexingBatchSize)
	retrievalPipe := retrieval.NewRetrieval(components, cfg.TableRetrievalSize, cfg.TableColumnRetrievalSize)
	historicalPipe := retrieval.NewHistoricalQuestion(components)

	sqlGenPipe := generation.NewSQLGeneration(components)
	sqlCorrectionPipe := generation.NewSQLCorrection(components)
	sqlSummaryPipe := generation.NewSQLSummary(components)
	sqlAnswerPipe := generation.NewSQLAnswer(components)
	sqlBreakdownPipe := generation.NewSQLBreakdown(components)
	sqlExpansionPipe := generation.NewSQLExpansion(components)
	sqlRegenerationPipe := generation.NewSQLRegeneration(components)
	sqlExplanationPipe := generation.NewSQLExplanation(components)
	followupSQLPipe := generation.NewFollowUpSQLGeneration(components)
	semanticsDescPipe := generation.NewSemanticsDescription(components)
	relationshipRecPipe := generation.NewRelationshipRecommendation(components)

	return &Container{
		AskService:                   NewAskService(queryCache, retrievalPipe, historicalPipe, sqlGenPipe, sqlCorrectionPipe, sqlSummaryPipe, followupSQLPipe),
		AskDetailsService:            NewAskDetailsService(queryCache, sqlBreakdownPipe),
		SQLAnswerService:             NewSQLAnswerService(queryCache, sqlAnswerPipe),
		SQLExpansionService:          NewSQLExpansionService(queryCache, retrievalPipe, sqlExpansionPipe, sqlCorrectionPipe, sqlSummaryPipe),
		SQLExplanationService:        NewSQLExplanationService(queryCache, sqlExplanationPipe),
		SQLRegenerationService:       NewSQLRegenerationService(queryCache, sqlRegenerationPipe),
		SemanticsPreparationService:  NewSemanticsPreparationService(queryCache, indexingPipe),
		SemanticsDescriptionService:  NewSemanticsDescriptionService(queryCache, semanticsDescPipe),
		RelationshipRecommendationService: NewRelationshipRecommendationService(queryCache, relationshipRecPipe),
	}
}
