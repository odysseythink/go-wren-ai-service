package docstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

// QdrantProvider implements core.DocStoreProvider using Qdrant's REST API.
type QdrantProvider struct {
	host         string
	apiKey       string
	embeddingDim int
	timeout      int
	httpClient   *http.Client
}

// NewQdrantProvider creates a new Qdrant provider.
func NewQdrantProvider(host, apiKey string, embeddingDim, timeout int) *QdrantProvider {
	if timeout <= 0 {
		timeout = 120
	}
	return &QdrantProvider{
		host:         host,
		apiKey:       apiKey,
		embeddingDim: embeddingDim,
		timeout:      timeout,
		httpClient:   &http.Client{Timeout: time.Duration(timeout) * time.Second},
	}
}

// GetStore returns a DocumentStore for the given collection.
func (p *QdrantProvider) GetStore(opts core.StoreOpts) core.DocumentStore {
	return &QdrantDocumentStore{
		host:         p.host,
		apiKey:       p.apiKey,
		collection:   opts.DatasetName,
		embeddingDim: p.embeddingDim,
		httpClient:   p.httpClient,
	}
}

// GetRetriever returns a Retriever for the given store.
func (p *QdrantProvider) GetRetriever(store core.DocumentStore, topK int) core.Retriever {
	return &QdrantRetriever{
		store: store,
		topK:  topK,
	}
}

// QdrantDocumentStore implements core.DocumentStore using Qdrant REST API.
type QdrantDocumentStore struct {
	host         string
	apiKey       string
	collection   string
	embeddingDim int
	httpClient   *http.Client
}

// WriteDocuments upserts documents into Qdrant.
func (s *QdrantDocumentStore) WriteDocuments(ctx context.Context, docs []core.Document, policy core.WritePolicy) (int, error) {
	if err := s.ensureCollection(ctx); err != nil {
		return 0, fmt.Errorf("ensure collection: %w", err)
	}

	var points []map[string]any
	for _, doc := range docs {
		payload := map[string]any{}
		if doc.Meta != nil {
			for k, v := range doc.Meta {
				payload[k] = v
			}
		}
		payload["content"] = doc.Content
		points = append(points, map[string]any{
			"id":      doc.ID,
			"vector":  doc.Embedding,
			"payload": payload,
		})
	}

	body := map[string]any{"points": points}
	var resp map[string]any
	if err := s.doRequest(ctx, "PUT", fmt.Sprintf("/collections/%s/points", s.collection), body, &resp); err != nil {
		return 0, fmt.Errorf("upsert points: %w", err)
	}
	return len(docs), nil
}

// DeleteDocuments deletes documents matching the filters.
func (s *QdrantDocumentStore) DeleteDocuments(ctx context.Context, filters map[string]any) error {
	must := []map[string]any{}
	for k, v := range filters {
		must = append(must, map[string]any{
			"key": k,
			"match": map[string]any{"value": v},
		})
	}
	body := map[string]any{
		"filter": map[string]any{"must": must},
	}
	var resp map[string]any
	return s.doRequest(ctx, "POST", fmt.Sprintf("/collections/%s/points/delete", s.collection), body, &resp)
}

// QueryByEmbedding searches for documents by vector similarity.
func (s *QdrantDocumentStore) QueryByEmbedding(ctx context.Context, embedding []float32, filters map[string]any, topK int) ([]core.Document, error) {
	body := map[string]any{
		"query":        embedding,
		"limit":        topK,
		"with_payload": true,
	}
	if len(filters) > 0 {
		must := []map[string]any{}
		for k, v := range filters {
			must = append(must, map[string]any{
				"key": k,
				"match": map[string]any{"value": v},
			})
		}
		body["filter"] = map[string]any{"must": must}
	}

	var resp struct {
		Result []struct {
			ID      string         `json:"id"`
			Score   float32        `json:"score"`
			Payload map[string]any `json:"payload"`
		} `json:"result"`
	}
	if err := s.doRequest(ctx, "POST", fmt.Sprintf("/collections/%s/points/query", s.collection), body, &resp); err != nil {
		return nil, fmt.Errorf("query points: %w", err)
	}

	var docs []core.Document
	for _, r := range resp.Result {
		content, _ := r.Payload["content"].(string)
		delete(r.Payload, "content")
		docs = append(docs, core.Document{
			ID:      r.ID,
			Content: content,
			Meta:    r.Payload,
			Score:   r.Score,
		})
	}
	return docs, nil
}

func (s *QdrantDocumentStore) ensureCollection(ctx context.Context) error {
	// Check if collection exists
	req, err := http.NewRequestWithContext(ctx, "GET", s.host+"/collections/"+s.collection, nil)
	if err != nil {
		return err
	}
	if s.apiKey != "" {
		req.Header.Set("api-key", s.apiKey)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	// Create collection
	body := map[string]any{
		"vectors": map[string]any{
			"size":     s.embeddingDim,
			"distance": "Cosine",
		},
	}
	var createResp map[string]any
	return s.doRequest(ctx, "PUT", "/collections/"+s.collection, body, &createResp)
}

func (s *QdrantDocumentStore) doRequest(ctx context.Context, method, path string, body any, result any) error {
	url := s.host + path
	var bodyReader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("api-key", s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		return fmt.Errorf("qdrant %s %s returned %d: %v", method, path, resp.StatusCode, errBody)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// QdrantRetriever implements core.Retriever.
type QdrantRetriever struct {
	store core.DocumentStore
	topK  int
}

// Run performs vector similarity search.
func (r *QdrantRetriever) Run(ctx context.Context, queryEmbedding []float32, filters map[string]any) (*core.RetrievalResult, error) {
	docs, err := r.store.QueryByEmbedding(ctx, queryEmbedding, filters, r.topK)
	if err != nil {
		return nil, err
	}
	return &core.RetrievalResult{Documents: docs}, nil
}

func init() {
	provider.RegisterDocStore("qdrant", func(cfg map[string]any) (core.DocStoreProvider, error) {
		location, _ := cfg["location"].(string)
		if location == "" {
			location = "http://qdrant:6333"
		}
		apiKey, _ := cfg["api_key"].(string)
		timeout := 120
		if t, ok := cfg["timeout"].(int); ok {
			timeout = t
		}
		if t, ok := cfg["timeout"].(float64); ok {
			timeout = int(t)
		}
		dim := 3072
		if d, ok := cfg["embedding_model_dim"].(int); ok {
			dim = d
		}
		if d, ok := cfg["embedding_model_dim"].(float64); ok {
			dim = int(d)
		}
		return NewQdrantProvider(location, apiKey, dim, timeout), nil
	})
}
