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
	ColumnIndexingBatchSize  int `env:"COLUMN_INDEXING_BATCH_SIZE" envDefault:"50"`
	TableRetrievalSize       int `env:"TABLE_RETRIEVAL_SIZE" envDefault:"10"`
	TableColumnRetrievalSize int `env:"TABLE_COLUMN_RETRIEVAL_SIZE" envDefault:"1000"`
	QueryCacheTTL            int `env:"QUERY_CACHE_TTL" envDefault:"120"`
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
