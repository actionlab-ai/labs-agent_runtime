package httpapi

import (
	"context"
	"fmt"
	"strings"

	"novel-agent-runtime/internal/config"
	"novel-agent-runtime/internal/runtime"
	"novel-agent-runtime/internal/store"
	"novel-agent-runtime/internal/workflow"
)

type runtimeFactory struct {
	cfg      config.Config
	debug    bool
	models   modelConfigStore
	projects projectConfigStore
}

type runSession struct {
	activeModel    *store.ModelProfile
	activeProject  *store.Project
	projectContext string
	runtimeModel   runtime.ModelConfig
	runtime        *runtime.Runtime
}

type workflowSession struct {
	activeModel   *store.ModelProfile
	activeProject *store.Project
	contextPack   workflow.ContextPack
	runtimeModel  runtime.ModelConfig
	runtime       *runtime.Runtime
}

func (f runtimeFactory) resolveRunSession(ctx context.Context, req runRequest) (runSession, error) {
	selectedModelID, err := f.resolveSelectedModelID(ctx, req.Model)
	if err != nil {
		return runSession{}, err
	}
	activeModel, runtimeModel, err := loadRuntimeModelConfig(ctx, f.models, selectedModelID)
	if err != nil {
		return runSession{}, withStatus(400, err)
	}
	projectID := strings.TrimSpace(req.Project)
	projectContext, activeProject, err := loadProjectContext(ctx, f.projects, projectID)
	if err != nil {
		return runSession{}, withStatus(400, err)
	}
	rt, err := f.buildRuntime(runtimeModel, activeProject, projectContext, req.Debug)
	if err != nil {
		return runSession{}, err
	}
	return runSession{
		activeModel:    activeModel,
		activeProject:  activeProject,
		projectContext: projectContext,
		runtimeModel:   runtimeModel,
		runtime:        rt,
	}, nil
}

func (f runtimeFactory) resolveWorkflowSession(ctx context.Context, req workflowRunRequest) (workflowSession, error) {
	selectedModelID, err := f.resolveSelectedModelID(ctx, req.Model)
	if err != nil {
		return workflowSession{}, err
	}
	activeModel, runtimeModel, err := loadRuntimeModelConfig(ctx, f.models, selectedModelID)
	if err != nil {
		return workflowSession{}, withStatus(400, err)
	}
	contextPack, activeProject, err := loadProjectContextPack(ctx, f.projects, req.Project, req.Input)
	if err != nil {
		return workflowSession{}, withStatus(400, err)
	}
	if activeProject == nil {
		return workflowSession{}, withStatus(400, fmt.Errorf("project %q not found", req.Project))
	}
	rt, err := f.buildRuntime(runtimeModel, activeProject, contextPack.Text, req.Debug)
	if err != nil {
		return workflowSession{}, err
	}
	return workflowSession{
		activeModel:   activeModel,
		activeProject: activeProject,
		contextPack:   contextPack,
		runtimeModel:  runtimeModel,
		runtime:       rt,
	}, nil
}

func (f runtimeFactory) resolveSelectedModelID(ctx context.Context, explicitModel string) (string, error) {
	defaultModelID, err := f.models.GetDefaultModelID(ctx)
	if err != nil {
		return "", withStatus(500, err)
	}
	selectedModelID := firstNonEmpty(explicitModel, defaultModelID)
	if strings.TrimSpace(selectedModelID) == "" {
		return "", withStatus(400, errRequiredModel())
	}
	return selectedModelID, nil
}

func (f runtimeFactory) buildRuntime(runtimeModel runtime.ModelConfig, activeProject *store.Project, projectContext string, reqDebug *bool) (*runtime.Runtime, error) {
	rt, err := runtime.New(f.cfg, runtimeModel)
	if err != nil {
		return nil, withStatus(500, err)
	}
	rt.ProjectDocs = runtimeProjectDocumentProvider{Projects: f.projects}
	if activeProject != nil {
		rt.ProjectID = activeProject.ID
	}
	rt.ProjectContext = projectContext
	rt.Debug = f.debug
	if reqDebug != nil {
		rt.Debug = *reqDebug
	}
	return rt, nil
}
