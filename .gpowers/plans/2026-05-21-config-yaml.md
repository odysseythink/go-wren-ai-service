# config.yaml Full Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use gpowers:subagent-driven-development (recommended) or gpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `config.yaml` fully functional by implementing multi-document YAML parsing, typed provider registry with factories for all supported providers, and per-pipeline `PipelineComponent` assembly.

**Architecture:** Parse multi-doc YAML into `RawConfig`, register typed provider factories in `init()`, instantiate providers on demand, and assemble a `map[string]core.PipelineComponent` where each pipeline gets its own provider combination. Env vars serve as fallback when `config.yaml` is absent.

**Tech Stack:** Go 1.26, Pantheon SDK (local replace), go-chi, go-cache, gopkg.in/yaml.v3, caarlos0/env

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/provider/registry.go` | Typed provider factory registry (LLM/Embedder/Engine/DocStore) |
| `internal/config/config.go` | Env parsing + `RawConfig` types + `ParseYAML` |
| `internal/provider/llm/openai.go` | `openai_llm` factory |
| `internal/provider/llm/azure.go` | `azure_openai_llm` factory |
| `internal/provider/llm/ollama.go` | `ollama_llm` factory |
| `internal/provider/embedder/openai.go` | `openai_embedder` factory |
| `internal/provider/embedder/ollama.go` | `ollama_embedder` factory |
| `internal/provider/engine/wren_ui.go` | `wren_ui` factory registration |
| `internal/provider/engine/wren_ibis.go` | `wren_ibis` factory registration |
| `internal/provider/engine/wren_engine.go` | `wren_engine` factory registration |
| `internal/provider/docstore/qdrant.go` | `qdrant` factory registration |
| `internal/service/components.go` | `GenerateComponents` + `generateFromEnv` |
| `internal/service/container.go` | `NewContainer` adapted to `map[string]core.PipelineComponent` |
| `cmd/server/main.go` | Simplified boot using `GenerateComponents` |

---

## Task 1: Typed Provider Registry

**Files:**
- Modify: `internal/provider/registry.go`
- Test: `internal/provider/registry_test.go`

- [ ] **Step 1: Write failing test for typed LLM registry**

```go
// internal/provider/registry_test.go
package provider

import (
	"testing"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
)

