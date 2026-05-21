package service

import (
	"testing"

	"github.com/odysseythink/go-wren-ai-service/internal/config"
)

func TestGenerateFromEnv(t *testing.T) {
	t.Setenv("LLM_OPENAI_API_KEY", "test")
	cfg := config.Load()
	comps, err := GenerateComponents(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	required := []string{
		"indexing", "retrieval", "historical_question",
		"sql_generation", "sql_correction", "followup_sql_generation",
		"sql_summary", "sql_answer", "sql_breakdown",
		"sql_expansion", "sql_explanation", "sql_regeneration",
		"semantics_description", "relationship_recommendation",
	}
	for _, name := range required {
		if _, ok := comps[name]; !ok {
			t.Fatalf("missing pipeline component: %s", name)
		}
	}
}

func TestGenerateComponents_FromYAML(t *testing.T) {
	yaml := `
type: llm
provider: openai_llm
models:
  - model: gpt-4o-mini
    kwargs:
      temperature: 0
api_key: sk-test

---
type: embedder
provider: openai_embedder
models:
  - model: text-embedding-3-large
    dimension: 3072
api_key: sk-test

---
type: document_store
provider: qdrant
location: http://localhost:6333
api_key: ""
embedding_model_dim: 3072

---
type: engine
provider: wren_ui
endpoint: http://localhost:3000

---
type: pipeline
pipes:
  - name: sql_generation
    llm: openai_llm.gpt-4o-mini
    engine: wren_ui
  - name: indexing
    embedder: openai_embedder.text-embedding-3-large
    document_store: qdrant
`
	cfg := &config.Config{Raw: mustParseYAML(yaml)}
	comps, err := GenerateComponents(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := comps["sql_generation"]; !ok {
		t.Fatal("missing sql_generation component")
	}
	if _, ok := comps["indexing"]; !ok {
		t.Fatal("missing indexing component")
	}

	sqlGen := comps["sql_generation"]
	if sqlGen.LLMProvider == nil {
		t.Fatal("sql_generation should have LLMProvider")
	}
	if sqlGen.Engine == nil {
		t.Fatal("sql_generation should have Engine")
	}
	if sqlGen.EmbedderProvider != nil {
		t.Fatal("sql_generation should NOT have EmbedderProvider")
	}

	idx := comps["indexing"]
	if idx.EmbedderProvider == nil {
		t.Fatal("indexing should have EmbedderProvider")
	}
	if idx.DocStoreProvider == nil {
		t.Fatal("indexing should have DocStoreProvider")
	}
}

func mustParseYAML(s string) *config.RawConfig {
	raw, err := config.ParseYAML([]byte(s))
	if err != nil {
		panic(err)
	}
	return raw
}
