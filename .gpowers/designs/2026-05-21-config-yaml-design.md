# Design: Make config.yaml Fully Functional

## Context

The current Go rewrite of `wren-ai-service` has a placeholder `applyYAML()` in `internal/config/config.go` — it reads `config.yaml` but does nothing with it. All provider initialization is hard-coded in `cmd/server/main.go` to OpenAI LLM + OpenAI Embedder + Qdrant + a single engine. This means:

- No multi-provider support (Azure OpenAI, Ollama are unavailable).
- No per-pipeline provider selection.
- The `config.yaml` format from the Python reference (v0.9.0) is completely ignored.

The goal is to bring the Go service to functional parity with the Python 0.9.0 Docker image regarding configuration.

## Scope

**In scope:**
- Multi-document YAML parsing (`---` separators).
- Provider registry that is actually used.
- LLM providers: `openai_llm`, `azure_openai_llm`, `ollama_llm`.
- Embedder providers: `openai_embedder`, `ollama_embedder`. (Azure embedder is **out of scope** for now.)
- Engine providers: `wren_ui`, `wren_ibis`, `wren_engine`.
- Document store provider: `qdrant`.
- Per-pipeline `PipelineComponent` assembly (each pipeline can reference its own LLM/embedder/engine/docstore).
- Env-var fallback when `config.yaml` is absent.

**Out of scope:**
- `azure_openai_embedder` (Pantheon Go SDK does not expose EmbeddingModel for Azure).
- Additional providers beyond those listed above.
- Runtime hot-reload of `config.yaml`.

## Architecture

```
config.yaml
    │
    ▼
internal/config/config.go  ──►  RawConfig  ──►  service.GenerateComponents()
                                                      │
                                                      ▼
                                          map[string]core.PipelineComponent
                                                      │
                                                      ▼
                                          service.NewContainer(components, cfg)
                                                      │
                                                      ▼
                                          handler.NewRouter(container)
                                                      │
                                                      ▼
                                                 HTTP Server
```

## Data Structures

```go
// internal/config/config.go

type Config struct {
    // Service
    Port              int    `env:"WREN_AI_SERVICE_PORT" envDefault:"8000"`
    Host              string `env:"WREN_AI_SERVICE_HOST" envDefault:"127.0.0.1"`
    LoggingLevel      string `env:"LOGGING_LEVEL" envDefault:"INFO"`
    EnableTimer       bool   `env:"ENABLE_TIMER" envDefault:"false"`
    ShouldForceDeploy bool   `env:"SHOULD_FORCE_DEPLOY" envDefault:"false"`

    // LLM (env fallbacks, still needed when config.yaml is absent)
    LLMProvider      string  `env:"LLM_PROVIDER" envDefault:"openai_llm"`
    LLMOpenAIAPIKey  string  `env:"LLM_OPENAI_API_KEY"`
    LLMOpenAIAPIBase string  `env:"LLM_OPENAI_API_BASE" envDefault:"https://api.openai.com/v1"`
    GenerationModel  string  `env:"GENERATION_MODEL" envDefault:"gpt-4o-mini"`
    LLMTimeout       float64 `env:"LLM_TIMEOUT" envDefault:"120"`

    // Embedder (env fallbacks)
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
    ColumnIndexingBatchSize  int `env:"COLUMN_INDEXING_BATCH_SIZE" envDefault:"50"`
    TableRetrievalSize       int `env:"TABLE_RETRIEVAL_SIZE" envDefault:"10"`
    TableColumnRetrievalSize int `env:"TABLE_COLUMN_RETRIEVAL_SIZE" envDefault:"1000"`
    QueryCacheTTL            int `env:"QUERY_CACHE_TTL" envDefault:"120"`

    // Parsed from config.yaml (nil when absent or empty)
    Raw *RawConfig
}

type RawConfig struct {
    LLMs           map[string]ProviderEntry
    Embedders      map[string]ProviderEntry
    DocumentStores map[string]ProviderEntry
    Engines        map[string]ProviderEntry
    Pipelines      map[string]PipelineEntry
}

type ProviderEntry struct {
    Provider string         // the provider type name, e.g. "openai_llm"
    Config   map[string]any // flattened fields from YAML
}

type PipelineEntry struct {
    LLM           string
    Embedder      string
    DocumentStore string
    Engine        string
}
```

## Provider Registry

