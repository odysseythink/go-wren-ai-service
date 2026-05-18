# Go Wren AI Service - Design Document

**Date**: 2026-05-18
**Topic**: Rewrite WrenAI 0.9.0 `wren-ai-service` (Python) in Go
**Status**: Approved

## 1. Background

WrenAI 0.9.0 (`ghcr.io/canner/wren-ai-service:0.9.0`) is a Python FastAPI application that provides AI-powered SQL generation, explanation, and semantic analysis services. It is part of the larger WrenAI platform but is deployed as an independent Docker container.

### Source Code Mapping

The Docker image `wren-ai-service:0.9.0` corresponds **only** to the `wren-ai-service/` directory in `D:\workspace\kb_work\WrenAI-0.9.0`. Other components (wren-engine, wren-ui, wren-launcher, wren-mdl) are separate images.

### Python Tech Stack

| Component | Technology |
|-----------|-----------|
| HTTP Framework | FastAPI + Uvicorn |
| LLM Pipeline | Haystack AI + Hamilton AsyncDriver |
| LLM/Embedding | OpenAI/Azure/Ollama via Haystack |
| Vector DB | Qdrant via haystack-ai + qdrant-haystack |
| SQL Processing | sqlglot |
| Observability | Langfuse |
| Caching | cachetools.TTLCache |
| Background Tasks | FastAPI BackgroundTasks |
| Config | pydantic-settings + YAML |

### Python Source Structure

```
wren-ai-service/src/
├── __main__.py           # FastAPI app entry point
├── globals.py            # ServiceContainer, ServiceMetadata
├── utils.py              # Logging, timer, Langfuse, trace_metadata
├── core/
│   ├── pipeline.py       # BasicPipeline, PipelineComponent
│   ├── provider.py       # LLMProvider, EmbedderProvider, DocStoreProvider
│   └── engine.py         # Engine, clean_generation_result, add_quotes, remove_limit_statement
├── providers/
│   ├── __init__.py       # generate_components, init_providers, config parsing
│   ├── loader.py         # Provider registry (decorator-based)
│   ├── llm/openai.py     # OpenAI LLM (AsyncGenerator)
│   ├── llm/azure_openai.py
│   ├── llm/ollama.py
│   ├── embedder/openai.py  # OpenAI Embedding (AsyncTextEmbedder, AsyncDocumentEmbedder)
│   ├── embedder/azure_openai.py
│   ├── embedder/ollama.py
│   ├── document_store/qdrant.py  # AsyncQdrantDocumentStore, AsyncQdrantEmbeddingRetriever
│   └── engine/wren.py    # WrenUI, WrenIbis, WrenEngine
├── pipelines/
│   ├── common.py         # SQLGenPostProcessor, SQLBreakdownGenPostProcessor, prompts
│   ├── generation/
│   │   ├── sql_generation.py
│   │   ├── sql_correction.py
│   │   ├── sql_explanation.py
│   │   ├── sql_expansion.py
│   │   ├── sql_regeneration.py
│   │   ├── sql_summary.py
│   │   ├── sql_answer.py
│   │   ├── sql_breakdown.py
│   │   ├── followup_sql_generation.py
│   │   ├── semantics_description.py
│   │   └── relationship_recommendation.py
│   ├── indexing/indexing.py  # Indexing pipeline + MDL validation/conversion
│   └── retrieval/
│       ├── retrieval.py      # Table/column retrieval pipeline
│       └── historical_question.py
└── web/v1/
    ├── __init__.py           # Router aggregation
    ├── routers/
    │   ├── ask.py
    │   ├── ask_details.py
    │   ├── sql_answers.py
    │   ├── sql_explanations.py
    │   ├── sql_expansions.py
    │   ├── sql_regenerations.py
    │   ├── semantics_preparations.py
    │   ├── semantics_description.py
    │   └── relationship_recommendation.py
    └── services/
        ├── ask.py
        ├── ask_details.py
        ├── sql_answer.py
        ├── sql_explanation.py
        ├── sql_expansion.py
        ├── sql_regeneration.py
        ├── semantics_preparation.py
        ├── semantics_description.py
        └── relationship_recommendation.py
```

