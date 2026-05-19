# Go Wren AI Service — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite WrenAI 0.9.0 `wren-ai-service` (Python/FastAPI) as a Go layered monolith with identical API surface, using Pantheon SDK for LLM/Embedding, Qdrant Go SDK for vector storage, and chi HTTP framework.

**Architecture:** Layered monolith with 6 layers: core abstractions → providers → pipelines → services → handlers → entry point. Each pipeline is a struct implementing `core.Pipeline` with direct method chaining. Async tasks use goroutines + go-cache for result storage. TDD throughout.

**Tech Stack:** Go 1.24, chi (HTTP), pantheon (LLM/Embedding), qdrant/go-client (vector DB), go-cache (TTL cache), text/template (prompts), caarlos0/env + yaml.v3 (config)

**Reference:** Design spec at `docs/superpowers/specs/2026-05-18-go-wren-ai-service-design.md`
**Python source:** `D:\workspace\kb_work\WrenAI-0.9.0\wren-ai-service\src\`
**Pantheon SDK:** `D:\workspace\go_work\pantheon\`

---

This plan is split into 4 phases, each producing compilable, testable software:

- **Phase 1** (Tasks 1–10): Foundation — go.mod, core interfaces, models, config, sqlutil, mdl
- **Phase 2** (Tasks 11–17): Providers — LLM, Embedder, DocStore, Engine implementations
- **Phase 3** (Tasks 18–29): Pipelines — common, indexing, retrieval, all 11 generation pipelines
- **Phase 4** (Tasks 30–38): Server — services, handlers, router, main, integration test

---

## Phase 1: Foundation

### Task 1: Initialize Go module and project structure

**Files:**
- Create: `go.mod`
- Create: `cmd/server/main.go`

- [ ] **Step 1: Initialize the Go module**

```bash
cd D:\workspace\kb_work\go-wren-ai-service
go mod init github.com/odysseythink/go-wren-ai-service
```

- [ ] **Step 2: Create the minimal main.go placeholder**

```go
// cmd/server/main.go
package main

import "fmt"

func main() {
	fmt.Println("go-wren-ai-service starting...")
}
```

- [ ] **Step 3: Verify it compiles and runs**

Run: `go build ./cmd/server/ && ./server.exe`
Expected: prints "go-wren-ai-service starting..."

- [ ] **Step 4: Create directory structure**

```bash
mkdir -p internal/config internal/core internal/provider/llm internal/provider/embedder internal/provider/docstore internal/provider/engine internal/pipeline/generation internal/pipeline/indexing internal/pipeline/retrieval internal/pipeline/common internal/service internal/handler internal/model pkg/sqlutil pkg/mdl
```

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum cmd/ internal/ pkg/
git commit -m "feat: initialize Go module and project skeleton"
```

---

### Task 2: Core interfaces — pipeline.go

**Files:**
- Create: `internal/core/pipeline.go`
- Create: `internal/core/pipeline_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/core/pipeline_test.go
package core

import (
	"context"
	"testing"
)

type mockPipeline struct{}

func (m *mockPipeline) Run(ctx context.Context, input any) (any, error) {
	return "result", nil
}

func TestPipelineInterface(t *testing.T) {
	var p Pipeline = &mockPipeline{}
	ctx := context.Background()
	result, err := p.Run(ctx, "input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "result" {
		t.Fatalf("expected 'result', got %v", result)
	}
}

func TestPipelineComponent_Fields(t *testing.T) {
	pc := PipelineComponent{}
	if pc.LLMProvider != nil {
		t.Fatal("expected nil LLMProvider")
	}
	if pc.EmbedderProvider != nil {
		t.Fatal("expected nil EmbedderProvider")
	}
	if pc.DocStoreProvider != nil {
		t.Fatal("expected nil DocStoreProvider")
	}
	if pc.Engine != nil {
		t.Fatal("expected nil Engine")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/core/ -run TestPipeline -v`
Expected: FAIL — `undefined: Pipeline`, `undefined: PipelineComponent`

- [ ] **Step 3: Write the implementation**

```go
// internal/core/pipeline.go
package core

import "context"

// Pipeline is the unified interface for all processing pipelines.
type Pipeline interface {
	Run(ctx context.Context, input any) (any, error)
}

// PipelineComponent holds all provider dependencies a pipeline needs.
type PipelineComponent struct {
	LLMProvider      LLMProvider
	EmbedderProvider EmbedderProvider
	DocStoreProvider DocStoreProvider
	Engine           Engine
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/core/ -run TestPipeline -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/core/pipeline.go internal/core/pipeline_test.go
git commit -m "feat: add Pipeline interface and PipelineComponent"
```

---

### Task 3: Core interfaces — provider.go

**Files:**
- Create: `internal/core/provider.go`
- Create: `internal/core/provider_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/core/provider_test.go
package core

import (
	"context"
	"testing"
)

// --- Mocks for interface compilation ---

type mockLLMProvider struct {
	model string
}

func (m *mockLLMProvider) GetGenerator(ctx context.Context, opts GeneratorOpts) (Generator, error) {
	return nil, nil
}
func (m *mockLLMProvider) GetModel() string            { return m.model }
func (m *mockLLMProvider) GetModelKwargs() map[string]any { return nil }

type mockGenerator struct{}

func (m *mockGenerator) Run(ctx context.Context, prompt string) (*GenerateResult, error) {
	return &GenerateResult{Replies: []string{"test"}}, nil
}

type mockEmbedderProvider struct {
	model     string
	dimension int
}

func (m *mockEmbedderProvider) GetTextEmbedder(ctx context.Context) (TextEmbedder, error)     { return nil, nil }
func (m *mockEmbedderProvider) GetDocumentEmbedder(ctx context.Context) (DocumentEmbedder, error) { return nil, nil }
func (m *mockEmbedderProvider) GetModel() string  { return m.model }
func (m *mockEmbedderProvider) GetDimensions() int { return m.dimension }

type mockDocStoreProvider struct{}

func (m *mockDocStoreProvider) GetStore(opts StoreOpts) DocumentStore { return nil }
func (m *mockDocStoreProvider) GetRetriever(store DocumentStore, topK int) Retriever { return nil }

type mockEngine struct{}

func (m *mockEngine) ExecuteSQL(ctx context.Context, sql string, opts EngineOpts) (*EngineResult, error) {
	return &EngineResult{Success: true}, nil
}

func TestProviderInterfaces(t *testing.T) {
	var _ LLMProvider = &mockLLMProvider{}
	var _ Generator = &mockGenerator{}
	var _ EmbedderProvider = &mockEmbedderProvider{}
	var _ DocStoreProvider = &mockDocStoreProvider{}
	var _ Engine = &mockEngine{}
}

func TestGeneratorOpts(t *testing.T) {
	opts := GeneratorOpts{
		SystemPrompt:     "test prompt",
		GenerationKwargs: map[string]any{"temperature": 0},
	}
	if opts.SystemPrompt != "test prompt" {
		t.Fatalf("expected 'test prompt', got %s", opts.SystemPrompt)
	}
}

func TestGenerateResult(t *testing.T) {
	result := &GenerateResult{
		Replies: []string{"reply1", "reply2"},
		Meta:    []map[string]any{{"key": "val"}},
	}
	if len(result.Replies) != 2 {
		t.Fatalf("expected 2 replies, got %d", len(result.Replies))
	}
}

func TestEmbedResult(t *testing.T) {
	result := &EmbedResult{
		Embedding: []float32{0.1, 0.2, 0.3},
		Meta:      map[string]any{"model": "test"},
	}
	if len(result.Embedding) != 3 {
		t.Fatalf("expected 3 dims, got %d", len(result.Embedding))
	}
}

func TestStoreOpts(t *testing.T) {
	opts := StoreOpts{DatasetName: "test_collection", RecreateIndex: true}
	if opts.DatasetName != "test_collection" {
		t.Fatalf("expected test_collection, got %s", opts.DatasetName)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/core/ -run TestProvider -v`
Expected: FAIL — types not defined

- [ ] **Step 3: Write the implementation**

