package service

import (
	"fmt"

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
func NewContainer(components map[string]core.PipelineComponent, cfg *config.Config) *Container {
	queryCache := cache.New(cache.DefaultExpiration, cache.NoExpiration)

	get := func(name string) core.PipelineComponent {
		c, ok := components[name]
		if !ok {
			panic(fmt.Sprintf("pipeline component %q not found", name))
		}
		return c
	}

	indexingPipe := indexing.NewIndexing(get("indexing"), cfg.ColumnIndexingBatchSize)
	retrievalPipe := retrieval.NewRetrieval(get("retrieval"), cfg.TableRetrievalSize, cfg.TableColumnRetrievalSize)
	historicalPipe := retrieval.NewHistoricalQuestion(get("historical_question"))

	sqlGenPipe := generation.NewSQLGeneration(get("sql_generation"))
	sqlCorrectionPipe := generation.NewSQLCorrection(get("sql_correction"))
	sqlSummaryPipe := generation.NewSQLSummary(get("sql_summary"))
	sqlAnswerPipe := generation.NewSQLAnswer(get("sql_answer"))
	sqlBreakdownPipe := generation.NewSQLBreakdown(get("sql_breakdown"))
	sqlExpansionPipe := generation.NewSQLExpansion(get("sql_expansion"))
	sqlRegenerationPipe := generation.NewSQLRegeneration(get("sql_regeneration"))
	sqlExplanationPipe := generation.NewSQLExplanation(get("sql_explanation"))
	followupSQLPipe := generation.NewFollowUpSQLGeneration(get("followup_sql_generation"))
	semanticsDescPipe := generation.NewSemanticsDescription(get("semantics_description"))
	relationshipRecPipe := generation.NewRelationshipRecommendation(get("relationship_recommendation"))

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
