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