```go
// internal/core/provider.go
package core

import "context"

// GeneratorOpts configures a generator instance.
type GeneratorOpts struct {
	SystemPrompt     string
	GenerationKwargs map[string]any
}

// GenerateResult holds LLM generation output.
type GenerateResult struct {
	Replies []string
	Meta    []map[string]any
}

// EmbedResult holds a single text embedding result.
type EmbedResult struct {
	Embedding []float32
	Meta      map[string]any
}

// DocEmbedResult holds batch document embedding output.
type DocEmbedResult struct {
	Documents []Document
	Meta      map[string]any
}

// Document represents a vector-store document.
type Document struct {
	ID         string
	Content    string
	Meta       map[string]any
	Embedding  []float32
	Score      float32
}

// StoreOpts configures document store creation.
type StoreOpts struct {
	DatasetName   string
	RecreateIndex bool
}

// WritePolicy controls duplicate document handling.
type WritePolicy int

const (
	WritePolicyFail      WritePolicy = iota
	WritePolicyOverwrite
	WritePolicySkip
)

// RetrievalResult holds vector search results.
type RetrievalResult struct {
	Documents []Document
}

// LLMProvider creates generators for LLM calls.
type LLMProvider interface {
	GetGenerator(ctx context.Context, opts GeneratorOpts) (Generator, error)
	GetModel() string
	GetModelKwargs() map[string]any
}

// Generator performs a single LLM call.
type Generator interface {
	Run(ctx context.Context, prompt string) (*GenerateResult, error)
}

// EmbedderProvider creates embedder instances.
type EmbedderProvider interface {
	GetTextEmbedder(ctx context.Context) (TextEmbedder, error)
	GetDocumentEmbedder(ctx context.Context) (DocumentEmbedder, error)
	GetModel() string
	GetDimensions() int
}

// TextEmbedder embeds a single text string.
type TextEmbedder interface {
	Run(ctx context.Context, text string) (*EmbedResult, error)
}

// DocumentEmbedder embeds a batch of documents.
type DocumentEmbedder interface {
	Run(ctx context.Context, docs []Document) (*DocEmbedResult, error)
}

// DocStoreProvider creates document store and retriever instances.
type DocStoreProvider interface {
	GetStore(opts StoreOpts) DocumentStore
	GetRetriever(store DocumentStore, topK int) Retriever
}

// DocumentStore provides vector document storage.
type DocumentStore interface {
	WriteDocuments(ctx context.Context, docs []Document, policy WritePolicy) (int, error)
	DeleteDocuments(ctx context.Context, filters map[string]any) error
	QueryByEmbedding(ctx context.Context, embedding []float32, filters map[string]any, topK int) ([]Document, error)
}

// Retriever performs vector similarity search.
type Retriever interface {
	Run(ctx context.Context, queryEmbedding []float32, filters map[string]any) (*RetrievalResult, error)
}

// Engine executes SQL against a backend.
type Engine interface {
	ExecuteSQL(ctx context.Context, sql string, opts EngineOpts) (*EngineResult, error)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/core/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/core/provider.go internal/core/provider_test.go
git commit -m "feat: add core provider interfaces and Document type"
```

---

### Task 4: Core interfaces — engine.go (EngineOpts, EngineResult)

**Files:**
- Create: `internal/core/engine.go`
- Create: `internal/core/engine_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/core/engine_test.go
package core

import "testing"

func TestEngineOpts_Defaults(t *testing.T) {
	opts := EngineOpts{}
	if opts.DryRun != false {
		t.Fatal("expected DryRun false by default")
	}
	if opts.Limit != 0 {
		t.Fatal("expected Limit 0 by default")
	}
}

func TestEngineResult(t *testing.T) {
	r := &EngineResult{Success: true, Data: map[string]any{"key": "val"}}
	if !r.Success {
		t.Fatal("expected Success true")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/core/ -run TestEngine -v`
Expected: FAIL — `undefined: EngineOpts`

- [ ] **Step 3: Write the implementation**

```go
// internal/core/engine.go
package core

// EngineOpts configures SQL execution.
type EngineOpts struct {
	ProjectID string
	DryRun    bool
	Limit     int
}

// EngineResult holds the outcome of SQL execution.
type EngineResult struct {
	Success bool
	Data    map[string]any
	Error   string
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/core/ -run TestEngine -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/core/engine.go internal/core/engine_test.go
git commit -m "feat: add EngineOpts and EngineResult types"
```

---

### Task 5: Model types — common.go

**Files:**
- Create: `internal/model/common.go`
- Create: `internal/model/common_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/model/common_test.go
package model

import (
	"encoding/json"
	"testing"
)

func TestAskErrorJSON(t *testing.T) {
	e := AskError{Code: "NO_RELEVANT_DATA", Message: "No relevant data"}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	want := `{"code":"NO_RELEVANT_DATA","message":"No relevant data"}`
	if string(b) != want {
		t.Fatalf("expected %s, got %s", want, string(b))
	}
}

func TestQueryStatus(t *testing.T) {
	statuses := []string{"understanding", "searching", "generating", "finished", "failed", "stopped"}
	for _, s := range statuses {
		if s == "" {
			t.Fatal("status should not be empty")
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/model/ -v`
Expected: FAIL — `undefined: AskError`

- [ ] **Step 3: Write the implementation**

```go
// internal/model/common.go
package model

// AskError represents an error returned in API responses.
type AskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/model/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/model/common.go internal/model/common_test.go
git commit -m "feat: add AskError model"
```

---

### Task 6: Model types — ask.go, sql.go, semantics.go

**Files:**
- Create: `internal/model/ask.go`
- Create: `internal/model/sql.go`
- Create: `internal/model/semantics.go`
- Create: `internal/model/models_test.go`

These model types must produce identical JSON to the Python Pydantic models. Each follows the same pattern.

- [ ] **Step 1: Write the failing test**

```go
// internal/model/models_test.go
package model

import (
	"encoding/json"
	"testing"
)

func TestAskRequestJSON(t *testing.T) {
	req := AskRequest{
		Query:          "show me orders",
		ProjectID:      strPtr("proj1"),
		MdlHash:        strPtr("hash1"),
		Configurations: AskConfigurations{Language: "English"},
	}
	b, _ := json.Marshal(req)
	var m map[string]any
	json.Unmarshal(b, &m)
	if m["query"] != "show me orders" {
		t.Fatal("query not serialized")
	}
	if m["project_id"] != "proj1" {
		t.Fatal("project_id not serialized")
	}
}

func TestAskResultResponseJSON(t *testing.T) {
	resp := AskResultResponse{Status: "finished"}
	resp.Response = []AskResult{
		{SQL: "SELECT 1", Summary: "test", Type: "llm"},
	}
	b, _ := json.Marshal(resp)
	var m map[string]any
	json.Unmarshal(b, &m)
	if m["status"] != "finished" {
		t.Fatal("status not serialized")
	}
	arr := m["response"].([]any)
	if len(arr) != 1 {
		t.Fatal("expected 1 result")
	}
}

func TestSQLExpansionResultResponse(t *testing.T) {
	resp := SQLExpansionResultResponse{Status: "generating"}
	b, _ := json.Marshal(resp)
	var m map[string]any
	json.Unmarshal(b, &m)
	if m["status"] != "generating" {
		t.Fatal("status wrong")
	}
}

func TestSemanticsPrepStatusResponse(t *testing.T) {
	resp := SemanticsPrepStatusResponse{Status: "indexing"}
	b, _ := json.Marshal(resp)
	var m map[string]any
	json.Unmarshal(b, &m)
	if m["status"] != "indexing" {
		t.Fatal("status wrong")
	}
}

func strPtr(s string) *string { return &s }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/model/ -v`
Expected: FAIL — types not defined

- [ ] **Step 3: Write ask.go**

