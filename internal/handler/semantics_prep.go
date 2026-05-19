package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/odysseythink/go-wren-ai-service/internal/model"
	"github.com/odysseythink/go-wren-ai-service/internal/service"
)

func newSemanticsPrepHandler(s *service.SemanticsPreparationService) chi.Router {
	r := chi.NewRouter()
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		var req model.SemanticsPrepRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		go s.SemanticsPrep(r.Context(), &req)
		json.NewEncoder(w).Encode(model.SemanticsPrepResponse{MdlHash: req.MdlHash})
	})
	r.Get("/{mdl_hash}/status", func(w http.ResponseWriter, r *http.Request) {
		mdlHash := chi.URLParam(r, "mdl_hash")
		result := s.GetResult(mdlHash)
		var resp model.SemanticsPrepStatusResponse
		resp.Status = result.Status
		if result.Error != nil {
			resp.Error = result.Error
		}
		json.NewEncoder(w).Encode(resp)
	})
	return r
}