### API Endpoints (9 groups under /v1/)

| Endpoint | Methods | Description |
|----------|---------|-------------|
| `/asks` | POST, PATCH, GET | Natural language to SQL |
| `/ask-details` | POST, GET | SQL breakdown analysis |
| `/sql-answers` | POST, GET | SQL answer generation |
| `/sql-explanations` | POST, GET | SQL explanation |
| `/sql-expansions` | POST, PATCH, GET | SQL expansion |
| `/sql-regenerations` | POST, GET | SQL regeneration |
| `/semantics-preparations` | POST, GET | MDL indexing |
| `/semantics-descriptions` | POST, GET | Semantic description |
| `/relationship-recommendations` | POST, GET | Relationship recommendation |

All endpoints follow the same async pattern: POST initiates a background task and returns a query_id, GET polls for results.

## 2. Design Goals

1. **API Compatibility**: Expose identical REST API endpoints with identical request/response schemas
2. **Architecture Optimization**: Leverage Go's concurrency model (goroutines, errgroup, context cancellation) to improve internal architecture
3. **TDD**: Write tests first, then implement
4. **Use Pantheon SDK**: Replace Haystack AI with `github.com/odysseythink/pantheon` for LLM/Embedding calls, extending it where needed
5. **No Python Dependencies**: Pure Go implementation
6. **No Langfuse**: Omit observability layer
7. **Qdrant Go SDK**: Use official `qdrant/go-client` for vector storage
8. **Go-native SQL Processing**: Replace sqlglot with custom lightweight implementation

## 3. Architecture: Layered Monolith

### 3.1 Directory Structure

```
go-wren-ai-service/
├── cmd/
│   └── server/
│       └── main.go                    # Entry point
├── internal/
│   ├── config/
│   │   └── config.go                  # Configuration loading (env + YAML)
│   ├── core/
│   │   ├── pipeline.go                # Pipeline interface + PipelineComponent
│   │   ├── provider.go                # LLMProvider, EmbedderProvider, DocStoreProvider, Engine interfaces
│   │   └── engine.go                  # Engine interface + SQL utility types
│   ├── provider/
│   │   ├── registry.go                # Provider registry (replaces Python loader.py)
│   │   ├── llm/
│   │   │   └── pantheon.go            # LLM provider based on Pantheon core.LanguageModel
│   │   ├── embedder/
│   │   │   └── pantheon.go            # Embedding provider based on Pantheon embed.Provider
│   │   ├── docstore/
│   │   │   └── qdrant.go              # Document store based on Qdrant Go SDK
│   │   └── engine/
│   │       ├── wren_ui.go             # WrenUI HTTP engine
│   │       ├── wren_ibis.go           # WrenIbis HTTP engine
│   │       └── wren_engine.go         # WrenEngine HTTP engine
│   ├── pipeline/
│   │   ├── generation/
│   │   │   ├── sql_generation.go      # SQL generation pipeline
│   │   │   ├── sql_correction.go      # SQL correction pipeline
│   │   │   ├── sql_explanation.go     # SQL explanation pipeline
│   │   │   ├── sql_expansion.go       # SQL expansion pipeline
│   │   │   ├── sql_regeneration.go    # SQL regeneration pipeline
│   │   │   ├── sql_summary.go         # SQL summary pipeline
│   │   │   ├── sql_answer.go          # SQL answer pipeline
│   │   │   ├── sql_breakdown.go       # SQL breakdown pipeline
│   │   │   ├── followup_sql.go        # Follow-up SQL generation pipeline
│   │   │   ├── semantics_desc.go      # Semantics description pipeline
│   │   │   └── relationship_rec.go    # Relationship recommendation pipeline
│   │   ├── indexing/
│   │   │   └── indexing.go            # MDL indexing pipeline
│   │   ├── retrieval/
│   │   │   ├── retrieval.go           # Document retrieval pipeline
│   │   │   └── historical.go          # Historical question retrieval
│   │   └── common/
│   │       ├── prompt.go              # Prompt templates and construction
│   │       └── postprocessor.go       # SQL generation post-processing
│   ├── service/
│   │   ├── container.go               # ServiceContainer (dependency injection)
│   │   ├── ask.go                     # Ask business logic
│   │   ├── ask_details.go             # AskDetails business logic
│   │   ├── sql_answer.go              # SQL Answer business logic
│   │   ├── sql_explanation.go         # SQL Explanation business logic
│   │   ├── sql_expansion.go           # SQL Expansion business logic
│   │   ├── sql_regeneration.go        # SQL Regeneration business logic
│   │   ├── semantics_prep.go          # Semantics Preparation business logic
│   │   ├── semantics_desc.go          # Semantics Description business logic
│   │   └── relationship_rec.go        # Relationship Recommendation business logic
│   ├── handler/
│   │   ├── router.go                  # Route registration
│   │   ├── ask.go                     # /v1/asks handlers
│   │   ├── ask_details.go             # /v1/ask-details handlers
│   │   ├── sql_answers.go             # /v1/sql-answers handlers
│   │   ├── sql_explanations.go        # /v1/sql-explanations handlers
│   │   ├── sql_expansions.go          # /v1/sql-expansions handlers
│   │   ├── sql_regenerations.go       # /v1/sql-regenerations handlers
│   │   ├── semantics_prep.go          # /v1/semantics-preparations handlers
│   │   ├── semantics_desc.go          # /v1/semantics-descriptions handlers
│   │   └── relationship_rec.go        # /v1/relationship-recommendations handlers
│   └── model/
│       ├── ask.go                     # Ask request/response models
│       ├── sql.go                     # SQL-related request/response models
│       ├── semantics.go               # Semantics-related request/response models
│       └── common.go                  # Common models (errors, status, etc.)
├── pkg/
│   ├── sqlutil/
│   │   ├── clean.go                   # SQL cleaning (clean_generation_result)
│   │   ├── quotes.go                  # Identifier quoting (add_quotes)
│   │   ├── limit.go                   # LIMIT statement removal
│   │   └── transpile.go              # SQL dialect transpilation
│   └── mdl/
│       ├── parser.go                  # MDL JSON parsing
│       ├── ddl.go                     # MDL to DDL conversion
│       └── schema.go                  # MDL data structures
├── config.yaml                        # Default configuration
├── go.mod
└── go.sum
```