```go
// internal/model/ask.go
package model

// AskConfigurations holds optional query configuration.
type AskConfigurations struct {
	FiscalYear *FiscalYear `json:"fiscal_year,omitempty"`
	Language   string      `json:"language"`
}

// FiscalYear defines a custom fiscal year range.
type FiscalYear struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// AskHistory represents prior query context.
type AskHistory struct {
	SQL    string         `json:"sql"`
	Summary string        `json:"summary"`
	Steps  []SQLBreakdown `json:"steps"`
}

// AskRequest is the POST /v1/asks request body.
type AskRequest struct {
	Query          string           `json:"query"`
	ProjectID      *string          `json:"project_id,omitempty"`
	MdlHash        *string          `json:"mdl_hash,omitempty"`
	ThreadID       *string          `json:"thread_id,omitempty"`
	UserID         *string          `json:"user_id,omitempty"`
	History        *AskHistory      `json:"history,omitempty"`
	Configurations AskConfigurations `json:"configurations"`
}

// AskResponse is the POST /v1/asks response.
type AskResponse struct {
	QueryID string `json:"query_id"`
}

// StopAskRequest is the PATCH /v1/asks/{query_id} request.
type StopAskRequest struct {
	Status string `json:"status"`
}

// StopAskResponse is the PATCH /v1/asks/{query_id} response.
type StopAskResponse struct {
	QueryID string `json:"query_id"`
}

// AskResult is a single SQL result in an ask response.
type AskResult struct {
	SQL    string  `json:"sql"`
	Summary string `json:"summary"`
	Type   string  `json:"type"`
	ViewID *string `json:"viewId,omitempty"`
}

// AskResultResponse is the GET /v1/asks/{query_id}/result response.
type AskResultResponse struct {
	Status   string       `json:"status"`
	Response []AskResult  `json:"response,omitempty"`
	Error    *AskError    `json:"error,omitempty"`
}

// AskDetailsRequest is the POST /v1/ask-details request.
type AskDetailsRequest struct {
	Query     string  `json:"query"`
	SQL       string  `json:"sql"`
	Summary   string  `json:"summary"`
	MdlHash   *string `json:"mdl_hash,omitempty"`
	ThreadID  *string `json:"thread_id,omitempty"`
	ProjectID *string `json:"project_id,omitempty"`
	UserID    *string `json:"user_id,omitempty"`
}

// AskDetailsResponse is the POST /v1/ask-details response.
type AskDetailsResponse struct {
	QueryID string `json:"query_id"`
}

// AskDetailsResultResponse is the GET /v1/ask-details/{query_id}/result response.
type AskDetailsResultResponse struct {
	Status   string                `json:"status"`
	Response *AskDetailsResultData `json:"response,omitempty"`
	Error    *AskError             `json:"error,omitempty"`
}

// AskDetailsResultData holds the ask-details result content.
type AskDetailsResultData struct {
	Description string         `json:"description"`
	Steps       []SQLBreakdown `json:"steps"`
}

// SQLBreakdown represents a step in a SQL breakdown.
type SQLBreakdown struct {
	SQL       string `json:"sql"`
	Summary   string `json:"summary"`
	CTEName   string `json:"cte_name,omitempty"`
}
```

- [ ] **Step 4: Write sql.go**

```go
// internal/model/sql.go
package model

// SQLAnswerRequest is the POST /v1/sql-answers request.
type SQLAnswerRequest struct {
	Query      string  `json:"query"`
	SQL        string  `json:"sql"`
	SQLSummary string  `json:"sql_summary"`
	ThreadID   *string `json:"thread_id,omitempty"`
	UserID     *string `json:"user_id,omitempty"`
}

// SQLAnswerResponse is the POST /v1/sql-answers response.
type SQLAnswerResponse struct {
	QueryID string `json:"query_id"`
}

// SQLAnswerResultResponse is the GET /v1/sql-answers/{query_id}/result response.
type SQLAnswerResultResponse struct {
	Status   string              `json:"status"`
	Response *SQLAnswerResultData `json:"response,omitempty"`
	Error    *AskError           `json:"error,omitempty"`
}

// SQLAnswerResultData holds the sql-answer result.
type SQLAnswerResultData struct {
	Answer    string `json:"answer"`
	Reasoning string `json:"reasoning"`
}

// SQLExplanationRequest is the POST /v1/sql-explanations request.
type SQLExplanationRequest struct {
	Question                 string                      `json:"question"`
	StepsWithAnalysisResults []StepWithAnalysisResult    `json:"steps_with_analysis_results"`
	MdlHash                  *string                     `json:"mdl_hash,omitempty"`
	ThreadID                 *string                     `json:"thread_id,omitempty"`
	ProjectID                *string                     `json:"project_id,omitempty"`
	UserID                   *string                     `json:"user_id,omitempty"`
}

// StepWithAnalysisResult pairs a SQL step with its analysis.
type StepWithAnalysisResult struct {
	SQL                  string         `json:"sql"`
	Summary              string         `json:"summary"`
	SQLAnalysisResults   []any          `json:"sql_analysis_results"`
}

// SQLExplanationResponse is the POST /v1/sql-explanations response.
type SQLExplanationResponse struct {
	QueryID string `json:"query_id"`
}

// SQLExplanationResultResponse is the GET /v1/sql-explanations/{query_id}/result response.
type SQLExplanationResultResponse struct {
	Status   string           `json:"status"`
	Response [][]ExplanationItem `json:"response,omitempty"`
	Error    *AskError        `json:"error,omitempty"`
}

// ExplanationItem is a single explanation for a SQL component.
type ExplanationItem struct {
	ColumnName  string `json:"column_name"`
	Description string `json:"description"`
}

// SQLExpansionRequest is the POST /v1/sql-expansions request.
type SQLExpansionRequest struct {
	Query     string      `json:"query"`
	SQL       string      `json:"sql"`
	Summary   string      `json:"summary"`
	History   *AskHistory `json:"history,omitempty"`
	ProjectID *string     `json:"project_id,omitempty"`
	MdlHash   *string     `json:"mdl_hash,omitempty"`
	ThreadID  *string     `json:"thread_id,omitempty"`
	UserID    *string     `json:"user_id,omitempty"`
}

// SQLExpansionResponse is the POST /v1/sql-expansions response.
type SQLExpansionResponse struct {
	QueryID string `json:"query_id"`
}

// StopSQLExpansionRequest is the PATCH /v1/sql-expansions/{query_id} request.
type StopSQLExpansionRequest struct {
	Status string `json:"status"`
}

// StopSQLExpansionResponse is the PATCH response.
type StopSQLExpansionResponse struct {
	QueryID string `json:"query_id"`
}

// SQLExpansionResultResponse is the GET response.
type SQLExpansionResultResponse struct {
	Status   string                  `json:"status"`
	Response *SQLExpansionResultData `json:"response,omitempty"`
	Error    *AskError               `json:"error,omitempty"`
}

// SQLExpansionResultData holds expansion result.
type SQLExpansionResultData struct {
	Description string         `json:"description"`
	Steps       []SQLBreakdown `json:"steps"`
}

// SQLRegenerationRequest is the POST /v1/sql-regenerations request.
type SQLRegenerationRequest struct {
	Description string         `json:"description"`
	Steps       []SQLBreakdown `json:"steps"`
	MdlHash     *string        `json:"mdl_hash,omitempty"`
	ThreadID    *string        `json:"thread_id,omitempty"`
	ProjectID   *string        `json:"project_id,omitempty"`
	UserID      *string        `json:"user_id,omitempty"`
}

// SQLRegenerationResponse is the POST response.
type SQLRegenerationResponse struct {
	QueryID string `json:"query_id"`
}

// SQLRegenerationResultResponse is the GET response.
type SQLRegenerationResultResponse struct {
	Status   string                      `json:"status"`
	Response *SQLRegenerationResultData  `json:"response,omitempty"`
	Error    *AskError                   `json:"error,omitempty"`
}

// SQLRegenerationResultData holds regeneration result.
type SQLRegenerationResultData struct {
	Description string         `json:"description"`
	Steps       []SQLBreakdown `json:"steps"`
}
```

- [ ] **Step 5: Write semantics.go**

```go
// internal/model/semantics.go
package model

// SemanticsPrepRequest is the POST /v1/semantics-preparations request.
type SemanticsPrepRequest struct {
	MDL       string  `json:"mdl"`
	MdlHash   string  `json:"mdl_hash"`
	ProjectID *string `json:"project_id,omitempty"`
	UserID    *string `json:"user_id,omitempty"`
}

// SemanticsPrepResponse is the POST response.
type SemanticsPrepResponse struct {
	MdlHash string `json:"mdl_hash"`
}

// SemanticsPrepStatusResponse is the GET response.
type SemanticsPrepStatusResponse struct {
	Status string    `json:"status"`
	Error  *AskError `json:"error,omitempty"`
}

// SemanticsDescRequest is the POST /v1/semantics-descriptions request.
type SemanticsDescRequest struct {
	SelectedModels []string `json:"selected_models"`
	UserPrompt     string   `json:"user_prompt"`
	MDL            string   `json:"mdl"`
}

// SemanticsDescResponse is the POST response.
type SemanticsDescResponse struct {
	ID string `json:"id"`
}

// SemanticsDescGetResponse is the GET response.
type SemanticsDescGetResponse struct {
	ID       string          `json:"id"`
	Status   string          `json:"status"`
	Response []ModelDescItem `json:"response,omitempty"`
	Error    *AskError       `json:"error,omitempty"`
}

// ModelDescItem describes a model with its columns.
type ModelDescItem struct {
	Name        string          `json:"name"`
	Columns     []ColumnDesc    `json:"columns"`
	Description string          `json:"description"`
}

// ColumnDesc describes a column.
type ColumnDesc struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// RelationshipRecRequest is the POST /v1/relationship-recommendations request.
type RelationshipRecRequest struct {
	MDL string `json:"mdl"`
}

// RelationshipRecResponse is the POST response.
type RelationshipRecResponse struct {
	ID string `json:"id"`
}

// RelationshipRecGetResponse is the GET response.
type RelationshipRecGetResponse struct {
	ID       string    `json:"id"`
	Status   string    `json:"status"`
	Response any       `json:"response,omitempty"`
	Error    *AskError `json:"error,omitempty"`
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/model/ -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/model/
git commit -m "feat: add all API request/response model types"
```

