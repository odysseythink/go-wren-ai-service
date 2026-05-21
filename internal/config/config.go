package config

import (
	"bytes"
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
	ColumnIndexingBatchSize  int `env:"COLUMN_INDEXING_BATCH_SIZE" envDefault:"50"`
	TableRetrievalSize       int `env:"TABLE_RETRIEVAL_SIZE" envDefault:"10"`
	TableColumnRetrievalSize int `env:"TABLE_COLUMN_RETRIEVAL_SIZE" envDefault:"1000"`
	QueryCacheTTL            int `env:"QUERY_CACHE_TTL" envDefault:"120"`

	// Parsed YAML config (nil when config.yaml absent)
	Raw *RawConfig
}

// RawConfig holds parsed multi-document YAML config.
type RawConfig struct {
	LLMs           map[string]ProviderEntry
	Embedders      map[string]ProviderEntry
	DocumentStores map[string]ProviderEntry
	Engines        map[string]ProviderEntry
	Pipelines      map[string]PipelineEntry
}

// ProviderEntry holds a single provider instance config.
type ProviderEntry struct {
	Provider string
	Config   map[string]any
}

// PipelineEntry holds per-pipeline provider references.
type PipelineEntry struct {
	LLM           string
	Embedder      string
	DocumentStore string
	Engine        string
}

// Load reads config from env vars and optional YAML file.
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

// ParseYAML reads multi-document YAML and converts it to RawConfig.
func ParseYAML(data []byte) (*RawConfig, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var docs []map[string]any
	for {
		var doc map[string]any
		if err := dec.Decode(&doc); err != nil {
			break
		}
		if doc != nil {
			docs = append(docs, doc)
		}
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
			"provider":  provider,
			"model":     modelName,
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
