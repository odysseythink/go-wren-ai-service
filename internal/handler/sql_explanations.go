package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/odysseythink/go-wren-ai-service/internal/model"
	"github.com/odysseythink/go-wren-ai-service/internal/service"
)

func newSQLExplanationHandler(s *service.SQLExplanationService) chi.Router {
	r := chi.NewRouter()
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		var req model.SQLExplanationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		queryID := uuid.New().String()
		go s.SQLExplanation(r.Context(), &req)
		json.NewEncoder(w).Encode(model.SQLExplanationResponse{QueryID: queryID})
	})
	r.Get("/{query_id}/result", func(w http.ResponseWriter, r *http.Request) {
		queryID := chi.URLParam(r, "query_id")
		result := s.GetResult(queryID)
		var resp model.SQLExplanationResultResponse
		resp.Status = result.Status
		resp.Response = result.Response
		if result.Error != nil {
			resp.Error = result.Error
		}
		json.NewEncoder(w).Encode(resp)
	})
	return r
}