```go
// internal/provider/registry.go

var (
    llmFactories      = map[string]func(cfg map[string]any) (core.LLMProvider, error){}
    embedderFactories = map[string]func(cfg map[string]any) (core.EmbedderProvider, error){}
    engineFactories   = map[string]func(cfg map[string]any) (core.Engine, error){}
    docStoreFactories = map[string]func(cfg map[string]any) (core.DocStoreProvider, error){}
)

func RegisterLLM(name string, f func(cfg map[string]any) (core.LLMProvider, error))
func RegisterEmbedder(name string, f func(cfg map[string]any) (core.EmbedderProvider, error))
func RegisterEngine(name string, f func(cfg map[string]any) (core.Engine, error))
func RegisterDocStore(name string, f func(cfg map[string]any) (core.DocStoreProvider, error))

func GetLLMFactory(name string) (func(cfg map[string]any) (core.LLMProvider, error), bool)
func GetEmbedderFactory(name string) (func(cfg map[string]any) (core.EmbedderProvider, error), bool)
func GetEngineFactory(name string) (func(cfg map[string]any) (core.Engine, error), bool)
func GetDocStoreFactory(name string) (func(cfg map[string]any) (core.DocStoreProvider, error), bool)
```

## Provider Implementations

### LLM Factories

| Name | Factory File | Pantheon Provider | Notes |
|------|--------------|-------------------|-------|
| `openai_llm` | `internal/provider/llm/openai.go` | `openai.New()` | Supports `api_key`, `api_base`, `model`, `kwargs`, `timeout` |
| `azure_openai_llm` | `internal/provider/llm/azure.go` | `azure.New()` | Supports `api_key`, `api_base` (Azure endpoint URL), `api_version`, `model`, `kwargs`, `timeout`. Factory extracts `resourceName` + `deployment` from `api_base` URL (e.g. `https://{resource}.openai.azure.com/openai/deployments/{deployment}`); if extraction fails, falls back to `resource_name` + `deployment` fields. |
| `ollama_llm` | `internal/provider/llm/ollama.go` | `ollama.New()` | Supports `url`, `model`, `kwargs`, `timeout` |

### Embedder Factories

| Name | Factory File | Implementation | Notes |
|------|--------------|----------------|-------|
| `openai_embedder` | `internal/provider/embedder/openai.go` | `openai.New()` → `NewPantheonEmbedderProvider()` | Supports `api_key`, `api_base`, `model`, `dimension`, `timeout` |
| `ollama_embedder` | `internal/provider/embedder/ollama.go` | `openai.New(baseURL)` → `NewPantheonEmbedderProvider()` | Ollama's `/v1/embeddings` is OpenAI-compatible; `url` defaults to `http://localhost:11434` |

> **Azure embedder is intentionally out of scope.** Pantheon Go SDK does not expose `EmbeddingModel` for Azure. Adding it later requires only a new `internal/provider/embedder/azure.go` + `RegisterEmbedder` call — zero impact on existing code.

### Engine Factories

| Name | Factory File | Existing Constructor |
|------|--------------|----------------------|
| `wren_ui` | `internal/provider/engine/wren_ui.go` | `engine.NewWrenUI(endpoint)` |
| `wren_ibis` | `internal/provider/engine/wren_ibis.go` | `engine.NewWrenIbis(endpoint, source, manifest, connInfo)` |
| `wren_engine` | `internal/provider/engine/wren_engine.go` | `engine.NewWrenEngine(endpoint, manifest)` |

### Document Store Factory

| Name | Factory File | Existing Constructor |
|------|--------------|----------------------|
| `qdrant` | `internal/provider/docstore/qdrant.go` | `docstore.NewQdrantProvider(host, apiKey, embeddingDim, timeout)` |

## Component Assembly

```go
// internal/service/components.go

func GenerateComponents(cfg *config.Config) (map[string]core.PipelineComponent, error) {
    if cfg.Raw == nil || len(cfg.Raw.Pipelines) == 0 {
        return generateFromEnv(cfg)
    }

    // Instantiate all providers into lookup maps
    llms, embedders, docStores, engines := map[string]core.LLMProvider{}, ...
    // ... type-safe factory calls with error handling

    // Assemble per-pipeline components
    components := make(map[string]core.PipelineComponent)
    for name, pipe := range cfg.Raw.Pipelines {
        comp := core.PipelineComponent{}
        if pipe.LLM != "" {
            comp.LLMProvider = llms[pipe.LLM]
        }
        if pipe.Embedder != "" {
            comp.EmbedderProvider = embedders[pipe.Embedder]
        }
        if pipe.DocumentStore != "" {
            comp.DocStoreProvider = docStores[pipe.DocumentStore]
        }
        if pipe.Engine != "" {
            comp.Engine = engines[pipe.Engine]
        }
        components[name] = comp
    }
    return components, nil
}
```

### Env-Var Fallback (`generateFromEnv`)

