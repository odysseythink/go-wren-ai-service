package retrieval

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/odysseythink/go-wren-ai-service/internal/core"
	"github.com/odysseythink/go-wren-ai-service/internal/pipeline/common"
	"github.com/odysseythink/go-wren-ai-service/pkg/sqlutil"
)

// Retrieval implements core.Pipeline for query retrieval.
type Retrieval struct {
	components         core.PipelineComponent
	tableRetriever     core.Retriever
	schemaRetriever    core.Retriever
	tableDescStore     core.DocumentStore
	schemaStore        core.DocumentStore
	tableRetrievalSize int
	columnRetrievalSize int
}

// NewRetrieval creates a new retrieval pipeline.
func NewRetrieval(components core.PipelineComponent, tableSize, columnSize int) *Retrieval {
	return &Retrieval{
		components:         components,
		tableDescStore:     components.DocStoreProvider.GetStore(core.StoreOpts{DatasetName: "table_descriptions"}),
		schemaStore:        components.DocStoreProvider.GetStore(core.StoreOpts{DatasetName: "Document"}),
		tableRetriever:     components.DocStoreProvider.GetRetriever(components.DocStoreProvider.GetStore(core.StoreOpts{DatasetName: "table_descriptions"}), tableSize),
		schemaRetriever:    components.DocStoreProvider.GetRetriever(components.DocStoreProvider.GetStore(core.StoreOpts{DatasetName: "Document"}), columnSize),
		tableRetrievalSize: tableSize,
		columnRetrievalSize: columnSize,
	}
}

// RetrievalResult holds the output of retrieval.
type RetrievalResult struct {
	Documents []string // DDL strings
}

// Run executes the retrieval pipeline.
func (p *Retrieval) Run(ctx context.Context, input any) (any, error) {
	req, ok := input.(*RetrievalRequest)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	// Embed query
	textEmbedder, err := p.components.EmbedderProvider.GetTextEmbedder(ctx)
	if err != nil {
		return nil, err
	}
	embedResult, err := textEmbedder.Run(ctx, req.Query)
	if err != nil {
		return nil, err
	}

	filters := map[string]any{}
	if req.ProjectID != "" {
		filters["project_id"] = req.ProjectID
	}

	// Retrieve table descriptions
	tableResult, err := p.tableRetriever.Run(ctx, embedResult.Embedding, filters)
	if err != nil {
		return nil, err
	}

	// Get table names from retrieved descriptions
	tableNames := map[string]bool{}
	for _, doc := range tableResult.Documents {
		if name, ok := doc.Meta["name"].(string); ok && name != "" {
			tableNames[name] = true
		}
	}

	// Retrieve schema documents for those tables
	schemaResult, err := p.schemaRetriever.Run(ctx, embedResult.Embedding, filters)
	if err != nil {
		return nil, err
	}

	// Build DDL strings
	var ddls []string
	for _, doc := range schemaResult.Documents {
		if doc.Content != "" {
			ddls = append(ddls, doc.Content)
		}
	}

	// Column selection via LLM (simplified: return all DDLs)
	return &RetrievalResult{Documents: ddls}, nil
}

// RetrievalRequest is the input to retrieval.
type RetrievalRequest struct {
	Query     string
	ProjectID string
}

// buildColumnSelectionPrompt builds the prompt for LLM column selection.
func buildColumnSelectionPrompt(dbSchemas []string, question string) (string, error) {
	builder, err := common.NewPromptBuilder(tableColumnsSelectionUserPrompt)
	if err != nil {
		return "", err
	}
	return builder.Build(map[string]any{
		"db_schemas": dbSchemas,
		"question":   question,
	})
}

const tableColumnsSelectionSystemPrompt = `You are a highly skilled data analyst. Your goal is to examine the provided database schema, interpret the posed question, and identify the specific columns from the relevant tables required to construct an accurate SQL query.`

const tableColumnsSelectionUserPrompt = `### Database Schema ###
{{range .db_schemas}}
    {{.}}
{{end}}

### INPUT ###
{{.question}}
`

// ColumnSelectionResult holds the LLM column selection output.
type ColumnSelectionResult struct {
	Results []struct {
		TableName            string   `json:"table_name"`
		TableSelectionReason string   `json:"table_selection_reason"`
		TableContents        struct {
			ChainOfThoughtReasoning []string `json:"chain_of_thought_reasoning"`
			Columns                 []string `json:"columns"`
		} `json:"table_contents"`
	} `json:"results"`
}

// selectColumns runs the LLM to select relevant columns.
func (p *Retrieval) selectColumns(ctx context.Context, dbSchemas []string, question string) (*ColumnSelectionResult, error) {
	prompt, err := buildColumnSelectionPrompt(dbSchemas, question)
	if err != nil {
		return nil, err
	}

	gen, err := p.components.LLMProvider.GetGenerator(ctx, core.GeneratorOpts{
		SystemPrompt:     tableColumnsSelectionSystemPrompt,
		GenerationKwargs: map[string]any{"response_format": map[string]any{"type": "json_object"}},
	})
	if err != nil {
		return nil, err
	}

	result, err := gen.Run(ctx, prompt)
	if err != nil {
		return nil, err
	}
	if len(result.Replies) == 0 {
		return nil, fmt.Errorf("no reply")
	}

	cleaned := sqlutil.CleanGenerationResult(result.Replies[0])
	var selection ColumnSelectionResult
	if err := json.Unmarshal([]byte(cleaned), &selection); err != nil {
		return nil, err
	}
	return &selection, nil
}
