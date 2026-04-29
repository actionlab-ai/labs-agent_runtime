package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"novel-agent-runtime/internal/config"
	"novel-agent-runtime/internal/logging"
	"novel-agent-runtime/internal/skillsession"
	"novel-agent-runtime/internal/store"
)

type skillSessionStartRequest struct {
	Project   string         `json:"project" binding:"required"`
	Model     string         `json:"model"`
	SkillID   string         `json:"skill_id" binding:"required"`
	Input     string         `json:"input" binding:"required"`
	Arguments map[string]any `json:"arguments,omitempty"`
	Debug     *bool          `json:"debug"`
}

type skillSessionContinueRequest struct {
	Input   string            `json:"input,omitempty"`
	Answers map[string]string `json:"answers,omitempty"`
	Notes   string            `json:"notes,omitempty"`
}

type skillSessionResponse struct {
	Session skillsession.Snapshot `json:"session"`
	Project *store.Project        `json:"project,omitempty"`
	Model   *store.ModelProfile   `json:"model,omitempty"`
}

type SkillSessionService struct {
	manager *skillsession.Manager
	factory runtimeFactory
}

func NewSkillSessionService(cfg config.Config, debug bool, manager *skillsession.Manager, models modelConfigStore, projects projectConfigStore) SkillSessionService {
	return SkillSessionService{
		manager: manager,
		factory: runtimeFactory{
			cfg:      cfg,
			debug:    debug,
			models:   models,
			projects: projects,
		},
	}
}

func (s SkillSessionService) Start(ctx context.Context, req skillSessionStartRequest) (skillSessionResponse, error) {
	if s.manager == nil {
		return skillSessionResponse{}, withStatus(http.StatusInternalServerError, errSkillSessionManagerUnavailable())
	}
	req.SkillID = strings.TrimSpace(req.SkillID)
	req.Input = strings.TrimSpace(req.Input)
	if req.SkillID == "" {
		return skillSessionResponse{}, withStatus(http.StatusBadRequest, errRequiredField("skill_id"))
	}
	if req.Input == "" {
		return skillSessionResponse{}, withStatus(http.StatusBadRequest, errRequiredField("input"))
	}

	session, err := s.factory.resolveWorkflowSession(ctx, workflowRunRequest{
		Input:   req.Input,
		Project: req.Project,
		Model:   req.Model,
		Debug:   req.Debug,
	})
	if err != nil {
		return skillSessionResponse{}, err
	}

	logger := logging.FromContext(ctx)
	logger.Info("skill_session.start.accepted",
		zap.String("skill_id", req.SkillID),
		zap.String("project_id", session.activeProject.ID),
		zap.String("profile_id", session.activeModel.ID),
		zap.String("provider", session.activeModel.Provider),
		zap.String("model_id", session.activeModel.ModelID),
		zap.String("base_url", session.activeModel.BaseURL),
		zap.Int("timeout_seconds", session.runtimeModel.TimeoutSeconds),
		zap.Int("max_output_tokens", session.runtimeModel.MaxOutput),
		zap.Int("input_bytes", len(req.Input)),
		zap.Bool("explicit_model", strings.TrimSpace(req.Model) != ""),
		zap.Bool("debug_artifacts", session.runtime.Debug),
	)

	timeout := time.Duration(session.runtimeModel.TimeoutSeconds+30) * time.Second
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	snapshot, err := s.manager.Start(runCtx, session.runtime, skillsession.StartInput{
		ProjectID: session.activeProject.ID,
		SkillID:   req.SkillID,
		Request:   req.Input,
		Arguments: cloneMap(req.Arguments),
	})
	if err != nil {
		return skillSessionResponse{}, withStatus(http.StatusInternalServerError, err)
	}
	logger.Info("skill_session.start.completed",
		zap.String("session_id", snapshot.ID),
		zap.String("status", snapshot.Status),
		zap.String("skill_id", snapshot.SkillID),
		zap.String("run_id", snapshot.RunID),
		zap.Bool("needs_input", snapshot.Status == skillsession.StatusNeedsInput),
	)
	return skillSessionResponse{Session: snapshot, Project: session.activeProject, Model: session.activeModel}, nil
}

func (s SkillSessionService) Continue(ctx context.Context, id string, req skillSessionContinueRequest) (skillSessionResponse, error) {
	if s.manager == nil {
		return skillSessionResponse{}, withStatus(http.StatusInternalServerError, errSkillSessionManagerUnavailable())
	}
	logger := logging.FromContext(ctx)
	logger.Info("skill_session.continue.accepted",
		zap.String("session_id", id),
		zap.Int("answer_count", len(req.Answers)),
		zap.Int("input_bytes", len(req.Input)),
	)
	snapshot, err := s.manager.Continue(ctx, id, skillsession.ContinueInput{
		Input:   req.Input,
		Answers: cloneStringMap(req.Answers),
		Notes:   req.Notes,
	})
	if err != nil {
		return skillSessionResponse{}, withStatus(http.StatusBadRequest, err)
	}
	logger.Info("skill_session.continue.completed",
		zap.String("session_id", snapshot.ID),
		zap.String("status", snapshot.Status),
		zap.String("skill_id", snapshot.SkillID),
		zap.String("run_id", snapshot.RunID),
		zap.Bool("needs_input", snapshot.Status == skillsession.StatusNeedsInput),
	)
	return skillSessionResponse{Session: snapshot}, nil
}

func (s SkillSessionService) Get(id string) (skillSessionResponse, error) {
	if s.manager == nil {
		return skillSessionResponse{}, withStatus(http.StatusInternalServerError, errSkillSessionManagerUnavailable())
	}
	snapshot, ok := s.manager.Get(id)
	if !ok {
		return skillSessionResponse{}, withStatus(http.StatusNotFound, errSkillSessionNotFound(id))
	}
	return skillSessionResponse{Session: snapshot}, nil
}

func registerSkillSessionRoutes(router *gin.Engine, deps routeDeps) {
	service := NewSkillSessionService(deps.cfg, deps.debug, deps.skills, deps.models, deps.projects)

	router.POST("/v1/skill-sessions", func(c *gin.Context) {
		var req skillSessionStartRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		resp, err := service.Start(c.Request.Context(), req)
		if err != nil {
			writeHTTPError(c, errorStatus(err, http.StatusInternalServerError), err)
			return
		}
		c.JSON(http.StatusOK, resp)
	})

	router.GET("/v1/skill-sessions/:id", func(c *gin.Context) {
		resp, err := service.Get(c.Param("id"))
		if err != nil {
			writeHTTPError(c, errorStatus(err, http.StatusInternalServerError), err)
			return
		}
		c.JSON(http.StatusOK, resp)
	})

	router.POST("/v1/skill-sessions/:id/turns", func(c *gin.Context) {
		var req skillSessionContinueRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		resp, err := service.Continue(c.Request.Context(), c.Param("id"), req)
		if err != nil {
			writeHTTPError(c, errorStatus(err, http.StatusInternalServerError), err)
			return
		}
		c.JSON(http.StatusOK, resp)
	})
}

func cloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func errRequiredField(name string) error {
	return fmt.Errorf("%s is required", name)
}

func errSkillSessionManagerUnavailable() error {
	return fmt.Errorf("skill session manager is unavailable")
}

func errSkillSessionNotFound(id string) error {
	return fmt.Errorf("skill session %q not found", strings.TrimSpace(id))
}