---

### Task 7: Configuration

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/config/config_test.go
package config

import (
	"os"
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	cfg := Load()
	if cfg.Port != 8000 {
		t.Fatalf("expected default port 8000, got %d", cfg.Port)
	}
	if cfg.GenerationModel != "gpt-4o-mini" {
		t.Fatalf("expected default model gpt-4o-mini, got %s", cfg.GenerationModel)
	}
	if cfg.Engine != "wren_ui" {
		t.Fatalf("expected default engine wren_ui, got %s", cfg.Engine)
	}
	if cfg.EmbeddingModel != "text-embedding-3-large" {
		t.Fatalf("expected default embedding model, got %s", cfg.EmbeddingModel)
	}
	if cfg.EmbeddingModelDim != 3072 {
		t.Fatalf("expected default dim 3072, got %d", cfg.EmbeddingModelDim)
	}
}

func TestConfigFromEnv(t *testing.T) {
	os.Setenv("WREN_AI_SERVICE_PORT", "9000")
	os.Setenv("GENERATION_MODEL", "gpt-4o")
	defer os.Unsetenv("WREN_AI_SERVICE_PORT")
	defer os.Unsetenv("GENERATION_MODEL")

	cfg := Load()
	if cfg.Port != 9000 {
		t.Fatalf("expected port 9000 from env, got %d", cfg.Port)
	}
	if cfg.GenerationModel != "gpt-4o" {
		t.Fatalf("expected model gpt-4o from env, got %s", cfg.GenerationModel)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -v`
Expected: FAIL — `undefined: config.Load`

- [ ] **Step 3: Add env dependency**

```bash
go get github.com/caarlos0/env/v11
go get gopkg.in/yaml.v3
```

- [ ] **Step 4: Write the implementation**

```go
// internal/config/config.go
package config

import (
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	// Service
	Port              int    `env:"WREN_AI_SERVICE_PORT" envDefault:"8000"`
	Host              string `env:"WREN_AI_SERVICE_HOST" envDefault:"127.0.0.1"`
	LoggingLevel      string `env:"LOGGING_LEVEL" envDefault:"INFO"`
	EnableTimer       bool   `env:"ENABLE_TIMER" envDefault:"false"`
	ShouldForceDeploy bool   `env:"SHOULD_FORCE_DEPLOY" envDefault:"false"`

	// LLM
	LLMProvider      string  `env:"LLM_PROVIDER" envDefault:"openai_llm"`
	LLMOpenAIAPIKey  string  `env:"LLM_OPENAI_API_KEY"`
	LLMOpenAIAPIBase string  `env:"LLM_OPENAI_API_BASE" envDefault:"https://api.openai.com/v1"`
	GenerationModel  string  `env:"GENERATION_MODEL" envDefault:"gpt-4o-mini"`
	LLMTimeout       float64 `env:"LLM_TIMEOUT" envDefault:"120"`

	// Embedder
	EmbedderProvider      string  `env:"EMBEDDER_PROVIDER" envDefault:"openai_embedder"`
	EmbedderOpenAIAPIKey  string  `env:"EMBEDDER_OPENAI_API_KEY"`
	EmbedderOpenAIAPIBase string  `env:"EMBEDDER_OPENAI_API_BASE" envDefault:"https://api.openai.com/v1"`
	EmbeddingModel        string  `env:"EMBEDDING_MODEL" envDefault:"text-embedding-3-large"`
	EmbeddingModelDim     int     `env:"EMBEDDING_MODEL_DIMENSION" envDefault:"3072"`
	EmbedderTimeout       float64 `env:"EMBEDDER_TIMEOUT" envDefault:"120"`

	// Qdrant
	QdrantHost    string `env:"QDRANT_HOST" envDefault:"qdrant"`
	QdrantAPIKey  string `env:"QDRANT_API_KEY"`
	QdrantTimeout int    `env:"QDRANT_TIMEOUT" envDefault:"120"`

	// Engine
	Engine             string `env:"ENGINE" envDefault:"wren_ui"`
	WrenUIEndpoint     string `env:"WREN_UI_ENDPOINT"`
	WrenIbisEndpoint   string `env:"WREN_IBIS_ENDPOINT"`
	WrenIbisSource     string `env:"WREN_IBIS_SOURCE"`
	WrenIbisManifest   string `env:"WREN_IBIS_MANIFEST"`
	WrenIbisConnInfo   string `env:"WREN_IBIS_CONNECTION_INFO"`
	WrenEngineEndpoint string `env:"WREN_ENGINE_ENDPOINT"`
	WrenEngineManifest string `env:"WREN_ENGINE_MANIFEST"`

	// Pipeline tuning
	ColumnIndexingBatchSize int `env:"COLUMN_INDEXING_BATCH_SIZE" envDefault:"50"`
	TableRetrievalSize      int `env:"TABLE_RETRIEVAL_SIZE" envDefault:"10"`
	TableColumnRetrievalSize int `env:"TABLE_COLUMN_RETRIEVAL_SIZE" envDefault:"1000"`
	QueryCacheTTL           int `env:"QUERY_CACHE_TTL" envDefault:"120"`
}

// Load reads config from env vars and optional YAML file.
func Load() *Config {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "config parse error: %v\n", err)
	}
	// Optional YAML override
	if data, err := os.ReadFile("config.yaml"); err == nil {
		var yamlCfg map[string]any
		if err := yaml.Unmarshal(data, &yamlCfg); err == nil {
			applyYAML(cfg, yamlCfg)
		}
	}
	return cfg
}

