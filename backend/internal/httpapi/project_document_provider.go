package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"novel-agent-runtime/internal/runtime"
	"novel-agent-runtime/internal/store"
)

type runtimeProjectDocumentProvider struct {
	Projects projectConfigStore
}

func (p runtimeProjectDocumentProvider) ListProjectDocuments(ctx context.Context, projectID string) ([]runtime.ProjectDocumentSummary, error) {
	docs, err := p.Projects.ListProjectDocuments(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]runtime.ProjectDocumentSummary, 0, len(docs))
	for _, doc := range docs {
		out = append(out, runtime.ProjectDocumentSummary{
			ProjectID: doc.ProjectID,
			Kind:      doc.Kind,
			Title:     doc.Title,
			BodyBytes: len(doc.Body),
		})
	}
	return out, nil
}

func (p runtimeProjectDocumentProvider) ReadProjectDocument(ctx context.Context, projectID, kind string) (runtime.ProjectDocumentReadResult, error) {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return runtime.ProjectDocumentReadResult{}, fmt.Errorf("project document kind is required")
	}
	docs, err := p.Projects.ListProjectDocuments(ctx, projectID)
	if err != nil {
		return runtime.ProjectDocumentReadResult{}, err
	}
	for _, doc := range docs {
		if doc.Kind != kind {
			continue
		}
		return runtime.ProjectDocumentReadResult{
			ProjectID: doc.ProjectID,
			Kind:      doc.Kind,
			Title:     doc.Title,
			Body:      doc.Body,
			Metadata:  metadataMap(doc.Metadata),
		}, nil
	}
	return runtime.ProjectDocumentReadResult{}, fmt.Errorf("project document %q not found", kind)
}

func (p runtimeProjectDocumentProvider) WriteProjectDocument(ctx context.Context, req runtime.ProjectDocumentWriteRequest) (runtime.ProjectDocumentWriteResult, error) {
	metadata := json.RawMessage(`{}`)
	if len(req.Metadata) > 0 {
		body, err := json.Marshal(req.Metadata)
		if err != nil {
			return runtime.ProjectDocumentWriteResult{}, err
		}
		metadata = body
	}
	doc, err := p.Projects.UpsertProjectDocument(ctx, store.UpsertProjectDocumentParams{
		ProjectID: req.ProjectID,
		Kind:      req.Kind,
		Title:     firstNonEmpty(req.Title, req.Kind),
		Body:      req.Body,
		Metadata:  metadata,
	})
	if err != nil {
		return runtime.ProjectDocumentWriteResult{}, err
	}
	return runtime.ProjectDocumentWriteResult{
		ProjectID: doc.ProjectID,
		Kind:      doc.Kind,
		Title:     doc.Title,
		BodyBytes: len(doc.Body),
		Synced:    true,
	}, nil
}

func metadataMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
