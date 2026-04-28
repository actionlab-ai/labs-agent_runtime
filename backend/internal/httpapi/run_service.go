package httpapi

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	"novel-agent-runtime/internal/config"
	"novel-agent-runtime/internal/logging"
	"novel-agent-runtime/internal/project"
	"novel-agent-runtime/internal/store"
)

type runResponse struct {
	DryRun    bool                `json:"dry_run,omitempty"`
	FinalText string              `json:"final_text,omitempty"`
	RunID     string              `json:"run_id"`
	RunDir    string              `json:"run_dir"`
	DBRun     *store.Run          `json:"db_run,omitempty"`
	Project   *store.Project      `json:"project,omitempty"`
	Model     *store.ModelProfile `json:"model,omitempty"`
}

type RunService struct {
	db      appStore
	factory runtimeFactory
}

func NewRunService(cfg config.Config, debug bool, db appStore, models modelConfigStore, projects projectConfigStore) RunService {
	return RunService{
		db: db,
		factory: runtimeFactory{
			cfg:      cfg,
			debug:    debug,
			models:   models,
			projects: projects,
		},
	}
}

func (s RunService) Execute(ctx context.Context, req runRequest) (runResponse, error) {
	logger := logging.FromContext(ctx)
	logger.Info("run.request.accepted",
		zap.Bool("dry_run", req.DryRun),
		zap.Bool("explicit_project", strings.TrimSpace(req.Project) != ""),
		zap.Bool("explicit_model", strings.TrimSpace(req.Model) != ""),
		zap.Int("input_bytes", len(req.Input)),
	)

	selectedModelID, err := s.factory.resolveSelectedModelID(ctx, req.Model)
	if err != nil {
		return runResponse{}, err
	}
	logger.Info("run.model.selected",
		zap.String("selected_profile_id", project.Slug(selectedModelID)),
		zap.Bool("explicit_model", strings.TrimSpace(req.Model) != ""),
	)

	session, err := s.factory.resolveRunSession(ctx, req)
	if err != nil {
		return runResponse{}, err
	}
	logger.Info("run.model.loaded",
		zap.String("profile_id", session.activeModel.ID),
		zap.String("provider", session.activeModel.Provider),
		zap.String("model_id", session.activeModel.ModelID),
		zap.String("base_url", session.activeModel.BaseURL),
		zap.Int("timeout_seconds", session.runtimeModel.TimeoutSeconds),
	)

	projectID := strings.TrimSpace(req.Project)
	projectLogFields := []zap.Field{
		zap.String("project_id", project.Slug(projectID)),
		zap.Int("context_bytes", len(session.projectContext)),
	}
	if session.activeProject != nil {
		projectLogFields = append(projectLogFields,
			zap.String("active_project_id", session.activeProject.ID),
			zap.String("storage_provider", session.activeProject.StorageProvider),
			zap.String("storage_bucket", session.activeProject.StorageBucket),
			zap.String("storage_prefix", session.activeProject.StoragePrefix),
		)
	}
	logger.Info("run.project_context.loaded", projectLogFields...)

	if req.DryRun {
		logger.Info("run.dry_run.start", zap.String("profile_id", session.activeModel.ID), zap.String("project_id", session.runtime.ProjectID))
		if err := session.runtime.DryRun(req.Input); err != nil {
			return runResponse{}, withStatus(500, err)
		}
		logger.Info("run.dry_run.completed", zap.String("run_id", session.runtime.Store.RunID), zap.String("run_dir", session.runtime.Store.Root))
		return runResponse{
			DryRun:  true,
			RunID:   session.runtime.Store.RunID,
			RunDir:  session.runtime.Store.Root,
			Project: session.activeProject,
			Model:   session.activeModel,
		}, nil
	}

	dbRun, err := s.db.CreateRun(ctx, projectID, req.Input, nil)
	if err != nil {
		return runResponse{}, withStatus(500, err)
	}
	logger.Info("run.pg.create", zap.Int64("db_run_id", dbRun.ID), zap.String("project_id", project.Slug(projectID)))

	timeout := time.Duration(session.runtimeModel.TimeoutSeconds+30) * time.Second
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	res, err := session.runtime.Run(runCtx, req.Input)
	if err != nil {
		_, _ = s.db.FailRun(ctx, dbRun.ID, err.Error())
		logger.Error("run.runtime.failed", zap.Int64("db_run_id", dbRun.ID), zap.Error(err))
		return runResponse{}, withStatus(500, err)
	}

	finished, err := s.db.FinishRun(ctx, dbRun.ID, res.FinalText, res.RunDir)
	if err != nil {
		return runResponse{}, withStatus(500, err)
	}
	logger.Info("run.completed",
		zap.Int64("db_run_id", finished.ID),
		zap.String("run_id", res.RunID),
		zap.String("run_dir", res.RunDir),
		zap.Int("final_text_bytes", len(res.FinalText)),
	)

	return runResponse{
		FinalText: res.FinalText,
		RunID:     res.RunID,
		RunDir:    res.RunDir,
		DBRun:     &finished,
		Project:   session.activeProject,
		Model:     session.activeModel,
	}, nil
}
