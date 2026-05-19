package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/odysseythink/go-wren-ai-service/internal/service"
)

// NewRouter creates a chi router with all API routes mounted.
func NewRouter(container *service.Container) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// API v1
	r.Route("/v1", func(r chi.Router) {
		r.Mount("/asks", newAskHandler(container.AskService))
		r.Mount("/ask-details", newAskDetailsHandler(container.AskDetailsService))
		r.Mount("/sql-answers", newSQLAnswerHandler(container.SQLAnswerService))
		r.Mount("/sql-expansions", newSQLExpansionHandler(container.SQLExpansionService))
		r.Mount("/sql-explanations", newSQLExplanationHandler(container.SQLExplanationService))
		r.Mount("/sql-regenerations", newSQLRegenerationHandler(container.SQLRegenerationService))
		r.Mount("/semantics-preparations", newSemanticsPrepHandler(container.SemanticsPreparationService))
		r.Mount("/semantics-descriptions", newSemanticsDescHandler(container.SemanticsDescriptionService))
		r.Mount("/relationship-recommendations", newRelationshipRecHandler(container.RelationshipRecommendationService))
	})

	return r
}