### 3.2 Dependency Mapping

| Function | Python Dependency | Go Replacement |
|----------|------------------|----------------|
| HTTP Framework | FastAPI + Uvicorn | chi + net/http |
| LLM Calls | Haystack + OpenAI SDK | pantheon (core.LanguageModel) |
| Embedding | Haystack + OpenAI SDK | pantheon (embed.Provider) |
| Vector DB | haystack-ai + qdrant-haystack | qdrant/go-client |
| SQL Parsing | sqlglot | pkg/sqlutil (custom) |
| JSON | orjson | encoding/json + goccy/go-json |
| Configuration | pydantic-settings + YAML | caarlos0/env + gopkg.in/yaml.v3 |
| Observability | Langfuse | Not implemented |
| Caching | cachetools.TTLCache | patrickmn/go-cache |
| Background Tasks | FastAPI BackgroundTasks | goroutine + sync.Map |
| Pipeline Orchestration | Hamilton AsyncDriver | Direct method chaining in Pipeline structs |
| Prompt Templates | Haystack PromptBuilder (Jinja2) | text/template |

### 3.3 Core Abstractions

```go
// core/pipeline.go
type Pipeline interface {
    Run(ctx context.Context, input any) (any, error)
}

type PipelineComponent struct {
    LLMProvider      LLMProvider
    EmbedderProvider EmbedderProvider
    DocStoreProvider DocStoreProvider
    Engine           Engine
}
```

