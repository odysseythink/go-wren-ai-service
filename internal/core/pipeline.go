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