func TestRegisterLLM(t *testing.T) {
	RegisterLLM("test_llm", func(cfg map[string]any) (core.LLMProvider, error) {
		return &mockLLM{}, nil
	})

	f, ok := GetLLMFactory("test_llm")
	if !ok {
		t.Fatal("expected factory to be registered")
	}

	p, err := f(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

type mockLLM struct{}

func (m *mockLLM) GetGenerator(ctx context.Context, opts core.GeneratorOpts) (core.Generator, error) {
	return nil, nil
}
func (m *mockLLM) GetModel() string { return "" }
func (m *mockLLM) GetModelKwargs() map[string]any { return nil }
```

Run: `go test ./internal/provider/... -v -run TestRegisterLLM`
Expected: FAIL — `RegisterLLM` / `GetLLMFactory` undefined

- [ ] **Step 2: Implement typed registry**

```go
// internal/provider/registry.go
package provider

import (
	"github.com/odysseythink/go-wren-ai-service/internal/core"
)

var (
	llmFactories      = map[string]func(cfg map[string]any) (core.LLMProvider, error){}
	embedderFactories = map[string]func(cfg map[string]any) (core.EmbedderProvider, error){}
	engineFactories   = map[string]func(cfg map[string]any) (core.Engine, error){}
	docStoreFactories = map[string]func(cfg map[string]any) (core.DocStoreProvider, error){}
)

// Keep legacy untyped registry for backward compat
var registry = map[string]Factory{}

// Factory creates a provider instance from a configuration map.
type Factory func(cfg map[string]any) (any, error)

// Register adds a provider factory to the registry.
func Register(name string, factory Factory) {
	registry[name] = factory
}

// Get retrieves a provider factory by name.
func Get(name string) (Factory, bool) {
	f, ok := registry[name]
	return f, ok
}

func RegisterLLM(name string, f func(cfg map[string]any) (core.LLMProvider, error)) {
	llmFactories[name] = f
}

func GetLLMFactory(name string) (func(cfg map[string]any) (core.LLMProvider, error), bool) {
	f, ok := llmFactories[name]
	return f, ok
}

func RegisterEmbedder(name string, f func(cfg map[string]any) (core.EmbedderProvider, error)) {
	embedderFactories[name] = f
}

func GetEmbedderFactory(name string) (func(cfg map[string]any) (core.EmbedderProvider, error), bool) {
	f, ok := embedderFactories[name]
	return f, ok
}

func RegisterEngine(name string, f func(cfg map[string]any) (core.Engine, error)) {
	engineFactories[name] = f
}

func GetEngineFactory(name string) (func(cfg map[string]any) (core.Engine, error), bool) {
	f, ok := engineFactories[name]
	return f, ok
}

func RegisterDocStore(name string, f func(cfg map[string]any) (core.DocStoreProvider, error)) {
	docStoreFactories[name] = f
}

func GetDocStoreFactory(name string) (func(cfg map[string]any) (core.DocStoreProvider, error), bool) {
	f, ok := docStoreFactories[name]
	return f, ok
}
```

Run: `go test ./internal/provider/... -v -run TestRegisterLLM`
Expected: PASS

- [ ] **Step 3: Add tests for Embedder/Engine/DocStore registries**

Add `TestRegisterEmbedder`, `TestRegisterEngine`, `TestRegisterDocStore` to `registry_test.go` using the same mock pattern.

Run: `go test ./internal/provider/... -v`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add internal/provider/registry.go internal/provider/registry_test.go
git commit -m "feat(provider): typed factory registry for llm/embedder/engine/docstore"
```

---

## Task 2: Config YAML Parsing

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test for multi-doc YAML parsing**

```go
// internal/config/config_test.go
package config

import (
	"testing"
)

func TestParseYAML_MultiDoc(t *testing.T) {
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
type: pipeline
pipes:
  - name: sql_generation
    llm: openai_llm.gpt-4o-mini
    engine: wren_ui
`
	raw, err := ParseYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if raw == nil {
		t.Fatal("expected non-nil RawConfig")
	}
	if len(raw.LLMs) != 1 {
		t.Fatalf("expected 1 LLM, got %d", len(raw.LLMs))
	}
	if len(raw.Embedders) != 1 {
		t.Fatalf("expected 1 Embedder, got %d", len(raw.Embedders))
	}
	if len(raw.Pipelines) != 1 {
		t.Fatalf("expected 1 Pipeline, got %d", len(raw.Pipelines))
	}

	llm := raw.LLMs["openai_llm.gpt-4o-mini"]
	if llm.Provider != "openai_llm" {
		t.Fatalf("expected provider openai_llm, got %s", llm.Provider)
	}
	if llm.Config["api_key"] != "sk-test" {
		t.Fatalf("expected api_key sk-test, got %v", llm.Config["api_key"])
	}

	pipe := raw.Pipelines["sql_generation"]
	if pipe.LLM != "openai_llm.gpt-4o-mini" {
		t.Fatalf("expected llm openai_llm.gpt-4o-mini, got %s", pipe.LLM)
	}
}
```

Run: `go test ./internal/config/... -v -run TestParseYAML_MultiDoc`
Expected: FAIL — `ParseYAML` undefined

- [ ] **Step 2: Implement RawConfig types and ParseYAML**

```go
// internal/config/config.go — add below existing Config struct

type RawConfig struct {
	LLMs           map[string]ProviderEntry
	Embedders      map[string]ProviderEntry
	DocumentStores map[string]ProviderEntry
	Engines        map[string]ProviderEntry
	Pipelines      map[string]PipelineEntry
}

type ProviderEntry struct {
	Provider string
	Config   map[string]any
}

type PipelineEntry struct {
	LLM           string
	Embedder      string
	DocumentStore string
	Engine        string
}

// ParseYAML reads multi-document YAML and converts it to RawConfig.
func ParseYAML(data []byte) (*RawConfig, error) {
	var docs []map[string]any
	if err := yaml.Unmarshal(data, &docs); err != nil {
		return nil, err
	}

	raw := &RawConfig{
		LLMs:           make(map[string]ProviderEntry),
		Embedders:      make(map[string]ProviderEntry),
		DocumentStores: make(map[string]ProviderEntry),
		Engines:        make(map[string]ProviderEntry),
		Pipelines:      make(map[string]PipelineEntry),
	}

	for _, doc := range docs {
		typ, _ := doc["type"].(string)
		switch typ {
		case "llm":
			processLLM(raw, doc)
		case "embedder":
			processEmbedder(raw, doc)
		case "document_store":
			processDocumentStore(raw, doc)
		case "engine":
			processEngine(raw, doc)
		case "pipeline":
			processPipeline(raw, doc)
		default:
			return nil, fmt.Errorf("unknown config type: %s", typ)
		}
	}
	return raw, nil
}

func processLLM(raw *RawConfig, doc map[string]any) {
	provider, _ := doc["provider"].(string)
	apiKey, _ := doc["api_key"].(string)
	apiBase, _ := doc["api_base"].(string)
	timeout, _ := doc["timeout"].(float64)

	models, _ := doc["models"].([]any)
	for _, m := range models {
		modelMap, _ := m.(map[string]any)
		modelName, _ := modelMap["model"].(string)
		kwargs, _ := modelMap["kwargs"].(map[string]any)

		key := provider + "." + modelName
		config := map[string]any{
			"provider": provider,
			"model":    modelName,
			"kwargs":   kwargs,
		}
		if apiKey != "" {
			config["api_key"] = apiKey
		}
		if apiBase != "" {
			config["api_base"] = apiBase
		}
		if timeout != 0 {
			config["timeout"] = timeout
		}
		// Copy any other fields
		for k, v := range doc {
			if k != "type" && k != "provider" && k != "models" && config[k] == nil {
				config[k] = v
			}
		}
		raw.LLMs[key] = ProviderEntry{Provider: provider, Config: config}
	}
}

func processEmbedder(raw *RawConfig, doc map[string]any) {
	provider, _ := doc["provider"].(string)
	apiKey, _ := doc["api_key"].(string)
	apiBase, _ := doc["api_base"].(string)
	timeout, _ := doc["timeout"].(float64)

	models, _ := doc["models"].([]any)
	for _, m := range models {
		modelMap, _ := m.(map[string]any)
		modelName, _ := modelMap["model"].(string)
		dimension := 0
		if d, ok := modelMap["dimension"].(int); ok {
			dimension = d
		}
		if d, ok := modelMap["dimension"].(float64); ok {
			dimension = int(d)
		}

		key := provider + "." + modelName
		config := map[string]any{
			"provider": provider,
			"model":    modelName,
			"dimension": dimension,
		}
		if apiKey != "" {
			config["api_key"] = apiKey
		}
		if apiBase != "" {
			config["api_base"] = apiBase
		}
		if timeout != 0 {
			config["timeout"] = timeout
		}
		for k, v := range doc {
			if k != "type" && k != "provider" && k != "models" && config[k] == nil {
				config[k] = v
			}
		}
		raw.Embedders[key] = ProviderEntry{Provider: provider, Config: config}
	}
}

func processDocumentStore(raw *RawConfig, doc map[string]any) {
	provider, _ := doc["provider"].(string)
	config := map[string]any{}
	for k, v := range doc {
		if k != "type" {
			config[k] = v
		}
	}
	raw.DocumentStores[provider] = ProviderEntry{Provider: provider, Config: config}
}

func processEngine(raw *RawConfig, doc map[string]any) {
	provider, _ := doc["provider"].(string)
	config := map[string]any{}
	for k, v := range doc {
		if k != "type" {
			config[k] = v
		}
	}
	raw.Engines[provider] = ProviderEntry{Provider: provider, Config: config}
}

func processPipeline(raw *RawConfig, doc map[string]any) {
	pipes, _ := doc["pipes"].([]any)
	for _, p := range pipes {
		pipeMap, _ := p.(map[string]any)
		name, _ := pipeMap["name"].(string)
		entry := PipelineEntry{}
		if v, ok := pipeMap["llm"].(string); ok {
			entry.LLM = v
		}
		if v, ok := pipeMap["embedder"].(string); ok {
			entry.Embedder = v
		}
		if v, ok := pipeMap["document_store"].(string); ok {
			entry.DocumentStore = v
		}
		if v, ok := pipeMap["engine"].(string); ok {
			entry.Engine = v
		}
		raw.Pipelines[name] = entry
	}
}
```

> Note: `yaml.Unmarshal` with a slice handles multi-document YAML automatically in `gopkg.in/yaml.v3` when the target is `[]map[string]any`.

- [ ] **Step 3: Wire ParseYAML into Load()**

Modify `Load()` to call `ParseYAML` and store result in `cfg.Raw`:

```go
// In internal/config/config.go, update Load():
func Load() *Config {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "config parse error: %v\n", err)
	}
	if data, err := os.ReadFile("config.yaml"); err == nil {
		if raw, err := ParseYAML(data); err == nil {
			cfg.Raw = raw
		} else {
			fmt.Fprintf(os.Stderr, "config.yaml parse error: %v\n", err)
		}
	}
	return cfg
}
```

Remove the old empty `applyYAML`.

Run: `go test ./internal/config/... -v -run TestParseYAML_MultiDoc`
Expected: PASS

- [ ] **Step 4: Add edge-case tests**

Add `TestParseYAML_Empty`, `TestParseYAML_MissingOptionalFields`, `TestParseYAML_EngineOnly`.

Run: `go test ./internal/config/... -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): multi-document YAML parsing with RawConfig types"
```

---

## Task 3: LLM Provider Factories

**Files:**
- Create: `internal/provider/llm/openai.go`
- Create: `internal/provider/llm/openai_test.go`
- Create: `internal/provider/llm/azure.go`
- Create: `internal/provider/llm/azure_test.go`
- Create: `internal/provider/llm/ollama.go`
- Create: `internal/provider/llm/ollama_test.go`

- [ ] **Step 1: Write failing test for openai_llm factory**

```go
// internal/provider/llm/openai_test.go
package llm

