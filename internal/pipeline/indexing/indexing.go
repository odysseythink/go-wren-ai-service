package indexing

import (
	"context"
	"fmt"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/pkg/mdl"
	"golang.org/x/sync/errgroup"
)

// Indexing implements core.Pipeline for MDL indexing.
type Indexing struct {
	components       core.PipelineComponent
	batchSize        int
	tableDescStore   core.DocumentStore
	ddlStore         core.DocumentStore
	viewStore        core.DocumentStore
}

// NewIndexing creates a new indexing pipeline.
func NewIndexing(components core.PipelineComponent, batchSize int) *Indexing {
	return &Indexing{
		components:     components,
		batchSize:      batchSize,
		tableDescStore: components.DocStoreProvider.GetStore(core.StoreOpts{DatasetName: "table_descriptions"}),
		ddlStore:       components.DocStoreProvider.GetStore(core.StoreOpts{DatasetName: "Document"}),
		viewStore:      components.DocStoreProvider.GetStore(core.StoreOpts{DatasetName: "view_questions"}),
	}
}

// Run executes the indexing pipeline.
func (p *Indexing) Run(ctx context.Context, input any) (any, error) {
	req, ok := input.(*IndexingRequest)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	// Parse and validate MDL
	mdlObj, err := mdl.ParseMDL(req.MDL)
	if err != nil {
		return nil, fmt.Errorf("parse MDL: %w", err)
	}

	// Clean old documents
	filters := map[string]any{}
	if req.ProjectID != "" {
		filters["project_id"] = req.ProjectID
	}
	_ = p.tableDescStore.DeleteDocuments(ctx, filters)
	_ = p.ddlStore.DeleteDocuments(ctx, filters)
	_ = p.viewStore.DeleteDocuments(ctx, filters)

	// Concurrent indexing branches
	g, ctx := errgroup.WithContext(ctx)

	// Branch 1: Table descriptions
	g.Go(func() error {
		descs := mdl.ConvertToTableDescriptions(mdlObj)
		var docs []core.Document
		for _, d := range descs {
			meta := map[string]any{"type": "TABLE_DESCRIPTION", "mdl_type": d.MDLType}
			if req.ProjectID != "" {
				meta["project_id"] = req.ProjectID
			}
			docs = append(docs, core.Document{
				ID:      fmt.Sprintf("%s-%s", d.MDLType, d.Name),
				Content: d.Description,
				Meta:    meta,
			})
		}
		if len(docs) == 0 {
			return nil
		}
		embedder, _ := p.components.EmbedderProvider.GetDocumentEmbedder(ctx)
		result, _ := embedder.Run(ctx, docs)
		_, err := p.tableDescStore.WriteDocuments(ctx, result.Documents, core.WritePolicyOverwrite)
		return err
	})

	// Branch 2: DDL schemas
	g.Go(func() error {
		commands := mdl.ConvertToDDL(mdlObj, p.batchSize)
		var docs []core.Document
		for _, cmd := range commands {
			meta := map[string]any{"type": cmd.Name, "name": cmd.Name}
			if req.ProjectID != "" {
				meta["project_id"] = req.ProjectID
			}
			docs = append(docs, core.Document{
				ID:      fmt.Sprintf("ddl-%s-%s", cmd.Name, cmd.Payload),
				Content: cmd.Payload,
				Meta:    meta,
			})
		}
		if len(docs) == 0 {
			return nil
		}
		embedder, _ := p.components.EmbedderProvider.GetDocumentEmbedder(ctx)
		result, _ := embedder.Run(ctx, docs)
		_, err := p.ddlStore.WriteDocuments(ctx, result.Documents, core.WritePolicyOverwrite)
		return err
	})

	// Branch 3: Views
	g.Go(func() error {
		views := mdl.ConvertViews(mdlObj)
		var docs []core.Document
		for i, v := range views {
			meta := map[string]any{"summary": v.Meta["summary"], "statement": v.Meta["statement"], "viewId": v.Meta["viewId"]}
			if req.ProjectID != "" {
				meta["project_id"] = req.ProjectID
			}
			docs = append(docs, core.Document{
				ID:      fmt.Sprintf("view-%d", i),
				Content: v.Content,
				Meta:    meta,
			})
		}
		if len(docs) == 0 {
			return nil
		}
		embedder, _ := p.components.EmbedderProvider.GetDocumentEmbedder(ctx)
		result, _ := embedder.Run(ctx, docs)
		_, err := p.viewStore.WriteDocuments(ctx, result.Documents, core.WritePolicyOverwrite)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return &IndexingResult{Success: true}, nil
}

// IndexingRequest is the input to the indexing pipeline.
type IndexingRequest struct {
	MDL       string
	ProjectID string
}

// IndexingResult is the output of the indexing pipeline.
type IndexingResult struct {
	Success bool
}
