package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/odysseythink/go-wren-ai-service/internal/model"
	"github.com/odysseythink/go-wren-ai-service/internal/service"
)

func newAskHandler(s *service.AskService) chi.Router {
	r := chi.NewRouter()
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		var req model.AskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		queryID := uuid.New().String()
		go s.Ask(r.Context(), queryID, &req)
		json.NewEncoder(w).Encode(model.AskResponse{QueryID: queryID})
	})
	r.Patch("/{query_id}", func(w http.ResponseWriter, r *http.Request) {
		queryID := chi.URLParam(r, "query_id")
		var req model.StopAskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.StopAsk(queryID)
		json.NewEncoder(w).Encode(model.StopAskResponse{QueryID: queryID})
	})
	r.Get("/{query_id}/result", func(w http.ResponseWriter, r *http.Request) {
		queryID := chi.URLParam(r, "query_id")
		result := s.GetResult(queryID)
		var resp model.AskResultResponse
		resp.Status = result.Status
		if result.Response != nil {
			resp.Response = result.Response
		}
		if result.Error != nil {
			resp.Error = result.Error
		}
		json.NewEncoder(w).Encode(resp)
	})
	return r
}