When `config.yaml` is absent, the system behaves exactly as it does today: read env vars, create a single set of providers, and map all 14 pipeline names to the same `PipelineComponent`. This preserves backward compatibility for Docker deployments that rely solely on environment variables.

### Pipeline-to-Service Mapping

There are 14 named pipelines. Each service receives the component(s) it needs by pipeline name:

| Pipeline | Used By | Required Providers |
|----------|---------|-------------------|
| `indexing` | SemanticsPreparation | embedder + docstore |
| `retrieval` | Ask, SQLExpansion | llm + embedder + docstore |
| `historical_question` | Ask | embedder + docstore |
| `sql_generation` | Ask | llm + engine |
| `sql_correction` | Ask, SQLExpansion | llm + engine |
| `followup_sql_generation` | Ask | llm + engine |
| `sql_summary` | Ask, SQLExpansion | llm |
| `sql_answer` | SQLAnswer | llm |
| `sql_breakdown` | AskDetails | llm + engine |
| `sql_expansion` | SQLExpansion | llm + engine |
| `sql_explanation` | SQLExplanation | llm |
| `sql_regeneration` | SQLRegeneration | llm + engine |
| `semantics_description` | SemanticsDescription | llm |
| `relationship_recommendation` | RelationshipRecommendation | llm + engine |

## Service Container

`NewContainer` changes from:

```go
func NewContainer(components core.PipelineComponent, cfg *config.Config) *Container
```

To:

```go
func NewContainer(components map[string]core.PipelineComponent, cfg *config.Config) *Container
```

Each pipeline is created with its named component:

```go
get := func(name string) core.PipelineComponent {
    c, ok := components[name]
    if !ok { panic(fmt.Sprintf("pipeline component %q not found", name)) }
    return c
}

indexingPipe := indexing.NewIndexing(get("indexing"), cfg.ColumnIndexingBatchSize)
retrievalPipe := retrieval.NewRetrieval(get("retrieval"), ...)
// ... etc
```

## Error Handling

| Scenario | Behavior |
|----------|----------|
| `config.yaml` absent | Silent fallback to env vars |
| `config.yaml` malformed | `log.Fatalf` on startup with file:line info |
| Unknown provider name | `log.Fatalf` with list of registered providers |
| Provider init failure | `log.Fatalf` with underlying error |
| Pipeline references missing provider | `log.Fatalf` with missing ID |
| Pipeline missing required provider | `log.Fatalf` (e.g. `sql_generation` without `llm`) |

**Principle: fail fast at startup.** No degraded runtime states.

## Testing Plan

1. **Config parsing tests** (`internal/config/config_test.go`): multi-doc YAML, all five types, missing optional fields.
2. **Factory registration tests** (`internal/provider/registry_test.go`): verify each factory is registered and returns correct interface type.
3. **Factory unit tests**: each new factory file gets a test using mock inputs (no live API calls).
4. **Component assembly tests** (`internal/service/components_test.go` **new**): test `generateFromEnv` produces all 14 pipelines; test YAML-driven assembly with mocked factories.
5. **Integration test** (`integration_test.go`): update to use new `NewContainer` signature.

## File Changes

| Path | Action | Description |
|------|--------|-------------|
| `internal/config/config.go` | Modify | Add `RawConfig`, `ProviderEntry`, `PipelineEntry`; implement `ParseYAML` |
| `internal/config/config_test.go` | Modify | Add YAML parsing tests |
| `internal/provider/registry.go` | Modify | Add typed factories + getters |
| `internal/provider/registry_test.go` | Modify | Add typed registry tests |
| `internal/provider/llm/openai.go` | **Create** | `openai_llm` factory |
| `internal/provider/llm/azure.go` | **Create** | `azure_openai_llm` factory |
| `internal/provider/llm/ollama.go` | **Create** | `ollama_llm` factory |
| `internal/provider/embedder/openai.go` | **Create** | `openai_embedder` factory |
| `internal/provider/embedder/ollama.go` | **Create** | `ollama_embedder` factory |
| `internal/provider/engine/wren_ui.go` | Modify | Add `init()` factory registration |
| `internal/provider/engine/wren_ibis.go` | Modify | Add `init()` factory registration |
| `internal/provider/engine/wren_engine.go` | Modify | Add `init()` factory registration |
| `internal/provider/docstore/qdrant.go` | Modify | Add `init()` factory registration |
| `internal/service/components.go` | **Create** | `GenerateComponents()` + `generateFromEnv()` |
| `internal/service/container.go` | Modify | Adapt to `map[string]core.PipelineComponent` |
| `cmd/server/main.go` | Modify | Remove hard-coded init, call `GenerateComponents` |
| `integration_test.go` | Modify | Adapt `NewContainer` call |

**Total: 8 new files, 10 modified files.**
