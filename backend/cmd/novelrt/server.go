package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"novel-agent-runtime/internal/config"
	"novel-agent-runtime/internal/project"
	"novel-agent-runtime/internal/runtime"
	"novel-agent-runtime/internal/skill"
	"novel-agent-runtime/internal/store"
)

type appStore interface {
	CreateProject(context.Context, store.CreateProjectParams) (store.Project, error)
	GetProject(context.Context, string) (store.Project, error)
	ListProjects(context.Context, int32, int32) ([]store.Project, error)
	UpdateProject(context.Context, store.UpdateProjectParams) (store.Project, error)
	DeleteProject(context.Context, string) error
	CreateModelProfile(context.Context, store.CreateModelProfileParams) (store.ModelProfile, error)
	GetModelProfile(context.Context, string) (store.ModelProfile, error)
	ListModelProfiles(context.Context, int32, int32) ([]store.ModelProfile, error)
	UpdateModelProfile(context.Context, store.UpdateModelProfileParams) (store.ModelProfile, error)
	DeleteModelProfile(context.Context, string) error
	UpsertProjectDocument(context.Context, store.UpsertProjectDocumentParams) (store.ProjectDocument, error)
	ListProjectDocuments(context.Context, string) ([]store.ProjectDocument, error)
	CreateRun(context.Context, string, string, json.RawMessage) (store.Run, error)
	FinishRun(context.Context, int64, string, string) (store.Run, error)
	FailRun(context.Context, int64, string) (store.Run, error)
}

type projectCreateRequest struct {
	ID          string          `json:"id"`
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Metadata    json.RawMessage `json:"metadata"`
}

type projectUpdateRequest struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Metadata    json.RawMessage `json:"metadata"`
}

type projectDocumentUpsertRequest struct {
	Kind     string          `json:"kind"`
	Title    string          `json:"title"`
	Body     string          `json:"body"`
	Metadata json.RawMessage `json:"metadata"`
}

type runRequest struct {
	Input   string `json:"input" binding:"required"`
	Project string `json:"project"`
	Model   string `json:"model"`
	DryRun  bool   `json:"dry_run"`
	Debug   *bool  `json:"debug"`
}

type modelProfileCreateRequest struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Provider        string          `json:"provider"`
	ModelID         string          `json:"model_id" binding:"required"`
	BaseURL         string          `json:"base_url" binding:"required"`
	APIKey          string          `json:"api_key"`
	ContextWindow   int             `json:"context_window"`
	MaxOutputTokens int             `json:"max_output_tokens"`
	Temperature     float64         `json:"temperature"`
	TimeoutSeconds  int             `json:"timeout_seconds"`
	Metadata        json.RawMessage `json:"metadata"`
}

type modelProfileUpdateRequest struct {
	Name            string          `json:"name"`
	Provider        string          `json:"provider"`
	ModelID         string          `json:"model_id" binding:"required"`
	BaseURL         string          `json:"base_url" binding:"required"`
	APIKey          string          `json:"api_key"`
	ContextWindow   int             `json:"context_window"`
	MaxOutputTokens int             `json:"max_output_tokens"`
	Temperature     float64         `json:"temperature"`
	TimeoutSeconds  int             `json:"timeout_seconds"`
	Status          string          `json:"status"`
	Metadata        json.RawMessage `json:"metadata"`
}