```go
// core/provider.go
type LLMProvider interface {
    GetGenerator(ctx context.Context, opts GeneratorOpts) (Generator, error)
    GetModel() string
    GetModelKwargs() map[string]any
}

type Generator interface {
    Run(ctx context.Context, prompt string) (*GenerateResult, error)
}

type GeneratorOpts struct {
    SystemPrompt     string
    GenerationKwargs map[string]any
}

type GenerateResult struct {
    Replies []string
    Meta    []map[string]any
}

type EmbedderProvider interface {
    GetTextEmbedder(ctx context.Context) (TextEmbedder, error)
    GetDocumentEmbedder(ctx context.Context) (DocumentEmbedder, error)
    GetModel() string
    GetDimensions() int
}

type TextEmbedder interface {
    Run(ctx context.Context, text string) (*EmbedResult, error)
}

type DocumentEmbedder interface {
    Run(ctx context.Context, docs []Document) (*DocEmbedResult, error)
}

type DocStoreProvider interface {
    GetStore(opts StoreOpts) DocumentStore
    GetRetriever(store DocumentStore, topK int) Retriever
}

type DocumentStore interface {
    WriteDocuments(ctx context.Context, docs []Document, policy WritePolicy) (int, error)
    DeleteDocuments(ctx context.Context, filters map[string]any) error
    CountDocuments(ctx context.Context, filters map[string]any) (int, error)
}

type Retriever interface {
    Run(ctx context.Context, queryEmbedding []float32, filters map[string]any) (*RetrievalResult, error)
}

type Engine interface {
    ExecuteSQL(ctx context.Context, sql string, opts EngineOpts) (*EngineResult, error)
}
```

```go
// core/engine.go
type EngineOpts struct {
    ProjectID string
    DryRun    bool
    Limit     int
}

type EngineResult struct {
    Success bool
    Data    map[string]any
    Error   string
}
```

### 3.4 Provider Implementations

**LLM Provider** (based on Pantheon):
- `PantheonLLMProvider` wraps `pantheon/core.Provider` + `LanguageModel`
- `PantheonGenerator` wraps a `LanguageModel` instance with system prompt and generation kwargs
- Structured output (JSON Schema) via `LanguageModel.GenerateObject()`
- Rate limit retry via Pantheon's `extensions/retry`

**Embedder Provider** (based on Pantheon):
- `PantheonEmbedderProvider` wraps `pantheon/extensions/embed.Provider` + `EmbeddingModel`
- `PantheonTextEmbedder` wraps `EmbeddingModel.Embed()` for single text
- `PantheonDocumentEmbedder` wraps `EmbeddingModel.Embed()` with batch processing logic

**Document Store** (based on Qdrant Go SDK):
- `QdrantProvider` creates `QdrantDocumentStore` instances
- `QdrantDocumentStore` wraps `qdrant.Client` with:
  - Collection management (create, configure HNSW/quantization)
  - Document ↔ Point conversion
  - Filter construction (Qdrant conditions)
  - Binary quantization optimization when embedding_dim >= 1024
- `QdrantRetriever` wraps vector search with filter support

**Engine Providers** (HTTP clients):
- `WrenUIEngine`: POST to wren-ui GraphQL API (`previewSql` mutation)
- `WrenIbisEngine`: POST to ibis-server REST API (`/v2/connector/{source}/query`)
- `WrenEngine`: GET to wren-engine REST API (`/v1/mdl/dry-run` or `/v1/mdl/preview`)
- All use `*http.Client` with configurable timeouts

### 3.5 Pipeline Orchestration

Each pipeline is a struct implementing `core.Pipeline`. Internal logic uses direct method chaining instead of Hamilton's function DAG. Go's `errgroup` enables concurrent steps.

**SQL Generation Pipeline** (representative example):
1. PromptBuilder builds prompt from template + context
2. Generator calls LLM (Pantheon LanguageModel)
3. PostProcessor cleans output, adds quotes, validates SQL via engine dry-run
4. Returns valid/invalid SQL results

**Indexing Pipeline**:
1. Clean old documents from stores
2. Validate MDL JSON
3. Three concurrent branches (errgroup):
   - DDL conversion → embed → write to dbschema store
   - Table description extraction → embed → write to table_descriptions store
   - View chunking → embed → write to view_questions store

**Retrieval Pipeline**:
1. Embed query text
2. Retrieve relevant tables from table_descriptions store
3. Retrieve DB schema for matched tables
4. Build DDL from schema
5. LLM column selection (structured output)
6. Construct final retrieval context

