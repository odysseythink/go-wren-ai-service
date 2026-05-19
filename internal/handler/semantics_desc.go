package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/odysseythink/go-wren-ai-service/internal/model"
	"github.com/odysseythink/go-wren-ai-service/internal/service"
)

func newSemanticsDescHandler(s *service.SemanticsDescriptionService) chi.Router {
	r := chi.NewRouter()
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		var req model.SemanticsDescRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id := uuid.New().String()
		go s.SemanticsDesc(r.Context(), id, &req)
		json.NewEncoder(w).Encode(model.SemanticsDescResponse{ID: id})
	})
	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		result := s.GetResult(id)
		var resp model.SemanticsDescGetResponse
		resp.ID = id
		resp.Status = result.Status
		resp.Response = result.Response
		if result.Error != nil {
			resp.Error = result.Error
		}
		json.NewEncoder(w).Encode(resp)
	})
	return r
}