func serveHTTP(cfg config.Config, addr string, debug bool) error {
	db, err := openConfiguredStore(context.Background(), cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	router := buildHTTPRouter(cfg, debug, db)
	return router.Run(addr)
}

func buildHTTPRouter(cfg config.Config, debug bool, db appStore) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	router.GET("/v1/skills", func(c *gin.Context) {
		reg, err := skill.LoadRegistry(cfg.Runtime.SkillsDir)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"skills": reg.List()})
	})

	router.GET("/v1/projects", func(c *gin.Context) {
		limit, offset := pagination(c)
		items, err := db.ListProjects(c.Request.Context(), limit, offset)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"projects": items})
	})

	router.POST("/v1/projects", func(c *gin.Context) {
		var req projectCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		p, err := db.CreateProject(c.Request.Context(), store.CreateProjectParams{
			ID:          req.ID,
			Name:        req.Name,
			Description: req.Description,
			Metadata:    req.Metadata,
		})
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"project": p})
	})

	router.GET("/v1/projects/:id", func(c *gin.Context) {
		p, err := db.GetProject(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeHTTPError(c, http.StatusNotFound, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"project": p})
	})

	router.PATCH("/v1/projects/:id", func(c *gin.Context) {
		var req projectUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		p, err := db.UpdateProject(c.Request.Context(), store.UpdateProjectParams{
			ID:          c.Param("id"),
			Name:        req.Name,
			Description: req.Description,
			Status:      req.Status,
			Metadata:    req.Metadata,
		})
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"project": p})
	})

	router.DELETE("/v1/projects/:id", func(c *gin.Context) {
		if err := db.DeleteProject(c.Request.Context(), c.Param("id")); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	})

	router.GET("/v1/models", func(c *gin.Context) {
		limit, offset := pagination(c)
		items, err := db.ListModelProfiles(c.Request.Context(), limit, offset)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"models": items})
	})

	router.POST("/v1/models", func(c *gin.Context) {
		var req modelProfileCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		item, err := db.CreateModelProfile(c.Request.Context(), store.CreateModelProfileParams{
			ID:              req.ID,
			Name:            req.Name,
			Provider:        req.Provider,
			ModelID:         req.ModelID,
			BaseURL:         req.BaseURL,
			APIKey:          req.APIKey,
			ContextWindow:   req.ContextWindow,
			MaxOutputTokens: req.MaxOutputTokens,
			Temperature:     req.Temperature,
			TimeoutSeconds:  req.TimeoutSeconds,
			Metadata:        req.Metadata,
		})
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"model": item})
	})

	router.GET("/v1/models/:id", func(c *gin.Context) {
		item, err := db.GetModelProfile(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeHTTPError(c, http.StatusNotFound, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"model": item})
	})

	router.PATCH("/v1/models/:id", func(c *gin.Context) {
		var req modelProfileUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		item, err := db.UpdateModelProfile(c.Request.Context(), store.UpdateModelProfileParams{
			ID:              c.Param("id"),
			Name:            req.Name,
			Provider:        req.Provider,
			ModelID:         req.ModelID,
			BaseURL:         req.BaseURL,
			APIKey:          req.APIKey,
			ContextWindow:   req.ContextWindow,
			MaxOutputTokens: req.MaxOutputTokens,
			Temperature:     req.Temperature,
			TimeoutSeconds:  req.TimeoutSeconds,
			Status:          req.Status,
			Metadata:        req.Metadata,
		})
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"model": item})
	})

	router.DELETE("/v1/models/:id", func(c *gin.Context) {
		if err := db.DeleteModelProfile(c.Request.Context(), c.Param("id")); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	})

	router.GET("/v1/projects/:id/documents", func(c *gin.Context) {
		docs, err := db.ListProjectDocuments(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"documents": docs})
	})

	router.PUT("/v1/projects/:id/documents/:kind", func(c *gin.Context) {
		var req projectDocumentUpsertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		kind := firstNonEmpty(req.Kind, c.Param("kind"))
		doc, err := db.UpsertProjectDocument(c.Request.Context(), store.UpsertProjectDocumentParams{
			ProjectID: c.Param("id"),
			Kind:      kind,
			Title:     firstNonEmpty(req.Title, kind),
			Body:      req.Body,
			Metadata:  req.Metadata,
		})
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"document": doc})
	})

	router.POST("/v1/runs", func(c *gin.Context) {
		var req runRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		runCfg := cfg
		activeModel, err := applyModelProfile(c.Request.Context(), db, &runCfg, req.Model)
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		projectID := firstNonEmpty(req.Project, runCfg.Runtime.ProjectID)
		projectContext, activeProject, err := loadProjectContext(c.Request.Context(), db, projectID)
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		if activeProject != nil {
			runCfg.Runtime.ProjectID = activeProject.ID
		}

		rt, err := runtime.New(runCfg)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		rt.ProjectContext = projectContext
		rt.Debug = debug
		if req.Debug != nil {
			rt.Debug = *req.Debug
		}

		if req.DryRun {
			if err := rt.DryRun(req.Input); err != nil {
				writeHTTPError(c, http.StatusInternalServerError, err)
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"dry_run": true,
				"run_id":  rt.Store.RunID,
				"run_dir": rt.Store.Root,
				"project": activeProject,
				"model":   activeModel,
			})
			return
		}

		dbRun, err := db.CreateRun(c.Request.Context(), projectID, req.Input, nil)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		timeout := time.Duration(runCfg.Model.TimeoutSeconds+30) * time.Second
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		res, err := rt.Run(ctx, req.Input)
		if err != nil {
			_, _ = db.FailRun(c.Request.Context(), dbRun.ID, err.Error())
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		finished, err := db.FinishRun(c.Request.Context(), dbRun.ID, res.FinalText, res.RunDir)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"final_text": res.FinalText,
			"run_id":     res.RunID,
			"run_dir":    res.RunDir,
			"db_run":     finished,
			"project":    activeProject,
			"model":      activeModel,
		})
	})

	return router
}