### 3.6 Service Layer

```go
type ServiceContainer struct {
    AskService              *AskService
    AskDetailsService       *AskDetailsService
    SQLAnswerService        *SQLAnswerService
    SQLExplanationService   *SQLExplanationService
    SQLExpansionService     *SQLExpansionService
    SQLRegenerationService  *SQLRegenerationService
    SemanticsPrepService    *SemanticsPrepService
    SemanticsDescService    *SemanticsDescService
    RelationshipRecService  *RelationshipRecService
}
```

Each service uses `go-cache` (TTL cache) to store async task results, keyed by queryID. Background tasks run in goroutines. `context.WithCancel` enables true task cancellation (improvement over Python which only sets status).

### 3.7 Handler Layer

chi router with CORS middleware. Each handler follows the pattern:
- Decode request JSON
- Call service method (which may launch a goroutine)
- Return response JSON

All 9 endpoint groups map 1:1 to Python routers.

### 3.8 Model Layer

Go structs with JSON tags matching Python's Pydantic model JSON serialization exactly. This ensures API compatibility.

### 3.9 SQL Utilities (pkg/sqlutil)

**CleanGenerationResult**: Normalize whitespace, remove markdown code fences, semicolons, triple quotes.

**AddQuotes**: Custom lightweight Trino SQL tokenizer that:
- Tokenizes SQL into keywords, identifiers, strings, numbers
- Adds double quotes to unquoted identifiers
- Does not build a full AST (sufficient for LLM-generated Trino SQL)

**RemoveLimitStatement**: Regex-based LIMIT clause removal.

### 3.10 MDL Package (pkg/mdl)

- `MDL` struct mirrors the Python MDL JSON structure
- `ParseMDL` validates and parses MDL JSON
- `ConvertToDDL` replicates Python's `DDLConverter._get_ddl_commands()`
- `ConvertToTableDescriptions` replicates Python's `TableDescriptionConverter`
- View chunking logic from Python's `ViewChunker`

### 3.11 Configuration

Environment variables with YAML file override. Same variable names as Python version for Docker compatibility. Uses `caarlos0/env` for struct-based env parsing.

### 3.12 Startup Flow

1. Load config (env vars + optional YAML)
2. Initialize providers (LLM, Embedder, DocStore, Engine)
3. Create ServiceContainer with pipelines
4. Optionally wait for Qdrant readiness, then force-deploy
5. Start HTTP server with graceful shutdown

## 4. Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Layered monolith over microservices | Project scope (11 pipelines, 9 API groups) does not warrant microservice complexity |
| Pantheon SDK instead of raw OpenAI client | Unified interface across providers, built-in retry/fallback, structured output support |
| Direct method chaining over DAG engine | 11 pipelines don't justify a generic DAG framework; method chaining is more readable and testable |
| Custom SQL tokenizer over sqlglot port | sqlglot is 30k+ lines; WrenAI only needs identifier quoting for LLM-generated Trino SQL |
| go-cache over custom TTL cache | Battle-tested, same semantics as Python's cachetools.TTLCache |
| chi over gin | Lightweight, net/http compatible, idiomatic Go |
| No Langfuse | Explicitly excluded per requirements |
| TDD approach | Each module tested before implementation |

## 5. Out of Scope

- wren-engine, wren-ui, wren-launcher, wren-mdl (separate images)
- Langfuse observability integration
- Haystack/Hamilton Python dependency
- Eval/demo tools from Python version
- Streaming API (Python version does not stream responses)
- Docker image build (focus on Go source code)

## 6. Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| Pantheon SDK missing features (e.g., specific provider) | Extend Pantheon library source code as needed |
| Custom SQL tokenizer may not handle all edge cases | Start with Trino-focused tokenizer; add test cases from Python version's SQL corpus |
| Qdrant Go SDK API differences from Python | Write integration tests against same Qdrant instance |
| Prompt templates (Jinja2 → Go text/template) subtle differences | Character-by-character comparison of template output |
| Async task state management (goroutine lifecycle) | Use context cancellation + go-cache TTL; add monitoring for leaked goroutines |
