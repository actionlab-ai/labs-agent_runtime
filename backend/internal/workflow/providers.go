package workflow

import (
	"context"
	"fmt"
	"strings"

	"novel-agent-runtime/internal/project"
	"novel-agent-runtime/internal/runtime"
	"novel-agent-runtime/internal/skill"
	"novel-agent-runtime/internal/store"
)

type LocalSkillProvider struct {
	SkillsDir string
}

func (p LocalSkillProvider) ListSkills(ctx context.Context) ([]SkillSpec, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	reg, err := skill.LoadRegistry(p.SkillsDir)
	if err != nil {
		return nil, err
	}
	cmds := reg.List()
	out := make([]SkillSpec, 0, len(cmds))
	for _, cmd := range cmds {
		out = append(out, skillSpecFromCommand(cmd))
	}
	return out, nil
}

func (p LocalSkillProvider) LoadSkill(ctx context.Context, id string) (SkillDefinition, error) {
	if err := ctx.Err(); err != nil {
		return SkillDefinition{}, err
	}
	reg, err := skill.LoadRegistry(p.SkillsDir)
	if err != nil {
		return SkillDefinition{}, err
	}
	cmd, err := reg.LoadInvocationCommand(id)
	if err != nil {
		return SkillDefinition{}, err
	}
	return SkillDefinition{
		Spec:            skillSpecFromCommand(cmd),
		MarkdownContent: cmd.MarkdownContent,
		EntryPath:       cmd.EntryPath,
		SkillRoot:       cmd.SkillRoot,
	}, nil
}

type ProjectStore interface {
	GetProject(context.Context, string) (store.Project, error)
	ListProjectDocuments(context.Context, string) ([]store.ProjectDocument, error)
}

type projectPolicyStore interface {
	GetAppSetting(context.Context, string) (store.AppSetting, error)
}

type PostgresContextProvider struct {
	Store ProjectStore
}

func (p PostgresContextProvider) BuildContext(ctx context.Context, projectID string, request string) (ContextPack, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return ContextPack{Request: strings.TrimSpace(request)}, nil
	}
	if p.Store == nil {
		return ContextPack{}, fmt.Errorf("project store is required")
	}
	policy := loadProjectDocumentPolicy(ctx, p.Store)
	activeProject, err := p.Store.GetProject(ctx, projectID)
	if err != nil {
		return ContextPack{}, err
	}
	docs, err := p.Store.ListProjectDocuments(ctx, activeProject.ID)
	if err != nil {
		return ContextPack{}, err
	}
	contextDocs := make([]project.Document, 0, len(docs))
	outDocs := make([]ContextDocument, 0, len(docs))
	for _, doc := range docs {
		contextDocs = append(contextDocs, project.Document{Kind: doc.Kind, Title: doc.Title, Body: doc.Body})
		outDocs = append(outDocs, ContextDocument{Kind: doc.Kind, Title: doc.Title, Body: doc.Body})
	}
	text := project.BuildContextWithPolicy(project.Project{
		ID:          activeProject.ID,
		Name:        activeProject.Name,
		Description: activeProject.Description,
		Status:      activeProject.Status,
	}, contextDocs, policy)
	return ContextPack{
		Project: ContextProject{
			ID:              activeProject.ID,
			Name:            activeProject.Name,
			Description:     activeProject.Description,
			Status:          activeProject.Status,
			StorageProvider: activeProject.StorageProvider,
			StorageBucket:   activeProject.StorageBucket,
			StoragePrefix:   activeProject.StoragePrefix,
		},
		ProjectID: activeProject.ID,
		Request:   strings.TrimSpace(request),
		Documents: outDocs,
		Policy:    policy,
		Text:      text,
	}, nil
}

func loadProjectDocumentPolicy(ctx context.Context, store ProjectStore) project.DocumentPolicy {
	defaultPolicy := project.DefaultDocumentPolicy()
	policyStore, ok := store.(projectPolicyStore)
	if !ok {
		return defaultPolicy
	}
	setting, err := policyStore.GetAppSetting(ctx, project.DocumentPolicySettingKey)
	if err != nil {
		return defaultPolicy
	}
	policy, err := project.ParseDocumentPolicy(setting.Value)
	if err != nil {
		return defaultPolicy
	}
	return policy
}

type RuntimeSkillRunner struct {
	Runtime *runtime.Runtime
}

func (r RuntimeSkillRunner) RunSkill(ctx context.Context, input SkillInput) (SkillOutput, error) {
	if r.Runtime == nil {
		return SkillOutput{}, fmt.Errorf("runtime is required")
	}
	previousContext := r.Runtime.ProjectContext
	if strings.TrimSpace(input.Context.Text) != "" {
		r.Runtime.ProjectContext = input.Context.Text
	}
	defer func() {
		r.Runtime.ProjectContext = previousContext
	}()
	out, err := r.Runtime.ExecuteSkill(ctx, input.SkillID, input.Request, skill.CloneMap(input.Arguments))
	if err != nil {
		return SkillOutput{}, err
	}
	return SkillOutput{
		SkillID: input.SkillID,
		Text:    out,
		RunID:   r.Runtime.Store.RunID,
		RunDir:  r.Runtime.Store.Root,
	}, nil
}

type SequentialWorkflowRunner struct {
	SkillRunner SkillRunner
}

func (r SequentialWorkflowRunner) RunWorkflow(ctx context.Context, input WorkflowInput) (WorkflowOutput, error) {
	if r.SkillRunner == nil {
		return WorkflowOutput{}, fmt.Errorf("skill runner is required")
	}
	out := WorkflowOutput{WorkflowID: strings.TrimSpace(input.WorkflowID)}
	out.StartedAt = nowUTC()
	for i, step := range input.Steps {
		stepID := strings.TrimSpace(step.ID)
		if stepID == "" {
			stepID = fmt.Sprintf("step-%02d", i+1)
		}
		skillOut, err := r.SkillRunner.RunSkill(ctx, SkillInput{
			SkillID:       step.SkillID,
			Request:       input.Request,
			Arguments:     mergeArguments(input.Arguments, step.Arguments),
			ProjectID:     input.ProjectID,
			Context:       input.Context,
			WorkflowRunID: input.WorkflowID,
		})
		if err != nil {
			return WorkflowOutput{}, err
		}
		out.Steps = append(out.Steps, WorkflowStepOutput{StepID: stepID, Output: skillOut})
	}
	out.FinishedAt = nowUTC()
	return out, nil
}

func skillSpecFromCommand(cmd skill.Command) SkillSpec {
	return SkillSpec{
		ID:             cmd.ID,
		Name:           cmd.Name,
		Version:        cmd.Version,
		Description:    cmd.Description,
		WhenToUse:      cmd.WhenToUse,
		Tags:           append([]string{}, cmd.Tags...),
		InputSchema:    skill.CloneMap(cmd.ToolInputSchema),
		OutputContract: cmd.ToolOutput,
		Source:         "local",
	}
}

func mergeArguments(base, override map[string]any) map[string]any {
	out := skill.CloneMap(base)
	if out == nil {
		out = map[string]any{}
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}