func applyModelProfile(ctx context.Context, db appStore, cfg *config.Config, modelID string) (*store.ModelProfile, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, nil
	}
	profile, err := db.GetModelProfile(ctx, modelID)
	if err != nil {
		return nil, err
	}
	if profile.Status != "" && profile.Status != "active" {
		return nil, fmt.Errorf("model profile %q is not active", profile.ID)
	}
	cfg.Model.Provider = firstNonEmpty(profile.Provider, cfg.Model.Provider)
	cfg.Model.ID = profile.ModelID
	cfg.Model.BaseURL = profile.BaseURL
	cfg.Model.APIKey = profile.APIKey
	cfg.Model.APIKeyEnv = profile.APIKeyEnv
	if profile.ContextWindow > 0 {
		cfg.Model.ContextWindow = profile.ContextWindow
	}
	if profile.MaxOutputTokens > 0 {
		cfg.Model.MaxOutput = profile.MaxOutputTokens
	}
	if profile.Temperature != 0 {
		cfg.Model.Temperature = profile.Temperature
	}
	if profile.TimeoutSeconds > 0 {
		cfg.Model.TimeoutSeconds = profile.TimeoutSeconds
	}
	return &profile, nil
}

func openConfiguredStore(ctx context.Context, cfg config.Config) (*store.Store, error) {
	if strings.TrimSpace(cfg.Database.URL) == "" {
		return nil, fmt.Errorf("database.url is required; set DATABASE_URL or NOVEL_DATABASE_URL")
	}
	if cfg.Database.AutoMigrate {
		if err := store.Migrate(cfg.Database.URL, cfg.Database.MigrationsDir); err != nil {
			return nil, fmt.Errorf("migrate database: %w", err)
		}
	}
	return store.Open(ctx, cfg.Database.URL)
}

func loadProjectContext(ctx context.Context, db appStore, projectID string) (string, *store.Project, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return "", nil, nil
	}
	p, err := db.GetProject(ctx, projectID)
	if err != nil {
		return "", nil, err
	}
	docs, err := db.ListProjectDocuments(ctx, p.ID)
	if err != nil {
		return "", nil, err
	}
	contextDocs := make([]project.Document, 0, len(docs))
	for _, doc := range docs {
		contextDocs = append(contextDocs, project.Document{Kind: doc.Kind, Title: doc.Title, Body: doc.Body})
	}
	contextText := project.BuildContext(project.Project{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Status:      p.Status,
	}, contextDocs)
	return contextText, &p, nil
}

func pagination(c *gin.Context) (int32, int32) {
	limit := int32(50)
	offset := int32(0)
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = int32(parsed)
		}
	}
	if raw := strings.TrimSpace(c.Query("offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			offset = int32(parsed)
		}
	}
	return limit, offset
}

func writeHTTPError(c *gin.Context, status int, err error) {
	msg := "request failed"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		msg = err.Error()
	}
	c.JSON(status, gin.H{"error": msg})
}