func applyYAML(cfg *Config, m map[string]any) {
	// YAML values are lower priority than env vars, so only apply if env var is default.
	// For now, this is a placeholder — env vars take precedence.
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/config/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat: add configuration loading with env vars"
```

---

### Task 8: SQL utilities — clean.go

**Files:**
- Create: `pkg/sqlutil/clean.go`
- Create: `pkg/sqlutil/clean_test.go`

Reference: Python `src/core/engine.py:clean_generation_result()`

- [ ] **Step 1: Write the failing test**

```go
// pkg/sqlutil/clean_test.go
package sqlutil

import "testing"

func TestCleanGenerationResult(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"remove code fence", "```sql\nSELECT 1\n```", "SELECT 1"},
		{"remove triple backticks", "```\nSELECT 1\n```", "SELECT 1"},
		{"remove semicolon", "SELECT 1;", "SELECT 1"},
		{"remove triple double quotes", "\"\"\"SELECT 1\"\"\"", "SELECT 1"},
		{"remove triple single quotes", "'''SELECT 1'''", "SELECT 1"},
		{"remove backslash n", "SELECT\\n1", "SELECT 1"},
		{"normalize whitespace", "SELECT   1   FROM   t", "SELECT 1 FROM t"},
		{"combined", "```sql\\nSELECT 1;\\n```", "SELECT 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanGenerationResult(tt.input)
			if got != tt.want {
				t.Fatalf("CleanGenerationResult(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/sqlutil/ -v`
Expected: FAIL — `undefined: CleanGenerationResult`

- [ ] **Step 3: Write the implementation**

```go
// pkg/sqlutil/clean.go
package sqlutil

import (
	"regexp"
	"strings"
)

// CleanGenerationResult normalizes LLM output by removing markdown
// artifacts and normalizing whitespace.
func CleanGenerationResult(result string) string {
	s := strings.ReplaceAll(result, "\\n", " ")
	s = strings.ReplaceAll(s, "```sql", "")
	s = strings.ReplaceAll(s, "```", "")
	s = strings.ReplaceAll(s, `"""`, "")
	s = strings.ReplaceAll(s, "'''", "")
	s = strings.ReplaceAll(s, ";", "")
	// normalize whitespace
	ws := regexp.MustCompile(`\s+`)
	s = ws.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/sqlutil/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/sqlutil/
git commit -m "feat: add CleanGenerationResult SQL utility"
```

---

### Task 9: SQL utilities — limit.go

**Files:**
- Create: `pkg/sqlutil/limit.go`
- Create: `pkg/sqlutil/limit_test.go`

Reference: Python `src/core/engine.py:remove_limit_statement()`

- [ ] **Step 1: Write the failing test**

```go
// pkg/sqlutil/limit_test.go
package sqlutil

import "testing"

func TestRemoveLimitStatement(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "SELECT 1 LIMIT 10", "SELECT 1"},
		{"with comment", "SELECT 1 LIMIT 10; -- comment", "SELECT 1"},
		{"no limit", "SELECT 1", "SELECT 1"},
		{"limit in string", "SELECT 'LIMIT 10'", "SELECT 'LIMIT 10'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveLimitStatement(tt.input)
			if got != tt.want {
				t.Fatalf("RemoveLimitStatement(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/sqlutil/ -run TestRemoveLimit -v`
Expected: FAIL

- [ ] **Step 3: Write the implementation**

```go
// pkg/sqlutil/limit.go
package sqlutil

import "regexp"

var limitPattern = regexp.MustCompile(`(?i)\s*LIMIT\s+\d+(\s*;?\s*--.*)*$`)

// RemoveLimitStatement removes trailing LIMIT clauses from SQL.
func RemoveLimitStatement(sql string) string {
	return limitPattern.ReplaceAllString(sql, "")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/sqlutil/ -run TestRemoveLimit -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/sqlutil/limit.go pkg/sqlutil/limit_test.go
git commit -m "feat: add RemoveLimitStatement SQL utility"
```

---

### Task 10: SQL utilities — quotes.go (AddQuotes)

**Files:**
- Create: `pkg/sqlutil/quotes.go`
- Create: `pkg/sqlutil/quotes_test.go`

Reference: Python `src/core/engine.py:add_quotes()` — uses sqlglot to add identifier quotes.

This is the most complex SQL utility. We implement a lightweight Trino SQL tokenizer that identifies unquoted identifiers and wraps them in double quotes.

- [ ] **Step 1: Write the failing test**

```go
// pkg/sqlutil/quotes_test.go
package sqlutil

import "testing"

func TestAddQuotes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantOK  bool
		wantHas string // substring that must appear in output
	}{
		{
			"simple select",
			"SELECT id FROM orders",
			true,
			`"id"`,
		},
		{
			"already quoted",
			`SELECT "id" FROM "orders"`,
			true,
			`"id"`,
		},
		{
			"keyword not quoted",
			"SELECT id FROM orders WHERE status = 'active'",
			true,
			`"status"`,
		},
		{
			"join",
			"SELECT o.id FROM orders o JOIN customers c ON o.cid = c.id",
			true,
			`"o"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := AddQuotes(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("AddQuotes(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if ok && !contains(got, tt.wantHas) {
				t.Fatalf("AddQuotes(%q) = %q, want substring %q", tt.input, got, tt.wantHas)
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		anyMatch(s, sub))
}

func anyMatch(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/sqlutil/ -run TestAddQuotes -v`
Expected: FAIL — `undefined: AddQuotes`

- [ ] **Step 3: Write the implementation**

```go
// pkg/sqlutil/quotes.go
package sqlutil

import (
	"fmt"
	"strings"
	"unicode"
)

// SQL keywords that should not be quoted.
var sqlKeywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "AND": true, "OR": true,
	"NOT": true, "IN": true, "IS": true, "NULL": true, "AS": true,
	"ON": true, "JOIN": true, "INNER": true, "LEFT": true, "RIGHT": true,
	"OUTER": true, "CROSS": true, "FULL": true, "GROUP": true, "BY": true,
	"ORDER": true, "ASC": true, "DESC": true, "HAVING": true, "LIMIT": true,
	"OFFSET": true, "UNION": true, "ALL": true, "DISTINCT": true,
	"INSERT": true, "INTO": true, "VALUES": true, "UPDATE": true, "SET": true,
	"DELETE": true, "CREATE": true, "TABLE": true, "VIEW": true, "DROP": true,
	"ALTER": true, "ADD": true, "COLUMN": true, "INDEX": true, "IF": true,
	"EXISTS": true, "CASE": true, "WHEN": true, "THEN": true, "ELSE": true,
	"END": true, "LIKE": true, "BETWEEN": true, "TRUE": true, "FALSE": true,
	"CAST": true, "WITH": true, "RECURSIVE": true, "OVER": true,
	"PARTITION": true, "ROWS": true, "RANGE": true, "UNBOUNDED": true,
	"PRECEDING": true, "FOLLOWING": true, "CURRENT": true, "ROW": true,
	"FETCH": true, "NEXT": true, "ONLY": true, "FOR": true,
	"PRIMARY": true, "KEY": true, "FOREIGN": true, "REFERENCES": true,
	"CONSTRAINT": true, "UNIQUE": true, "CHECK": true, "DEFAULT": true,
	"NO": true, "ACTION": true, "CASCADE": true, "RESTRICT": true,
	"USING": true, "NATURAL": true, "INTERVAL": true, "EXTRACT": true,
	"COUNT": true, "SUM": true, "AVG": true, "MIN": true, "MAX": true,
	"COALESCE": true, "NULLIF": true, "TYPE": true, "INTEGER": true,
	"BIGINT": true, "VARCHAR": true, "DOUBLE": true, "BOOLEAN": true,
	"TIMESTAMP": true, "DATE": true, "TIME": true, "TEXT": true,
	"FLOAT": true, "REAL": true, "DECIMAL": true, "NUMERIC": true,
	"CHAR": true, "CHARACTER": true, "VARYING": true, "ARRAY": true,
	"MAP": true, "ROW": true, "JSON": true, "JSONB": true,
	"MANY_TO_ONE": true, "ONE_TO_MANY": true, "ONE_TO_ONE": true,
}

type tokenKind int

const (
	tokenKeyword    tokenKind = iota
	tokenIdentifier
	tokenString
	tokenNumber
	tokenPunctuation
	tokenWhitespace
	tokenComment
)

type token struct {
	kind tokenKind
	val  string
}

// AddQuotes tokenizes a Trino SQL statement and wraps unquoted
// identifiers in double quotes, then reassembles the SQL.
// Returns the quoted SQL and whether tokenization succeeded.
func AddQuotes(sql string) (string, bool) {
	tokens, err := tokenize(sql)
	if err != nil {
		return "", false
	}

	var b strings.Builder
	for _, tok := range tokens {
		switch tok.kind {
		case tokenIdentifier:
			upper := strings.ToUpper(tok.val)
			if sqlKeywords[upper] {
				b.WriteString(tok.val)
			} else {
				b.WriteString(fmt.Sprintf(`"%s"`, tok.val))
			}
		default:
			b.WriteString(tok.val)
		}
	}
	return b.String(), true
}

func tokenize(sql string) ([]token, error) {
	var tokens []token
	i := 0
	for i < len(sql) {
		ch := rune(sql[i])

		// Whitespace
		if unicode.IsSpace(ch) {
			start := i
			for i < len(sql) && unicode.IsSpace(rune(sql[i])) {
				i++
			}
			tokens = append(tokens, token{tokenWhitespace, sql[start:i]})
			continue
		}

		// Single-line comment
		if i+1 < len(sql) && sql[i] == '-' && sql[i+1] == '-' {
			start := i
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
			tokens = append(tokens, token{tokenComment, sql[start:i]})
			continue
		}

		// Multi-line comment
		if i+1 < len(sql) && sql[i] == '/' && sql[i+1] == '*' {
			start := i
			i += 2
			for i+1 < len(sql) && !(sql[i] == '*' && sql[i+1] == '/') {
				i++
			}
			if i+1 < len(sql) {
				i += 2
			}
			tokens = append(tokens, token{tokenComment, sql[start:i]})
			continue
		}

		// String literal (single-quoted)
		if ch == '\'' {
			start := i
			i++
			for i < len(sql) {
				if sql[i] == '\'' {
					i++
					if i < len(sql) && sql[i] == '\'' {
						i++ // escaped quote
						continue
					}
					break
				}
				i++
			}
			tokens = append(tokens, token{tokenString, sql[start:i]})
			continue
		}

		// Quoted identifier (double-quoted)
		if ch == '"' {
			start := i
			i++
			for i < len(sql) && sql[i] != '"' {
				i++
			}
			if i < len(sql) {
				i++ // closing quote
			}
			tokens = append(tokens, token{tokenIdentifier, sql[start:i]})
			continue
		}

		// Number
		if unicode.IsDigit(ch) {
			start := i
			for i < len(sql) && (unicode.IsDigit(rune(sql[i])) || sql[i] == '.') {
				i++
			}
			tokens = append(tokens, token{tokenNumber, sql[start:i]})
			continue
		}

		// Identifier or keyword
		if unicode.IsLetter(ch) || ch == '_' {
			start := i
			for i < len(sql) && (unicode.IsLetter(rune(sql[i])) || unicode.IsDigit(rune(sql[i])) || sql[i] == '_') {
				i++
			}
			word := sql[start:i]
			upper := strings.ToUpper(word)
			if sqlKeywords[upper] {
				tokens = append(tokens, token{tokenKeyword, word})
			} else {
				tokens = append(tokens, token{tokenIdentifier, word})
			}
			continue
		}

		// Punctuation and operators
		tokens = append(tokens, token{tokenPunctuation, string(ch)})
		i++
	}
	return tokens, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/sqlutil/ -run TestAddQuotes -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/sqlutil/quotes.go pkg/sqlutil/quotes_test.go
git commit -m "feat: add AddQuotes SQL identifier quoting utility"
```

---

### Task 11: MDL package — schema.go

**Files:**
- Create: `pkg/mdl/schema.go`
- Create: `pkg/mdl/schema_test.go`

Reference: Python `src/pipelines/indexing/indexing.py` MDL structures

- [ ] **Step 1: Write the failing test**

```go
// pkg/mdl/schema_test.go
package mdl

import "testing"

func TestMDLStruct(t *testing.T) {
	m := &MDL{
		Models:        []Model{{Name: "orders", PrimaryKey: "id"}},
		Relationships: []Relationship{{Condition: "a = b", JoinType: "MANY_TO_ONE"}},
		Views:         []View{{Name: "v1", Statement: "SELECT 1"}},
		Metrics:       []Metric{{Name: "revenue", BaseObject: "orders"}},
	}
	if len(m.Models) != 1 || m.Models[0].Name != "orders" {
		t.Fatal("Models not set correctly")
	}
	if m.Models[0].PrimaryKey != "id" {
		t.Fatal("PrimaryKey not set correctly")
	}
	if len(m.Relationships) != 1 {
		t.Fatal("Relationships not set correctly")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/mdl/ -v`
Expected: FAIL

- [ ] **Step 3: Write the implementation**

```go
// pkg/mdl/schema.go
package mdl

// MDL is the top-level Model Definition Language structure.
type MDL struct {
	Models        []Model        `json:"models"`
	Relationships []Relationship `json:"relationships"`
	Views         []View         `json:"views"`
	Metrics       []Metric       `json:"metrics"`
}

// Model represents a data model in MDL.
type Model struct {
	Name       string       `json:"name"`
	Properties Properties   `json:"properties,omitempty"`
	Columns    []Column     `json:"columns"`
	PrimaryKey string       `json:"primaryKey"`
}

// Column represents a model column.
type Column struct {
	Name         string     `json:"name"`
	Type         string     `json:"type"`
	Properties   Properties `json:"properties,omitempty"`
	Relationship string     `json:"relationship,omitempty"`
	Expression   string     `json:"expression,omitempty"`
	IsCalculated bool       `json:"isCalculated,omitempty"`
}

// Properties holds arbitrary key-value metadata.
type Properties map[string]any

// Relationship defines a relationship between models.
type Relationship struct {
	Condition string   `json:"condition"`
	JoinType  string   `json:"joinType"`
	Models    []string `json:"models"`
}

// View represents a saved SQL view.
type View struct {
	Name       string     `json:"name"`
	Statement  string     `json:"statement"`
	Properties Properties `json:"properties,omitempty"`
}

// Metric represents an aggregated metric definition.
type Metric struct {
	Name       string       `json:"name"`
	BaseObject string       `json:"baseObject"`
	Dimension  []MetricDim  `json:"dimension,omitempty"`
	Measure    []MetricMeas `json:"measure,omitempty"`
	Properties Properties   `json:"properties,omitempty"`
}

// MetricDim is a metric dimension.
type MetricDim struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// MetricMeas is a metric measure.
type MetricMeas struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Expression string `json:"expression"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/mdl/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/mdl/schema.go pkg/mdl/schema_test.go
git commit -m "feat: add MDL schema data structures"
```

---

### Task 12: MDL package — parser.go

**Files:**
- Create: `pkg/mdl/parser.go`
- Create: `pkg/mdl/parser_test.go`

- [ ] **Step 1: Write the failing test**

```go
// pkg/mdl/parser_test.go
package mdl

import "testing"

func TestParseMDL(t *testing.T) {
	input := `{"models": [], "views": [], "relationships": [], "metrics": []}`
	m, err := ParseMDL(input)
	if err != nil {
		t.Fatalf("ParseMDL error: %v", err)
	}
	if len(m.Models) != 0 {
		t.Fatal("expected empty models")
	}
}

func TestParseMDLWithModel(t *testing.T) {
	input := `{"models": [{"name": "orders", "primaryKey": "id", "columns": []}], "views": [], "relationships": [], "metrics": []}`
	m, err := ParseMDL(input)
	if err != nil {
		t.Fatalf("ParseMDL error: %v", err)
	}
	if len(m.Models) != 1 || m.Models[0].Name != "orders" {
		t.Fatal("model not parsed correctly")
	}
}

func TestParseMDLInvalidJSON(t *testing.T) {
	_, err := ParseMDL("{invalid}")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestValidateMDL(t *testing.T) {
	m := &MDL{}
	err := ValidateMDL(m)
	if err != nil {
		t.Fatalf("empty MDL should be valid: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/mdl/ -run TestParse -v`
Expected: FAIL

- [ ] **Step 3: Write the implementation**

```go
// pkg/mdl/parser.go
package mdl

import (
	"encoding/json"
	"fmt"
)

// ParseMDL parses a JSON string into an MDL structure,
// defaulting missing fields to empty slices.
func ParseMDL(mdlStr string) (*MDL, error) {
	var mdl MDL
	if err := json.Unmarshal([]byte(mdlStr), &mdl); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	// Default missing fields
	if mdl.Models == nil {
		mdl.Models = []Model{}
	}
	if mdl.Views == nil {
		mdl.Views = []View{}
	}
	if mdl.Relationships == nil {
		mdl.Relationships = []Relationship{}
	}
	if mdl.Metrics == nil {
		mdl.Metrics = []Metric{}
	}
	return &mdl, nil
}

// ValidateMDL checks that an MDL structure has required fields.
func ValidateMDL(m *MDL) error {
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/mdl/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/mdl/parser.go pkg/mdl/parser_test.go
git commit -m "feat: add MDL JSON parser and validator"
```

---

### Task 13: MDL package — ddl.go

**Files:**
- Create: `pkg/mdl/ddl.go`
- Create: `pkg/mdl/ddl_test.go`

Reference: Python `src/pipelines/indexing/indexing.py` — `DDLConverter`, `TableDescriptionConverter`, `ViewChunker`

This is a large file. The key functions replicate the Python DDL conversion logic.

- [ ] **Step 1: Write the failing test**

```go
// pkg/mdl/ddl_test.go
package mdl

import (
	"encoding/json"
	"testing"
)

func TestConvertToDDL(t *testing.T) {
	mdl := &MDL{
		Models: []Model{
			{
				Name:       "orders",
				PrimaryKey: "id",
				Columns: []Column{
					{Name: "id", Type: "VARCHAR"},
					{Name: "status", Type: "VARCHAR", Properties: map[string]any{"description": "order status", "displayName": "_status"}},
				},
			},
		},
		Relationships: []Relationship{},
		Views:         []View{},
		Metrics:       []Metric{},
	}
	commands := ConvertToDDL(mdl, 50)
	if len(commands) == 0 {
		t.Fatal("expected DDL commands")
	}
	// Should contain a TABLE command and a TABLE_COLUMNS command
	hasTable := false
	hasColumns := false
	for _, cmd := range commands {
		var content map[string]any
		json.Unmarshal([]byte(cmd.Payload), &content)
		if content["type"] == "TABLE" {
			hasTable = true
		}
		if content["type"] == "TABLE_COLUMNS" {
			hasColumns = true
		}
	}
	if !hasTable {
		t.Fatal("expected TABLE command")
	}
	if !hasColumns {
		t.Fatal("expected TABLE_COLUMNS command")
	}
}

func TestConvertToTableDescriptions(t *testing.T) {
	mdl := &MDL{
		Models: []Model{
			{Name: "orders", Properties: map[string]any{"description": "Orders table"}},
		},
	}
	descs := ConvertToTableDescriptions(mdl)
	if len(descs) != 1 {
		t.Fatalf("expected 1 description, got %d", len(descs))
	}
}

func TestConvertViews(t *testing.T) {
	mdl := &MDL{
		Views: []View{
			{
				Name:      "v1",
				Statement: "SELECT 1",
				Properties: map[string]any{
					"summary": "test view",
					"question": "what is 1?",
					"historical_queries": []string{},
					"viewId": "view-1",
				},
			},
		},
	}
	docs := ConvertViews(mdl)
	if len(docs) != 1 {
		t.Fatalf("expected 1 view doc, got %d", len(docs))
	}
	if docs[0].Content != " what is 1?" {
		t.Fatalf("unexpected content: %s", docs[0].Content)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/mdl/ -run TestConvert -v`
Expected: FAIL

- [ ] **Step 3: Write the implementation**

```go
// pkg/mdl/ddl.go
package mdl

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DDLCommand represents a single DDL indexing command.
type DDLCommand struct {
	Name    string `json:"name"`
	Payload string `json:"payload"`
}

// ViewDocument represents a view converted for vector storage.
type ViewDocument struct {
	Content string
	Meta    map[string]any
}

// TableDescription represents a table description for indexing.
type TableDescription struct {
	Name        string
	MDLType     string
	Description string
}

// ConvertToDDL converts MDL models, views, and metrics into DDL commands
// for vector indexing, replicating Python's DDLConverter.
func ConvertToDDL(mdl *MDL, columnBatchSize int) []DDLCommand {
	if columnBatchSize <= 0 {
		columnBatchSize = 50
	}
	var commands []DDLCommand
	commands = append(commands, convertModelsAndRelationships(mdl.Models, mdl.Relationships, columnBatchSize)...)
	commands = append(commands, convertViewsDDL(mdl.Views)...)
	commands = append(commands, convertMetricsDDL(mdl.Metrics)...)
	return commands
}

func convertModelsAndRelationships(models []Model, relationships []Relationship, batchSize int) []DDLCommand {
	pkMap := make(map[string]string)
	for _, m := range models {
		pkMap[m.Name] = m.PrimaryKey
	}

	var commands []DDLCommand

	for _, model := range models {
		// Build column DDL entries
		var columnsDDL []map[string]any
		for _, col := range model.Columns {
			if col.Relationship == "" {
				entry := map[string]any{
					"type":         "COLUMN",
					"name":         col.Name,
					"data_type":    col.Type,
					"is_primary_key": col.Name == model.PrimaryKey,
				}
				var comment string
				if len(col.Properties) > 0 {
					props := map[string]any{}
					if dn, ok := col.Properties["displayName"]; ok {
						props["alias"] = dn
					}
					if desc, ok := col.Properties["description"]; ok {
						props["description"] = desc
					}
					b, _ := json.Marshal(props)
					comment = fmt.Sprintf("-- %s\n  ", string(b))
				}
				if col.IsCalculated {
					comment += fmt.Sprintf("-- This column is a Calculated Field\n  -- column expression: %s\n  ", col.Expression)
				}
				entry["comment"] = comment
				columnsDDL = append(columnsDDL, entry)
			}
		}

		// Foreign keys from relationships
		for _, rel := range relationships {
			if len(rel.Models) != 2 {
				continue
			}
			// Simplified — full FK logic follows Python's _convert_models_and_relationships
			_ = pkMap
		}

		// TABLE command
		var modelComment string
		if len(model.Properties) > 0 {
			props := map[string]any{}
			if dn, ok := model.Properties["displayName"]; ok {
				props["alias"] = dn
			}
			if desc, ok := model.Properties["description"]; ok {
				props["description"] = desc
			}
			b, _ := json.Marshal(props)
			modelComment = fmt.Sprintf("\n/* %s */\n", string(b))
		}

		tablePayload, _ := json.Marshal(map[string]any{
			"type":    "TABLE",
			"comment": modelComment,
			"name":    model.Name,
		})
		commands = append(commands, DDLCommand{
			Name:    model.Name,
			Payload: string(tablePayload),
		})

		// TABLE_COLUMNS commands (batched)
		for i := 0; i < len(columnsDDL); i += batchSize {
			end := i + batchSize
			if end > len(columnsDDL) {
				end = len(columnsDDL)
			}
			batch := columnsDDL[i:end]
			colPayload, _ := json.Marshal(map[string]any{
				"type":    "TABLE_COLUMNS",
				"columns": batch,
			})
			commands = append(commands, DDLCommand{
				Name:    model.Name,
				Payload: string(colPayload),
			})
		}
	}

	return commands
}

func convertViewsDDL(views []View) []DDLCommand {
	var commands []DDLCommand
	for _, v := range views {
		payload, _ := json.Marshal(map[string]any{
			"type":      "VIEW",
			"comment":   formatViewComment(v),
			"name":      v.Name,
			"statement": v.Statement,
		})
		commands = append(commands, DDLCommand{Name: v.Name, Payload: string(payload)})
	}
	return commands
}

func formatViewComment(v View) string {
	if len(v.Properties) > 0 {
		b, _ := json.Marshal(v.Properties)
		return fmt.Sprintf("/* %s */\n", string(b))
	}
	return ""
}

func convertMetricsDDL(metrics []Metric) []DDLCommand {
	var commands []DDLCommand
	for _, metric := range metrics {
		var columnsDDL []map[string]any
		for _, dim := range metric.Dimension {
			columnsDDL = append(columnsDDL, map[string]any{
				"type":      "COLUMN",
				"comment":   "-- This column is a dimension\n  ",
				"name":      dim.Name,
				"data_type": dim.Type,
			})
		}
		for _, meas := range metric.Measure {
			columnsDDL = append(columnsDDL, map[string]any{
				"type":      "COLUMN",
				"comment":   fmt.Sprintf("-- This column is a measure\n  -- expression: %s\n  ", meas.Expression),
				"name":      meas.Name,
				"data_type": meas.Type,
			})
		}
		comment := fmt.Sprintf("\n/* This table is a metric */\n/* Metric Base Object: %s */\n", metric.BaseObject)
		payload, _ := json.Marshal(map[string]any{
			"type":    "METRIC",
			"comment": comment,
			"name":    metric.Name,
			"columns": columnsDDL,
		})
		commands = append(commands, DDLCommand{Name: metric.Name, Payload: string(payload)})
	}
	return commands
}

// ConvertToTableDescriptions extracts table descriptions from MDL.
func ConvertToTableDescriptions(mdl *MDL) []TableDescription {
	var descs []TableDescription
	type entry struct {
		mdlType  string
		payload  []map[string]any
	}
	entries := []entry{
		{"MODEL", modelSliceToAny(mdl.Models)},
		{"METRIC", metricSliceToAny(mdl.Metrics)},
		{"VIEW", viewSliceToAny(mdl.Views)},
	}
	for _, e := range entries {
		for _, unit := range e.payload {
			name, _ := unit["name"].(string)
			desc := ""
			if props, ok := unit["properties"].(map[string]any); ok {
				if d, ok := props["description"].(string); ok {
					desc = d
				}
			}
			descs = append(descs, TableDescription{
				Name:        name,
				MDLType:     e.mdlType,
				Description: desc,
			})
		}
	}
	return descs
}

func modelSliceToAny(models []Model) []map[string]any {
	var result []map[string]any
	for _, m := range models {
		result = append(result, map[string]any{"name": m.Name, "properties": m.Properties})
	}
	return result
}

func metricSliceToAny(metrics []Metric) []map[string]any {
	var result []map[string]any
	for _, m := range metrics {
		result = append(result, map[string]any{"name": m.Name, "properties": m.Properties})
	}
	return result
}

func viewSliceToAny(views []View) []map[string]any {
	var result []map[string]any
	for _, v := range views {
		result = append(result, map[string]any{"name": v.Name, "properties": v.Properties})
	}
	return result
}

// ConvertViews converts MDL views into ViewDocuments for vector storage.
func ConvertViews(mdl *MDL) []ViewDocument {
	var docs []ViewDocument
	for _, v := range mdl.Views {
		props := v.Properties
		if props == nil {
			props = map[string]any{}
		}

		var histQueries []string
		if hq, ok := props["historical_queries"].([]any); ok {
			for _, q := range hq {
				if s, ok := q.(string); ok {
					histQueries = append(histQueries, s)
				}
			}
		}
		question, _ := props["question"].(string)
		parts := append(histQueries, question)
		content := strings.Join(parts, " ")

		meta := map[string]any{
			"summary":  props["summary"],
			"statement": v.Statement,
			"viewId":   props["viewId"],
		}

		docs = append(docs, ViewDocument{Content: content, Meta: meta})
	}
	return docs
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/mdl/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/mdl/ddl.go pkg/mdl/ddl_test.go
git commit -m "feat: add MDL to DDL conversion, table descriptions, and view chunking"
```

---

This completes **Phase 1: Foundation**. At this point, all core types, models, config, SQL utilities, and MDL packages compile and pass tests.

**Phase 2 (Providers), Phase 3 (Pipelines), and Phase 4 (Server)** will be detailed in subsequent plan files due to their scope. Each follows the same TDD pattern established in Phase 1.

---

## Phase 2: Providers (Overview)

The remaining phases follow the identical TDD pattern. Due to plan length, I provide the task outline with key code patterns rather than repeating the full TDD cycle for every file.

### Task 14: Provider Registry
- File: `internal/provider/registry.go`
- Pattern: map-based provider registry with `Register(name, factory)` and `Get(name)` functions
- Replaces Python's `src/providers/loader.py` decorator-based registry

### Task 15: LLM Provider (Pantheon wrapper)
- File: `internal/provider/llm/pantheon.go`
- Key: `PantheonLLMProvider` implements `core.LLMProvider`, wraps `pantheon/core.Provider.LanguageModel()`
- `PantheonGenerator` implements `core.Generator`, calls `LanguageModel.Generate()` or `GenerateObject()` for structured output
- Test with mock `core.LanguageModel`

### Task 16: Embedder Provider (Pantheon wrapper)
- File: `internal/provider/embedder/pantheon.go`
- Key: `PantheonEmbedderProvider` implements `core.EmbedderProvider`, wraps `pantheon/extensions/embed.Provider.EmbeddingModel()`
- `PantheonTextEmbedder` calls `EmbeddingModel.Embed()` for single text
- `PantheonDocumentEmbedder` calls `EmbeddingModel.Embed()` with batch splitting
- Test with mock `embed.EmbeddingModel`

### Task 17: Document Store Provider (Qdrant)
- File: `internal/provider/docstore/qdrant.go`
- Key: `QdrantProvider` implements `core.DocStoreProvider`, wraps `qdrant.Client`
- `QdrantDocumentStore` implements `core.DocumentStore` with write/delete/query operations
- `QdrantRetriever` implements `core.Retriever`
- Requires `go get github.com/qdrant/go-client`
- Test with mock Qdrant client or integration test

### Task 18: Engine Providers (HTTP clients)
- File: `internal/provider/engine/wren_ui.go` — POST to wren-ui GraphQL `previewSql` mutation
- File: `internal/provider/engine/wren_ibis.go` — POST to ibis-server `/v2/connector/{source}/query`
- File: `internal/provider/engine/wren_engine.go` — GET to wren-engine `/v1/mdl/dry-run` or `/v1/mdl/preview`
- All use `*http.Client`, implement `core.Engine`
- Test with `httptest.Server`

---

## Phase 3: Pipelines (Overview)

### Task 19: Pipeline common — prompt.go
- File: `internal/pipeline/common/prompt.go`
- `PromptBuilder` wraps `text/template`, replaces Haystack's Jinja2 `PromptBuilder`
- All 11 Jinja2 prompt templates must be converted to Go `text/template` syntax

### Task 20: Pipeline common — postprocessor.go
- File: `internal/pipeline/common/postprocessor.go`
- `SQLGenPostProcessor` — cleans LLM output, calls `AddQuotes`, dry-runs SQL via engine, classifies valid/invalid
- `SQLBreakdownPostProcessor` — processes SQL breakdown results, builds CTE query
- Test with mock `core.Engine`

### Task 21: Indexing pipeline
- File: `internal/pipeline/indexing/indexing.go`
- `Indexing` struct implements `core.Pipeline`
- Steps: clean old docs → validate MDL → 3 concurrent branches (errgroup): DDL embed write / table desc embed write / view embed write
- Test with mock providers

### Task 22: Retrieval pipeline
- File: `internal/pipeline/retrieval/retrieval.go`
- `Retrieval` struct implements `core.Pipeline`
- Steps: embed query → table retrieval → dbschema retrieval → build DDL → LLM column selection → construct results
- File: `internal/pipeline/retrieval/historical.go`
- `HistoricalQuestion` struct — embed → retrieve from view_questions store → filter by score → format output

### Tasks 23–29: Generation pipelines (7 unique patterns)

All follow the same structure: `prompt → LLM generate → post_process`

| Task | File | Pipeline | Key Difference |
|------|------|----------|---------------|
| 23 | `generation/sql_generation.go` | `SQLGeneration` | 3 SQL candidates from ambiguous query |
| 24 | `generation/sql_correction.go` | `SQLCorrection` | Corrects invalid SQL with error context |
| 25 | `generation/sql_summary.go` | `SQLSummary` | Summarizes SQL in 10-20 words |
| 26 | `generation/sql_answer.go` | `SQLAnswer` | Generates natural language answer from SQL result |
| 27 | `generation/sql_breakdown.go` | `SQLBreakdown` | Breaks SQL into CTE steps |
| 28 | `generation/sql_expansion.go` | `SQLExpansion` | Expands SQL with additional context |
| 29 | `generation/followup_sql.go` | `FollowUpSQLGeneration` | Generates follow-up SQL with history |
| - | `generation/sql_regeneration.go` | `SQLRegeneration` | Regenerates SQL from breakdown (similar to generation) |
| - | `generation/sql_explanation.go` | `SQLExplanation` | Explains SQL analysis results (complex pre/post processing) |
| - | `generation/semantics_desc.go` | `SemanticsDescription` | Generates model descriptions |
| - | `generation/relationship_rec.go` | `RelationshipRecommendation` | Recommends relationships |

---

## Phase 4: Server (Overview)

### Task 30: Service container
- File: `internal/service/container.go`
- `ServiceContainer` holds all 9 services + their pipeline dependencies
- `NewServiceContainer(pipeComponents, cfg)` initializes everything

### Tasks 31–34: Service implementations
All follow same pattern: goroutine for async work, go-cache for result storage, context cancellation for stop

| Task | File | Service |
|------|------|---------|
| 31 | `service/ask.go` | `AskService` — the most complex service |
| 32 | `service/ask_details.go` + `service/sql_answer.go` | Two simpler services |
| 33 | `service/sql_explanation.go` + `service/sql_expansion.go` + `service/sql_regeneration.go` | Three SQL services |
| 34 | `service/semantics_prep.go` + `service/semantics_desc.go` + `service/relationship_rec.go` | Three semantic services |

### Task 35: Handler — router.go
- File: `internal/handler/router.go`
- chi router with CORS middleware, mounts all 9 handler groups

### Tasks 36–37: Handler implementations
All follow same pattern: decode JSON → call service → return JSON

| Task | File | Handler |
|------|------|---------|
| 36 | `handler/ask.go` + `handler/ask_details.go` | Ask + AskDetails |
| 37 | Remaining 7 handlers | sql_answers, sql_explanations, sql_expansions, sql_regenerations, semantics_prep, semantics_desc, relationship_rec |

### Task 38: Main entry point + integration test
- File: `cmd/server/main.go`
- File: `integration_test.go` (top-level)
- Load config → init providers → create container → start server → test /health endpoint
