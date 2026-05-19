package retrieval

import (
	"context"
	"fmt"
	"sort"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
)

// HistoricalQuestion retrieves similar historical questions from view store.
type HistoricalQuestion struct {
	components core.PipelineComponent
	store      core.DocumentStore
	retriever  core.Retriever
}

// NewHistoricalQuestion creates a new historical question retriever.
func NewHistoricalQuestion(components core.PipelineComponent) *HistoricalQuestion {
	store := components.DocStoreProvider.GetStore(core.StoreOpts{DatasetName: "view_questions"})
	return &HistoricalQuestion{
		components: components,
		store:      store,
		retriever:  components.DocStoreProvider.GetRetriever(store, 10),
	}
}

// HistoricalResult holds formatted historical questions.
type HistoricalResult struct {
	Questions []HistoricalQuestionItem `json:"questions"`
}

// HistoricalQuestionItem is a single historical question result.
type HistoricalQuestionItem struct {
	Question  string `json:"question"`
	Summary   string `json:"summary"`
	Statement string `json:"statement"`
	ViewID    string `json:"viewId"`
}

// Run retrieves and formats historical questions.
func (h *HistoricalQuestion) Run(ctx context.Context, query string, projectID string) (*HistoricalResult, error) {
	textEmbedder, err := h.components.EmbedderProvider.GetTextEmbedder(ctx)
	if err != nil {
		return nil, err
	}
	embedResult, err := textEmbedder.Run(ctx, query)
	if err != nil {
		return nil, err
	}

	filters := map[string]any{}
	if projectID != "" {
		filters["project_id"] = projectID
	}

	result, err := h.retriever.Run(ctx, embedResult.Embedding, filters)
	if err != nil {
		return nil, err
	}

	// Filter by score >= 0.9 and sort descending
	var filtered []core.Document
	for _, doc := range result.Documents {
		if doc.Score >= 0.9 {
			filtered = append(filtered, doc)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Score > filtered[j].Score
	})

	var items []HistoricalQuestionItem
	for _, doc := range filtered {
		items = append(items, HistoricalQuestionItem{
			Question:  doc.Content,
			Summary:   fmt.Sprintf("%v", doc.Meta["summary"]),
			Statement: fmt.Sprintf("%v", doc.Meta["statement"]),
			ViewID:    fmt.Sprintf("%v", doc.Meta["viewId"]),
		})
	}
	return &HistoricalResult{Questions: items}, nil
}
