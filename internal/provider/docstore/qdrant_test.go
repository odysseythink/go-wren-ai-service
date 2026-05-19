package docstore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
)

func TestQdrantDocumentStore_WriteDocuments(t *testing.T) {
	var upsertCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/collections/test" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
			return
		}
		if r.Method == "PUT" && r.URL.Path == "/collections/test/points" {
			upsertCalled = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := &QdrantDocumentStore{
		host:       server.URL,
		collection: "test",
		httpClient: server.Client(),
	}

	docs := []core.Document{
		{ID: "1", Content: "hello", Meta: map[string]any{"project_id": "p1"}, Embedding: []float32{0.1, 0.2}},
	}
	written, err := store.WriteDocuments(context.Background(), docs, core.WritePolicyFail)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if written != 1 {
		t.Fatalf("expected 1 written, got %d", written)
	}
	if !upsertCalled {
		t.Fatal("expected upsert to be called")
	}
}

func TestQdrantDocumentStore_QueryByEmbedding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/collections/test/points/query" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"result": []map[string]any{
					{"id": "1", "score": 0.95, "payload": map[string]any{"content": "hello"}},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := &QdrantDocumentStore{
		host:       server.URL,
		collection: "test",
		httpClient: server.Client(),
	}

	docs, err := store.QueryByEmbedding(context.Background(), []float32{0.1, 0.2}, nil, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].ID != "1" {
		t.Fatalf("unexpected id: %s", docs[0].ID)
	}
	if docs[0].Score != 0.95 {
		t.Fatalf("unexpected score: %f", docs[0].Score)
	}
}

func TestQdrantProvider_GetStoreAndRetriever(t *testing.T) {
	provider := NewQdrantProvider("http://localhost:6333", "", 3072, 10)
	store := provider.GetStore(core.StoreOpts{DatasetName: "test"})
	if store == nil {
		t.Fatal("expected store")
	}
	retriever := provider.GetRetriever(store, 5)
	if retriever == nil {
		t.Fatal("expected retriever")
	}
}
