package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"novel-agent-runtime/internal/project"
	"novel-agent-runtime/internal/runtime"
	"novel-agent-runtime/internal/store"
	"novel-agent-runtime/internal/workflow"
)

func ensureWorkflowSkillsExist(ctx context.Context, rt *runtime.Runtime, steps []workflow.WorkflowStep) error {
	if rt == nil || rt.Registry == nil {
		return fmt.Errorf("runtime registry is unavailable")
	}
	for _, step := range steps {
		if _, err := rt.Registry.LoadInvocationCommand(step.SkillID); err != nil {
			return fmt.Errorf("workflow step %q requires skill %q: %w", step.ID, step.SkillID, err)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return nil
}

func loadProjectContextPack(ctx context.Context, db workflow.ProjectStore, projectID string, request string) (workflow.ContextPack, *store.Project, error) {
	provider := workflow.PostgresContextProvider{Store: db}
	pack, err := provider.BuildContext(ctx, projectID, request)
	if err != nil {
		return workflow.ContextPack{}, nil, err
	}
	if strings.TrimSpace(pack.Project.ID) == "" {
		return pack, nil, nil
	}
	return pack, &store.Project{
		ID:              pack.Project.ID,
		Name:            pack.Project.Name,
		Description:     pack.Project.Description,
		Status:          pack.Project.Status,
		StorageProvider: pack.Project.StorageProvider,
		StorageBucket:   pack.Project.StorageBucket,
		StoragePrefix:   pack.Project.StoragePrefix,
	}, nil
}

func persistWorkflowDocuments(ctx context.Context, projects projectConfigStore, projectID string, plan fixedWorkflowPlan, out workflow.WorkflowOutput) (workflowPersistenceResult, error) {
	finalText := strings.TrimSpace(workflowFinalText(out))
	if strings.TrimSpace(projectID) == "" {
		return inferWorkflowPersistenceResult(plan, finalText, nil), nil
	}

	switch plan.PersistMode {
	case persistModeSingleDocument:
		docs, err := persistSingleWorkflowDocument(ctx, projects, projectID, plan, out)
		if err != nil {
			return workflowPersistenceResult{}, err
		}
		return inferWorkflowPersistenceResult(plan, finalText, docs), nil
	case "", persistModeExtractSections:
		docs, err := persistExtractedWorkflowDocuments(ctx, projects, projectID, out)
		if err != nil {
			return workflowPersistenceResult{}, err
		}
		return inferWorkflowPersistenceResult(plan, finalText, docs), nil
	default:
		return workflowPersistenceResult{}, fmt.Errorf("unsupported workflow persist mode %q", plan.PersistMode)
	}
}

func persistExtractedWorkflowDocuments(ctx context.Context, projects projectConfigStore, projectID string, out workflow.WorkflowOutput) ([]workflowDocumentUpdate, error) {
	seen := map[string]workflowDocumentUpdate{}
	for _, step := range out.Steps {
		drafts := project.ExtractDocumentDrafts(step.Output.Text)
		for _, draft := range drafts {
			doc, err := upsertWorkflowDocument(ctx, projects, projectID, out.WorkflowID, step, draft.Kind, firstNonEmpty(draft.Title, project.DefaultDocumentTitle(draft.Kind)), draft.Body)
			if err != nil {
				return nil, err
			}
			seen[doc.Kind] = workflowDocumentUpdate{
				Kind:      doc.Kind,
				Title:     doc.Title,
				BodyBytes: len(doc.Body),
			}
		}
	}
	return sortWorkflowDocumentUpdates(seen), nil
}

func persistSingleWorkflowDocument(ctx context.Context, projects projectConfigStore, projectID string, plan fixedWorkflowPlan, out workflow.WorkflowOutput) ([]workflowDocumentUpdate, error) {
	finalText := strings.TrimSpace(workflowFinalText(out))
	if finalText == "" || !shouldPersistSingleWorkflowDocument(plan, finalText) {
		return nil, nil
	}
	step := workflow.WorkflowStepOutput{}
	if len(out.Steps) > 0 {
		step = out.Steps[len(out.Steps)-1]
	}
	doc, err := upsertWorkflowDocument(ctx, projects, projectID, out.WorkflowID, step, plan.PersistKind, firstNonEmpty(plan.PersistTitle, project.DefaultDocumentTitle(plan.PersistKind)), finalText)
	if err != nil {
		return nil, err
	}
	return []workflowDocumentUpdate{{
		Kind:      doc.Kind,
		Title:     doc.Title,
		BodyBytes: len(doc.Body),
	}}, nil
}

func inferWorkflowPersistenceResult(plan fixedWorkflowPlan, finalText string, docs []workflowDocumentUpdate) workflowPersistenceResult {
	result := workflowPersistenceResult{UpdatedDocuments: docs}
	finalText = strings.TrimSpace(finalText)

	switch {
	case len(docs) > 0:
		result.ResponseMode = workflowResponseModeDocument
	case looksLikeWorkflowClarification(plan, finalText):
		result.ResponseMode = workflowResponseModeClarification
		result.NeedsInput = true
	case plan.PersistMode == persistModeSingleDocument && finalText != "":
		if looksLikeWorkflowDocument(plan, finalText) {
			result.ResponseMode = workflowResponseModeDocument
		} else {
			// Single-document workflows are conservative: if the output does not
			// match the expected persisted heading, do not treat it as canon.
			result.ResponseMode = workflowResponseModeClarification
			result.NeedsInput = true
		}
	}

	return result
}

func shouldPersistSingleWorkflowDocument(plan fixedWorkflowPlan, finalText string) bool {
	finalText = strings.TrimSpace(finalText)
	if finalText == "" {
		return false
	}
	if looksLikeWorkflowClarification(plan, finalText) {
		return false
	}
	return looksLikeWorkflowDocument(plan, finalText)
}

func looksLikeWorkflowDocument(plan fixedWorkflowPlan, text string) bool {
	heading := normalizeWorkflowHeading(firstMarkdownHeading(text))
	if heading == "" {
		return false
	}

	candidates := []string{
		plan.PersistHeading,
		plan.PersistTitle,
		project.DefaultDocumentTitle(plan.PersistKind),
		plan.PersistKind,
	}
	for _, candidate := range candidates {
		if normalizeWorkflowHeading(candidate) == heading && heading != "" {
			return true
		}
	}
	return false
}

func looksLikeWorkflowClarification(plan fixedWorkflowPlan, text string) bool {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	if text == "" {
		return false
	}
	heading := normalizeWorkflowHeading(firstMarkdownHeading(text))
	if heading != "" && heading == normalizeWorkflowHeading(plan.ClarifyHeading) {
		return true
	}
	for _, marker := range []string{
		"请先回答以下问题",
		"还不能定稿的原因",
		"当前已确认",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func firstMarkdownHeading(text string) string {
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			return strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		}
	}
	return ""
}

func normalizeWorkflowHeading(s string) string {
	s = strings.TrimSpace(strings.Trim(s, "`"))
	return strings.ToLower(s)
}

func upsertWorkflowDocument(ctx context.Context, projects projectConfigStore, projectID, workflowID string, step workflow.WorkflowStepOutput, kind, title, body string) (store.ProjectDocument, error) {
	metadata, err := json.Marshal(map[string]any{
		"workflow_id":   workflowID,
		"workflow_step": step.StepID,
		"skill_id":      step.Output.SkillID,
		"source":        "workflow_postprocess",
	})
	if err != nil {
		return store.ProjectDocument{}, err
	}
	return projects.UpsertProjectDocument(ctx, store.UpsertProjectDocumentParams{
		ProjectID: projectID,
		Kind:      kind,
		Title:     title,
		Body:      body,
		Metadata:  metadata,
	})
}

func sortWorkflowDocumentUpdates(seen map[string]workflowDocumentUpdate) []workflowDocumentUpdate {
	keys := make([]string, 0, len(seen))
	for kind := range seen {
		keys = append(keys, kind)
	}
	sort.Strings(keys)

	outDocs := make([]workflowDocumentUpdate, 0, len(keys))
	for _, kind := range keys {
		outDocs = append(outDocs, seen[kind])
	}
	return outDocs
}

func workflowFinalText(out workflow.WorkflowOutput) string {
	for i := len(out.Steps) - 1; i >= 0; i-- {
		text := strings.TrimSpace(out.Steps[i].Output.Text)
		if text != "" {
			return text
		}
	}
	return ""
}