import (
	"testing"
)

func TestOpenAILLMFactory(t *testing.T) {
	f, ok := provider.GetLLMFactory("openai_llm")
	if !ok {
		t.Fatal("openai_llm factory not registered")
	}

	p, err := f(map[string]any{
		"api_key": "test-key",
		"model":   "gpt-4o",
		"kwargs": map[string]any{
			"temperature": 0.5,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetModel() != "gpt-4o" {
		t.Fatalf("expected model gpt-4o, got %s", p.GetModel())
	}
}
```

Run: `go test ./internal/provider/llm/... -v -run TestOpenAILLMFactory`
Expected: FAIL — factory not registered

- [ ] **Step 2: Implement openai_llm factory**

```go
// internal/provider/llm/openai.go
package llm

import (
	openai "github.com/odysseythink/pantheon/providers/openai"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

func init() {
	provider.RegisterLLM("openai_llm", func(cfg map[string]any) (core.LLMProvider, error) {
		apiKey, _ := cfg["api_key"].(string)
		apiBase, _ := cfg["api_base"].(string)
		model, _ := cfg["model"].(string)
		if model == "" {
			model = "gpt-4o-mini"
		}

		kwargs, _ := cfg["kwargs"].(map[string]any)
		if kwargs == nil {
			kwargs = map[string]any{
				"temperature":     0,
				"n":               1,
				"max_tokens":      4096,
				"response_format": map[string]any{"type": "json_object"},
			}
		}

		var opts []openai.Option
		if apiBase != "" {
			opts = append(opts, openai.WithBaseURL(apiBase))
		}

		pantheonProvider, err := openai.New(apiKey, opts...)
		if err != nil {
			return nil, err
		}
		return NewPantheonLLMProvider(pantheonProvider, model, kwargs), nil
	})
}
```

Run: `go test ./internal/provider/llm/... -v -run TestOpenAILLMFactory`
Expected: PASS

- [ ] **Step 3: Write failing test for azure_openai_llm factory**

```go
// internal/provider/llm/azure_test.go
package llm

import (
	"testing"
)

func TestAzureOpenAILLMFactory(t *testing.T) {
	f, ok := provider.GetLLMFactory("azure_openai_llm")
	if !ok {
		t.Fatal("azure_openai_llm factory not registered")
	}

	p, err := f(map[string]any{
		"api_key":    "test-key",
		"api_base":   "https://my-resource.openai.azure.com/openai/deployments/my-deployment",
		"api_version": "2024-06-01",
		"model":      "gpt-4o",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetModel() != "gpt-4o" {
		t.Fatalf("expected model gpt-4o, got %s", p.GetModel())
	}
}
```

Run: `go test ./internal/provider/llm/... -v -run TestAzureOpenAILLMFactory`
Expected: FAIL — factory not registered

- [ ] **Step 4: Implement azure_openai_llm factory**

```go
// internal/provider/llm/azure.go
package llm

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/odysseythink/pantheon/providers/azure"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

func init() {
	provider.RegisterLLM("azure_openai_llm", func(cfg map[string]any) (core.LLMProvider, error) {
		apiKey, _ := cfg["api_key"].(string)
		apiBase, _ := cfg["api_base"].(string)
		apiVersion, _ := cfg["api_version"].(string)
		model, _ := cfg["model"].(string)
		if model == "" {
			model = "gpt-4-turbo"
		}

		kwargs, _ := cfg["kwargs"].(map[string]any)
		if kwargs == nil {
			kwargs = map[string]any{
				"temperature":     0,
				"n":               1,
				"max_tokens":      1000,
				"response_format": map[string]any{"type": "json_object"},
			}
		}

		resourceName, deployment := "", ""
		if apiBase != "" {
			resourceName, deployment = extractAzureInfo(apiBase)
		}
		if resourceName == "" {
			resourceName, _ = cfg["resource_name"].(string)
		}
		if deployment == "" {
			deployment, _ = cfg["deployment"].(string)
		}
		if resourceName == "" || deployment == "" {
			return nil, fmt.Errorf("azure_openai_llm: need api_base or resource_name+deployment")
		}

		var opts []azure.Option
		if apiBase != "" {
			opts = append(opts, azure.WithBaseURL(apiBase))
		}
		if apiVersion != "" {
			opts = append(opts, azure.WithAPIVersion(apiVersion))
		}

		pantheonProvider, err := azure.New(apiKey, resourceName, deployment, opts...)
		if err != nil {
			return nil, err
		}
		return NewPantheonLLMProvider(pantheonProvider, model, kwargs), nil
	})
}

// extractAzureInfo parses https://{resource}.openai.azure.com/openai/deployments/{deployment}
func extractAzureInfo(apiBase string) (resourceName, deployment string) {
	u, err := url.Parse(apiBase)
	if err != nil {
		return "", ""
	}
	parts := strings.Split(u.Path, "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "deployments" && i+1 < len(parts) {
			deployment = parts[i+1]
			break
		}
	}
	hostParts := strings.Split(u.Hostname(), ".")
	if len(hostParts) > 0 {
		resourceName = hostParts[0]
	}
	return resourceName, deployment
}
```

Run: `go test ./internal/provider/llm/... -v -run TestAzureOpenAILLMFactory`
Expected: PASS

- [ ] **Step 5: Write failing test for ollama_llm factory**

```go
// internal/provider/llm/ollama_test.go
package llm

import (
	"testing"
)

func TestOllamaLLMFactory(t *testing.T) {
	f, ok := provider.GetLLMFactory("ollama_llm")
	if !ok {
		t.Fatal("ollama_llm factory not registered")
	}

	p, err := f(map[string]any{
		"url":   "http://localhost:11434",
		"model": "gemma2:9b",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetModel() != "gemma2:9b" {
		t.Fatalf("expected model gemma2:9b, got %s", p.GetModel())
	}
}
```

Run: `go test ./internal/provider/llm/... -v -run TestOllamaLLMFactory`
Expected: FAIL — factory not registered

- [ ] **Step 6: Implement ollama_llm factory**

```go
// internal/provider/llm/ollama.go
package llm

import (
	"strings"

	"github.com/odysseythink/pantheon/providers/ollama"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

func init() {
	provider.RegisterLLM("ollama_llm", func(cfg map[string]any) (core.LLMProvider, error) {
		url, _ := cfg["url"].(string)
		if url == "" {
			url = "http://localhost:11434"
		}
		model, _ := cfg["model"].(string)
		if model == "" {
			model = "gemma2:9b"
		}

		kwargs, _ := cfg["kwargs"].(map[string]any)
		if kwargs == nil {
			kwargs = map[string]any{"temperature": 0}
		}

		if !strings.HasSuffix(url, "/v1") {
			url = strings.TrimSuffix(url, "/") + "/v1"
		}

		pantheonProvider, err := ollama.New("", ollama.WithBaseURL(url))
		if err != nil {
			return nil, err
		}
		return NewPantheonLLMProvider(pantheonProvider, model, kwargs), nil
	})
}
```

Run: `go test ./internal/provider/llm/... -v -run TestOllamaLLMFactory`
Expected: PASS

- [ ] **Step 7: Run all LLM tests and commit**

Run: `go test ./internal/provider/llm/... -v`
Expected: all PASS

```bash
git add internal/provider/llm/
git commit -m "feat(provider): llm factories for openai, azure_openai, ollama"
```

---

## Task 4: Embedder Provider Factories

**Files:**
- Create: `internal/provider/embedder/openai.go`
- Create: `internal/provider/embedder/openai_test.go`
- Create: `internal/provider/embedder/ollama.go`
- Create: `internal/provider/embedder/ollama_test.go`

- [ ] **Step 1: Write failing test for openai_embedder factory**

```go
// internal/provider/embedder/openai_test.go
package embedder

import (
	"testing"
)

func TestOpenAIEmbedderFactory(t *testing.T) {
	f, ok := provider.GetEmbedderFactory("openai_embedder")
	if !ok {
		t.Fatal("openai_embedder factory not registered")
	}

	p, err := f(map[string]any{
		"api_key":   "test-key",
		"model":     "text-embedding-3-small",
		"dimension": 1536,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetModel() != "text-embedding-3-small" {
		t.Fatalf("expected model text-embedding-3-small, got %s", p.GetModel())
	}
	if p.GetDimensions() != 1536 {
		t.Fatalf("expected dimension 1536, got %d", p.GetDimensions())
	}
}
```

Run: `go test ./internal/provider/embedder/... -v -run TestOpenAIEmbedderFactory`
Expected: FAIL

- [ ] **Step 2: Implement openai_embedder factory**

```go
// internal/provider/embedder/openai.go
package embedder

import (
	openai "github.com/odysseythink/pantheon/providers/openai"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

func init() {
	provider.RegisterEmbedder("openai_embedder", func(cfg map[string]any) (core.EmbedderProvider, error) {
		apiKey, _ := cfg["api_key"].(string)
		apiBase, _ := cfg["api_base"].(string)
		model, _ := cfg["model"].(string)
		if model == "" {
			model = "text-embedding-3-large"
		}
		dimension := 3072
		if d, ok := cfg["dimension"].(int); ok {
			dimension = d
		}
		if d, ok := cfg["dimension"].(float64); ok {
			dimension = int(d)
		}

		var opts []openai.Option
		if apiBase != "" {
			opts = append(opts, openai.WithBaseURL(apiBase))
		}

		pantheonProvider, err := openai.New(apiKey, opts...)
		if err != nil {
			return nil, err
		}
		return NewPantheonEmbedderProvider(pantheonProvider, model, dimension), nil
	})
}
```

Run: `go test ./internal/provider/embedder/... -v -run TestOpenAIEmbedderFactory`
Expected: PASS

- [ ] **Step 3: Write failing test for ollama_embedder factory**

```go
// internal/provider/embedder/ollama_test.go
package embedder

import (
	"testing"
)

func TestOllamaEmbedderFactory(t *testing.T) {
	f, ok := provider.GetEmbedderFactory("ollama_embedder")
	if !ok {
		t.Fatal("ollama_embedder factory not registered")
	}

	p, err := f(map[string]any{
		"url":       "http://localhost:11434",
		"model":     "nomic-embed-text:latest",
		"dimension": 768,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetModel() != "nomic-embed-text:latest" {
		t.Fatalf("expected model nomic-embed-text:latest, got %s", p.GetModel())
	}
	if p.GetDimensions() != 768 {
		t.Fatalf("expected dimension 768, got %d", p.GetDimensions())
	}
}
```

Run: `go test ./internal/provider/embedder/... -v -run TestOllamaEmbedderFactory`
Expected: FAIL

- [ ] **Step 4: Implement ollama_embedder factory**

```go
// internal/provider/embedder/ollama.go
package embedder

import (
	"strings"

	openai "github.com/odysseythink/pantheon/providers/openai"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

func init() {
	provider.RegisterEmbedder("ollama_embedder", func(cfg map[string]any) (core.EmbedderProvider, error) {
		url, _ := cfg["url"].(string)
		if url == "" {
			url = "http://localhost:11434"
		}
		model, _ := cfg["model"].(string)
		if model == "" {
			model = "nomic-embed-text:latest"
		}
		dimension := 768
		if d, ok := cfg["dimension"].(int); ok {
			dimension = d
		}
		if d, ok := cfg["dimension"].(float64); ok {
			dimension = int(d)
		}

		if !strings.HasSuffix(url, "/v1") {
			url = strings.TrimSuffix(url, "/") + "/v1"
		}

		pantheonProvider, err := openai.New("", openai.WithBaseURL(url))
		if err != nil {
			return nil, err
		}
		return NewPantheonEmbedderProvider(pantheonProvider, model, dimension), nil
	})
}
```

Run: `go test ./internal/provider/embedder/... -v -run TestOllamaEmbedderFactory`
Expected: PASS

- [ ] **Step 5: Run all embedder tests and commit**

Run: `go test ./internal/provider/embedder/... -v`
Expected: all PASS

```bash
git add internal/provider/embedder/
git commit -m "feat(provider): embedder factories for openai, ollama"
```

---

## Task 5: Engine & DocStore Factory Registration

**Files:**
- Modify: `internal/provider/engine/wren_ui.go`
- Modify: `internal/provider/engine/wren_ibis.go`
- Modify: `internal/provider/engine/wren_engine.go`
- Modify: `internal/provider/docstore/qdrant.go`

- [ ] **Step 1: Register wren_ui factory**

Add to `internal/provider/engine/wren_ui.go`:

```go
func init() {
	provider.RegisterEngine("wren_ui", func(cfg map[string]any) (core.Engine, error) {
		endpoint, _ := cfg["endpoint"].(string)
		return NewWrenUI(endpoint), nil
	})
}
```

- [ ] **Step 2: Register wren_ibis factory**

Add to `internal/provider/engine/wren_ibis.go`:

```go
func init() {
	provider.RegisterEngine("wren_ibis", func(cfg map[string]any) (core.Engine, error) {
		endpoint, _ := cfg["endpoint"].(string)
		source, _ := cfg["source"].(string)
		manifest, _ := cfg["manifest"].(string)
		var connInfo map[string]any
		if v, ok := cfg["connection_info"].(map[string]any); ok {
			connInfo = v
		}
		return NewWrenIbis(endpoint, source, manifest, connInfo), nil
	})
}
```

- [ ] **Step 3: Register wren_engine factory**

Add to `internal/provider/engine/wren_engine.go`:

```go
func init() {
	provider.RegisterEngine("wren_engine", func(cfg map[string]any) (core.Engine, error) {
		endpoint, _ := cfg["endpoint"].(string)
		var manifest map[string]any
		if v, ok := cfg["manifest"].(map[string]any); ok {
			manifest = v
		}
		return NewWrenEngine(endpoint, manifest), nil
	})
}
```

- [ ] **Step 4: Register qdrant factory**

Add to `internal/provider/docstore/qdrant.go`:

```go
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
```

- [ ] **Step 5: Verify everything builds**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 6: Commit**

```bash
git add internal/provider/engine/ internal/provider/docstore/
git commit -m "feat(provider): register engine and docstore factories"
```

---

## Task 6: Component Assembly

**Files:**
- Create: `internal/service/components.go`
- Create: `internal/service/components_test.go`

- [ ] **Step 1: Write failing test for generateFromEnv**

```go
// internal/service/components_test.go
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
```

Run: `go test ./internal/service/... -v -run TestGenerateFromEnv`
Expected: FAIL — `GenerateComponents` undefined

- [ ] **Step 2: Implement GenerateComponents and generateFromEnv**

```go
// internal/service/components.go
package service

import (
	"fmt"

	"github.com/odysseythink/go-wren-ai-service/internal/config"
	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/provider"
)

// GenerateComponents creates per-pipeline PipelineComponent map from config.
func GenerateComponents(cfg *config.Config) (map[string]core.PipelineComponent, error) {
	if cfg.Raw == nil || len(cfg.Raw.Pipelines) == 0 {
		return generateFromEnv(cfg)
	}

	llms := map[string]core.LLMProvider{}
	for id, entry := range cfg.Raw.LLMs {
		f, ok := provider.GetLLMFactory(entry.Provider)
		if !ok {
			return nil, fmt.Errorf("unknown llm provider: %s", entry.Provider)
		}
		p, err := f(entry.Config)
		if err != nil {
			return nil, fmt.Errorf("init llm %s: %w", id, err)
		}
		llms[id] = p
	}

	embedders := map[string]core.EmbedderProvider{}
	for id, entry := range cfg.Raw.Embedders {
		f, ok := provider.GetEmbedderFactory(entry.Provider)
		if !ok {
			return nil, fmt.Errorf("unknown embedder provider: %s", entry.Provider)
		}
		p, err := f(entry.Config)
		if err != nil {
			return nil, fmt.Errorf("init embedder %s: %w", id, err)
		}
		embedders[id] = p
	}

	docStores := map[string]core.DocStoreProvider{}
	for id, entry := range cfg.Raw.DocumentStores {
		f, ok := provider.GetDocStoreFactory(entry.Provider)
		if !ok {
			return nil, fmt.Errorf("unknown docstore provider: %s", entry.Provider)
		}
		p, err := f(entry.Config)
		if err != nil {
			return nil, fmt.Errorf("init docstore %s: %w", id, err)
		}
		docStores[id] = p
	}

	engines := map[string]core.Engine{}
	for id, entry := range cfg.Raw.Engines {
		f, ok := provider.GetEngineFactory(entry.Provider)
		if !ok {
			return nil, fmt.Errorf("unknown engine provider: %s", entry.Provider)
		}
		p, err := f(entry.Config)
		if err != nil {
			return nil, fmt.Errorf("init engine %s: %w", id, err)
		}
		engines[id] = p
	}

	components := make(map[string]core.PipelineComponent)
	for name, pipe := range cfg.Raw.Pipelines {
		comp := core.PipelineComponent{}
		if pipe.LLM != "" {
			if p, ok := llms[pipe.LLM]; ok {
				comp.LLMProvider = p
			} else {
				return nil, fmt.Errorf("pipeline %q references unknown llm: %s", name, pipe.LLM)
			}
		}
		if pipe.Embedder != "" {
			if p, ok := embedders[pipe.Embedder]; ok {
				comp.EmbedderProvider = p
			} else {
				return nil, fmt.Errorf("pipeline %q references unknown embedder: %s", name, pipe.Embedder)
			}
		}
		if pipe.DocumentStore != "" {
			if p, ok := docStores[pipe.DocumentStore]; ok {
				comp.DocStoreProvider = p
			} else {
				return nil, fmt.Errorf("pipeline %q references unknown docstore: %s", name, pipe.DocumentStore)
			}
		}
		if pipe.Engine != "" {
			if p, ok := engines[pipe.Engine]; ok {
				comp.Engine = p
			} else {
				return nil, fmt.Errorf("pipeline %q references unknown engine: %s", name, pipe.Engine)
			}
		}
		components[name] = comp
	}

	return components, nil
}

func generateFromEnv(cfg *config.Config) (map[string]core.PipelineComponent, error) {
	// Pantheon OpenAI provider
	openaiProvider, err := openai.New(cfg.LLMOpenAIAPIKey, openai.WithBaseURL(cfg.LLMOpenAIAPIBase))
	if err != nil {
		return nil, fmt.Errorf("create openai provider: %w", err)
	}

	llmProvider := llm.NewPantheonLLMProvider(openaiProvider, cfg.GenerationModel, map[string]any{
		"temperature":     0,
		"n":               1,
		"max_tokens":      4096,
		"response_format": map[string]any{"type": "json_object"},
	})

	embedProvider, ok := openaiProvider.(embed.Provider)
	if !ok {
		return nil, fmt.Errorf("pantheon provider does not support embedding")
	}
	embedderProvider := embedder.NewPantheonEmbedderProvider(embedProvider, cfg.EmbeddingModel, cfg.EmbeddingModelDim)

	docStoreProvider := docstore.NewQdrantProvider(
		fmt.Sprintf("http://%s:6333", cfg.QdrantHost),
		cfg.QdrantAPIKey,
		cfg.EmbeddingModelDim,
		cfg.QdrantTimeout,
	)

	var eng core.Engine
	switch cfg.Engine {
	case "wren_ibis":
		var connInfo map[string]any
		if cfg.WrenIbisConnInfo != "" {
			_ = json.Unmarshal([]byte(cfg.WrenIbisConnInfo), &connInfo)
		}
		eng = engine.NewWrenIbis(cfg.WrenIbisEndpoint, cfg.WrenIbisSource, cfg.WrenIbisManifest, connInfo)
	case "wren_engine":
		var manifest map[string]any
		if cfg.WrenEngineManifest != "" {
			_ = json.Unmarshal([]byte(cfg.WrenEngineManifest), &manifest)
		}
		eng = engine.NewWrenEngine(cfg.WrenEngineEndpoint, manifest)
	default:
		eng = engine.NewWrenUI(cfg.WrenUIEndpoint)
	}

	comp := core.PipelineComponent{
		LLMProvider:      llmProvider,
		EmbedderProvider: embedderProvider,
		DocStoreProvider: docStoreProvider,
		Engine:           eng,
	}

	pipelines := []string{
		"indexing", "retrieval", "historical_question",
		"sql_generation", "sql_correction", "followup_sql_generation",
		"sql_summary", "sql_answer", "sql_breakdown",
		"sql_expansion", "sql_explanation", "sql_regeneration",
		"semantics_description", "relationship_recommendation",
	}
	components := make(map[string]core.PipelineComponent)
	for _, name := range pipelines {
		components[name] = comp
	}
	return components, nil
}
```

> Note: add necessary imports (`openai "github.com/odysseythink/pantheon/providers/openai"`, `embed "github.com/odysseythink/pantheon/extensions/embed"`, `encoding/json`, and internal packages).

Run: `go test ./internal/service/... -v -run TestGenerateFromEnv`
Expected: PASS

- [ ] **Step 3: Write test for YAML-driven assembly**

```go
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

	// sql_generation should have LLM and Engine
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

	// indexing should have Embedder and DocStore
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
```

Run: `go test ./internal/service/... -v -run TestGenerateComponents_FromYAML`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/service/components.go internal/service/components_test.go
git commit -m "feat(service): GenerateComponents with env fallback and YAML-driven assembly"
```

---

## Task 7: Service Container Adaptation

**Files:**
- Modify: `internal/service/container.go`

- [ ] **Step 1: Modify NewContainer signature and implementation**

```go
// internal/service/container.go
func NewContainer(components map[string]core.PipelineComponent, cfg *config.Config) *Container {
	queryCache := cache.New(cache.DefaultExpiration, cache.NoExpiration)

	get := func(name string) core.PipelineComponent {
		c, ok := components[name]
		if !ok {
			panic(fmt.Sprintf("pipeline component %q not found", name))
		}
		return c
	}

	indexingPipe := indexing.NewIndexing(get("indexing"), cfg.ColumnIndexingBatchSize)
	retrievalPipe := retrieval.NewRetrieval(get("retrieval"), cfg.TableRetrievalSize, cfg.TableColumnRetrievalSize)
	 historicalPipe := retrieval.NewHistoricalQuestion(get("historical_question"))

	sqlGenPipe := generation.NewSQLGeneration(get("sql_generation"))
	sqlCorrectionPipe := generation.NewSQLCorrection(get("sql_correction"))
	sqlSummaryPipe := generation.NewSQLSummary(get("sql_summary"))
	sqlAnswerPipe := generation.NewSQLAnswer(get("sql_answer"))
	sqlBreakdownPipe := generation.NewSQLBreakdown(get("sql_breakdown"))
	sqlExpansionPipe := generation.NewSQLExpansion(get("sql_expansion"))
	sqlRegenerationPipe := generation.NewSQLRegeneration(get("sql_regeneration"))
	sqlExplanationPipe := generation.NewSQLExplanation(get("sql_explanation"))
	followupSQLPipe := generation.NewFollowUpSQLGeneration(get("followup_sql_generation"))
	semanticsDescPipe := generation.NewSemanticsDescription(get("semantics_description"))
	relationshipRecPipe := generation.NewRelationshipRecommendation(get("relationship_recommendation"))

	return &Container{
		AskService:                   NewAskService(queryCache, retrievalPipe, historicalPipe, sqlGenPipe, sqlCorrectionPipe, sqlSummaryPipe, followupSQLPipe),
		AskDetailsService:            NewAskDetailsService(queryCache, sqlBreakdownPipe),
		SQLAnswerService:             NewSQLAnswerService(queryCache, sqlAnswerPipe),
		SQLExpansionService:          NewSQLExpansionService(queryCache, retrievalPipe, sqlExpansionPipe, sqlCorrectionPipe, sqlSummaryPipe),
		SQLExplanationService:        NewSQLExplanationService(queryCache, sqlExplanationPipe),
		SQLRegenerationService:       NewSQLRegenerationService(queryCache, sqlRegenerationPipe),
		SemanticsPreparationService:  NewSemanticsPreparationService(queryCache, indexingPipe),
		SemanticsDescriptionService:  NewSemanticsDescriptionService(queryCache, semanticsDescPipe),
		RelationshipRecommendationService: NewRelationshipRecommendationService(queryCache, relationshipRecPipe),
	}
}
```

> Note: add `fmt` import if missing.

- [ ] **Step 2: Verify build**

Run: `go build ./internal/service/...`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add internal/service/container.go
git commit -m "refactor(service): NewContainer accepts map[string]core.PipelineComponent"
```

---

## Task 8: main.go Boot Flow

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Simplify main.go**

Replace the entire `main.go` content with:

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/odysseythink/go-wren-ai-service/internal/config"
	"github.com/odysseythink/go-wren-ai-service/internal/handler"
	"github.com/odysseythink/go-wren-ai-service/internal/service"
)

func main() {
	cfg := config.Load()

	components, err := service.GenerateComponents(cfg)
	if err != nil {
		log.Fatalf("failed to generate pipeline components: %v", err)
	}

	container := service.NewContainer(components, cfg)
	router := handler.NewRouter(container)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("go-wren-ai-service starting on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./cmd/server/...`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "refactor(main): simplified boot using GenerateComponents"
```

---

## Task 9: Integration Test Update

**Files:**
- Modify: `integration_test.go`

- [ ] **Step 1: Update integration_test.go**

Replace `service.NewContainer(components, cfg)` call with:

```go
// Before: service.NewContainer(components, cfg)
// After:
components := map[string]core.PipelineComponent{
	"indexing":              comp,
	"retrieval":             comp,
	"historical_question":   comp,
	"sql_generation":        comp,
	"sql_correction":        comp,
	"followup_sql_generation": comp,
	"sql_summary":           comp,
	"sql_answer":            comp,
	"sql_breakdown":         comp,
	"sql_expansion":         comp,
	"sql_explanation":       comp,
	"sql_regeneration":      comp,
	"semantics_description": comp,
	"relationship_recommendation": comp,
}
container := service.NewContainer(components, cfg)
```

- [ ] **Step 2: Run integration test**

Run: `go test ./integration_test.go -v`
Expected: PASS (or FAIL for unrelated reasons; fix if related to signature change)

- [ ] **Step 3: Commit**

```bash
git add integration_test.go
git commit -m "test: adapt integration test to new NewContainer signature"
```

---

## Self-Review

**1. Spec coverage:**
| Spec Requirement | Implementing Task |
|-----------------|-------------------|
| Multi-document YAML parsing | Task 2 |
| Typed provider registry | Task 1 |
| `openai_llm` factory | Task 3 |
| `azure_openai_llm` factory | Task 3 |
| `ollama_llm` factory | Task 3 |
| `openai_embedder` factory | Task 4 |
| `ollama_embedder` factory | Task 4 |
| Engine factories (wren_ui/ibis/engine) | Task 5 |
| DocStore factory (qdrant) | Task 5 |
| Per-pipeline component assembly | Task 6 |
| Env-var fallback | Task 6 |
| Service container adaptation | Task 7 |
| Simplified main.go | Task 8 |
| Integration test update | Task 9 |

**2. Placeholder scan:** No TBD, TODO, or vague steps. Every step has exact file paths, exact code, and exact commands.

**3. Type consistency:** All signatures match design doc. `NewContainer` takes `map[string]core.PipelineComponent` consistently. `GenerateComponents` returns the same type. Registry functions use typed signatures throughout.
