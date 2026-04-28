// Package main 实现了 Novel Agent Runtime 后端的 HTTP 服务器入口。
//
// 概述:
// 该服务提供了一套 RESTful API，用于管理 AI 辅助写作项目、配置大语言模型 (LLM) 配置文件以及执行 Agent 运行任务。
// 它充当了客户端前端、持久化存储层和底层 AI 运行时引擎之间的编排层。
//
// 核心功能:
// 1. 项目管理: 对写作项目进行 CRUD 操作，包括元数据管理和状态跟踪。
// 2. 上下文管理: 上传和检索项目特定的文档（如大纲、角色设定表），以构建丰富的 LLM 提示词上下文。
// 3. 模型配置: 管理多个 LLM 提供商的配置（API 密钥、端点、参数），并设置全局默认模型以优化用户体验。
// 4. Agent 执行: 处理同步的 AI 生成请求，支持标准运行模式和用于调试提示词构建的干跑（Dry-run）模式。
//
// 业务场景:
// - AI 辅助小说创作: 用户创建项目，上传上下文，并使用配置的 LLM 生成章节内容。
// - 多模型编排: 管理员配置各种 LLM 提供商，用户可以动态切换使用不同的模型。
// - 调试与测试: 开发者使用干跑模式验证上下文组装是否正确，而无需产生实际的 LLM 调用成本。
//
// 架构说明:
// - HTTP 框架: Gin
// - 存储接口: appStore (抽象数据库操作)
// - 运行时: 将实际的 AI 执行委托给 internal/runtime 包
//
// @title Novel Agent Runtime API
// @version 1.0
// @description 用于管理 AI 写作 Agent 和项目的后端 API。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"novel-agent-runtime/internal/cache"
	"novel-agent-runtime/internal/config"
	"novel-agent-runtime/internal/logging"
	"novel-agent-runtime/internal/project"
	"novel-agent-runtime/internal/projectfs"
	"novel-agent-runtime/internal/runtime"
	"novel-agent-runtime/internal/store"
	"novel-agent-runtime/internal/workflow"
)

// appStore 定义了数据持久层的核心接口。
// 业务逻辑：该接口抽象了数据库操作，使得上层业务逻辑（如 HTTP 处理）不依赖于具体的数据库实现。
// 它涵盖了项目管理、模型配置管理、文档管理以及 AI 运行记录的管理。
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

	// 默认模型管理相关方法
	// 业务逻辑：允许用户在数据库中设置一个全局默认的 LLM 模型，避免每次请求都需指定模型 ID。
	GetDefaultModelID(context.Context) (string, error)
	SetDefaultModelID(context.Context, string) error
	ClearDefaultModelID(context.Context) error

	UpsertProjectDocument(context.Context, store.UpsertProjectDocumentParams) (store.ProjectDocument, error)
	ListProjectDocuments(context.Context, string) ([]store.ProjectDocument, error)
	CreateRun(context.Context, string, string, json.RawMessage) (store.Run, error)
	FinishRun(context.Context, int64, string, string) (store.Run, error)
	FailRun(context.Context, int64, string) (store.Run, error)
}

// projectCreateRequest 是创建新项目时的请求体结构。
// 用户场景：用户在前端点击“新建项目”，输入项目名称和描述后提交。
type projectCreateRequest struct {
	ID              string          `json:"id"`
	Name            string          `json:"name" binding:"required"`
	Description     string          `json:"description"`
	StorageProvider string          `json:"storage_provider"`
	StorageBucket   string          `json:"storage_bucket"`
	StoragePrefix   string          `json:"storage_prefix"`
	Metadata        json.RawMessage `json:"metadata"`
}

// projectUpdateRequest 是更新现有项目时的请求体结构。
// 用户场景：用户修改项目状态为“已完结”，或更新项目描述以反映最新剧情走向。
type projectUpdateRequest struct {
	Name            string          `json:"name" binding:"required"`
	Description     string          `json:"description"`
	Status          string          `json:"status"`
	StorageProvider string          `json:"storage_provider"`
	StorageBucket   string          `json:"storage_bucket"`
	StoragePrefix   string          `json:"storage_prefix"`
	Metadata        json.RawMessage `json:"metadata"`
}

// projectDocumentUpsertRequest 是上传或更新项目文档时的请求体结构。
// 业务逻辑：文档作为 AI 的上下文知识（如大纲、角色设定、前文摘要），通过此接口存入数据库。
// 用户场景：用户上传一份“第一章大纲”或“主角性格设定”，以便 AI 在生成后续内容时保持一致性。
type projectDocumentUpsertRequest struct {
	Kind     string          `json:"kind"`
	Title    string          `json:"title"`
	Body     string          `json:"body"`
	Metadata json.RawMessage `json:"metadata"`
}

// runRequest 是执行 AI 代理任务时的请求体结构。
// 用户场景：用户在聊天界面输入“请根据大纲写出第二章”，并可选指定使用的模型或项目。
type runRequest struct {
	Input   string `json:"input" binding:"required"` // AI 的提示词或指令
	Project string `json:"project"`                  // 可选：关联的项目 ID
	Model   string `json:"model"`                    // 可选：指定使用的模型 ID
	DryRun  bool   `json:"dry_run"`                  // 是否仅进行干跑（测试上下文构建，不实际调用 LLM）
	Debug   *bool  `json:"debug"`                    // 是否开启调试模式
}

// modelProfileCreateRequest 是注册新 LLM 模型配置时的请求体结构。
// 用户场景：管理员添加一个新的 OpenAI 兼容接口，或者配置本地部署的 Llama 模型地址和密钥。
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

// modelProfileUpdateRequest 是更新现有 LLM 模型配置时的请求体结构。
// 用户场景：管理员轮换 API Key，或调整模型的 Temperature 参数以改变生成内容的创造性。
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

// defaultModelUpdateRequest 是设置全局默认模型时的请求体结构。
// 用户场景：用户在设置页面选择“始终使用 GPT-4o 作为默认模型”，简化日常使用流程。
type defaultModelUpdateRequest struct {
	Model string `json:"model" binding:"required"`
}

// serveHTTP 初始化数据库连接并启动 HTTP 服务。
// 它是 Web 服务生命周期的入口点。
func serveHTTP(cfg config.Config, addr string, debug bool) error {
	logger, err := logging.New(cfg.Logging)
	if err != nil {
		return fmt.Errorf("open logger: %w", err)
	}
	defer func() {
		_ = logger.Sync()
	}()
	logger.Info("service.starting",
		zap.String("addr", addr),
		zap.String("config", cfg.Path),
		zap.Bool("debug", debug),
		zap.String("log_level", cfg.Logging.Level),
		zap.String("log_encoding", cfg.Logging.Encoding),
	)

	rootCtx := logging.WithLogger(context.Background(), logger)
	db, err := openConfiguredStore(rootCtx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	configCache, err := cache.OpenRedisConfigCache(rootCtx, cfg.Redis)
	if err != nil {
		if cfg.Redis.Required {
			return fmt.Errorf("open redis config cache: %w", err)
		}
		logger.Warn("redis.cache.disabled",
			zap.String("mode", cfg.Redis.Mode),
			zap.Strings("addrs", cfg.Redis.Addrs),
			zap.Error(err),
		)
		configCache = nil
	} else if configCache != nil {
		logger.Info("redis.cache.enabled",
			zap.String("mode", cfg.Redis.Mode),
			zap.Strings("addrs", cfg.Redis.Addrs),
			zap.String("key_prefix", cfg.Redis.KeyPrefix),
			zap.Int("ttl_seconds", cfg.Redis.TTLSeconds),
		)
	} else {
		logger.Info("redis.cache.skipped", zap.Bool("enabled", cfg.Redis.Enabled))
	}
	defer configCache.Close()
	router := buildHTTPRouter(cfg, debug, db, configCache, logger)
	logger.Info("service.listening", zap.String("addr", addr))
	return router.Run(addr)
}

// buildHTTPRouter 构建 Gin 路由引擎并注册所有 API 端点。
// 它负责将 HTTP 请求映射到具体的业务处理逻辑。
func buildHTTPRouter(cfg config.Config, debug bool, db appStore, configCache *cache.RedisConfigCache, logger *zap.Logger) *gin.Engine {
	if logger == nil {
		logger = zap.NewNop()
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(requestLoggerMiddleware(logger))
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logging.FromContext(c.Request.Context()).Error("http.panic", zap.Any("panic", recovered))
		writeHTTPError(c, http.StatusInternalServerError, fmt.Errorf("internal server error"))
	}))
	projectFiles := projectfs.New(filepath.Join(cfg.Runtime.WorkspaceRoot, "projects"))
	models := modelConfigStore{DB: db, Cache: configCache}
	projects := projectConfigStore{DB: db, Cache: configCache, Files: projectFiles}

	// 健康检查接口
	// 用户场景：Kubernetes 或负载均衡器定期调用此接口以确认服务是否正常运行。
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// 获取可用技能列表
	// 业务逻辑：从文件系统加载已注册的 Agent 技能（Skills）。
	// 用户场景：前端展示可用的 AI 能力插件供用户选择。
	router.GET("/v1/skills", func(c *gin.Context) {
		provider := workflow.LocalSkillProvider{SkillsDir: cfg.Runtime.SkillsDir}
		skills, err := provider.ListSkills(c.Request.Context())
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		logging.FromContext(c.Request.Context()).Info("skill.provider.list", zap.Int("count", len(skills)))
		c.JSON(http.StatusOK, gin.H{"skills": skills})
	})

	// --- 项目管理接口 ---

	// 列出所有项目（支持分页）
	// 用户场景：用户在仪表盘查看自己创建的所有小说项目列表。
	router.GET("/v1/projects", func(c *gin.Context) {
		limit, offset := pagination(c)
		items, err := projects.ListProjects(c.Request.Context(), limit, offset)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"projects": items})
	})

	// 创建新项目
	// 用户场景：用户初始化一个新的写作项目。
	router.POST("/v1/projects", func(c *gin.Context) {
		var req projectCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		p, err := projects.CreateProject(c.Request.Context(), store.CreateProjectParams{
			ID:              req.ID,
			Name:            req.Name,
			Description:     req.Description,
			StorageProvider: req.StorageProvider,
			StorageBucket:   req.StorageBucket,
			StoragePrefix:   req.StoragePrefix,
			Metadata:        req.Metadata,
		})
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"project": p})
	})

	// 获取项目详情
	// 用户场景：用户点击进入某个具体项目，查看其元数据。
	router.GET("/v1/projects/:id", func(c *gin.Context) {
		p, err := projects.GetProject(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeHTTPError(c, http.StatusNotFound, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"project": p})
	})

	// 更新项目信息
	// 用户场景：用户修改项目名称或标记项目为“归档”。
	router.PATCH("/v1/projects/:id", func(c *gin.Context) {
		var req projectUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		p, err := projects.UpdateProject(c.Request.Context(), store.UpdateProjectParams{
			ID:              c.Param("id"),
			Name:            req.Name,
			Description:     req.Description,
			Status:          req.Status,
			StorageProvider: req.StorageProvider,
			StorageBucket:   req.StorageBucket,
			StoragePrefix:   req.StoragePrefix,
			Metadata:        req.Metadata,
		})
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"project": p})
	})

	// 删除项目
	// 用户场景：用户清理不再需要的废弃项目。
	router.DELETE("/v1/projects/:id", func(c *gin.Context) {
		if err := projects.DeleteProject(c.Request.Context(), c.Param("id")); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	})

	// --- 模型配置管理接口 ---

	// 列出所有模型配置
	// 用户场景：管理员查看系统中已配置的所有 LLM 提供商和模型。
	router.GET("/v1/models", func(c *gin.Context) {
		limit, offset := pagination(c)
		items, err := db.ListModelProfiles(c.Request.Context(), limit, offset)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"models": items})
	})

	// 创建新的模型配置
	// 用户场景：添加一个新的自定义模型端点。
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
		logging.FromContext(c.Request.Context()).Info("model.pg.create",
			zap.String("profile_id", item.ID),
			zap.String("model_id", item.ModelID),
			zap.String("provider", item.Provider),
		)
		models.CacheModelProfile(c.Request.Context(), item)
		c.JSON(http.StatusOK, gin.H{"model": item})
	})

	// 获取模型配置详情
	// 用户场景：查看特定模型的详细参数设置。
	router.GET("/v1/models/:id", func(c *gin.Context) {
		item, err := models.GetModelProfile(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeHTTPError(c, http.StatusNotFound, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"model": item})
	})

	// 更新模型配置
	// 用户场景：修改模型的超时时间或最大输出长度。
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
		logging.FromContext(c.Request.Context()).Info("model.pg.update",
			zap.String("profile_id", item.ID),
			zap.String("model_id", item.ModelID),
			zap.String("provider", item.Provider),
			zap.String("status", item.Status),
		)
		models.CacheModelProfile(c.Request.Context(), item)
		c.JSON(http.StatusOK, gin.H{"model": item})
	})

	// 删除模型配置
	// 业务逻辑：在删除前检查该模型是否被设为“默认模型”，如果是则禁止删除以保护系统稳定性。
	// 用户场景：移除一个已停用的模型提供商配置。
	router.DELETE("/v1/models/:id", func(c *gin.Context) {
		defaultModelID, err := db.GetDefaultModelID(c.Request.Context())
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		// 防止删除当前正在使用的默认模型
		if defaultModelID != "" && defaultModelID == project.Slug(c.Param("id")) {
			writeHTTPError(c, http.StatusBadRequest, fmt.Errorf("model %q is the default model; change default first", c.Param("id")))
			return
		}
		if err := db.DeleteModelProfile(c.Request.Context(), c.Param("id")); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		logging.FromContext(c.Request.Context()).Info("model.pg.delete", zap.String("profile_id", project.Slug(c.Param("id"))))
		models.DeleteModelProfile(c.Request.Context(), c.Param("id"))
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	})

	// --- 全局设置接口 (默认模型) ---

	// 获取当前默认模型
	// 用户场景：前端在初始化时加载用户偏好的默认模型，以便在输入框旁边显示当前选中的模型。
	router.GET("/v1/settings/default-model", func(c *gin.Context) {
		modelID, err := models.GetDefaultModelID(c.Request.Context())
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		if modelID == "" {
			c.JSON(http.StatusOK, gin.H{"default_model_id": "", "model": nil})
			return
		}
		modelProfile, err := models.GetModelProfile(c.Request.Context(), modelID)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"default_model_id": modelID, "model": modelProfile})
	})

	// 设置默认模型
	// 业务逻辑：验证模型是否存在且处于活跃状态，然后将其 ID 存入设置表。
	// 用户场景：用户在设置页面切换默认使用的 AI 模型。
	router.PUT("/v1/settings/default-model", func(c *gin.Context) {
		var req defaultModelUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		modelProfile, err := models.GetModelProfile(c.Request.Context(), req.Model)
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		if modelProfile.Status != "" && modelProfile.Status != "active" {
			writeHTTPError(c, http.StatusBadRequest, fmt.Errorf("model profile %q is not active", modelProfile.ID))
			return
		}
		if err := models.SetDefaultModelID(c.Request.Context(), modelProfile.ID); err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"default_model_id": modelProfile.ID, "model": modelProfile})
	})

	// 清除默认模型设置
	// 用户场景：用户希望每次运行时都手动选择模型，不再使用默认值。
	router.DELETE("/v1/settings/default-model", func(c *gin.Context) {
		if err := models.ClearDefaultModelID(c.Request.Context()); err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	})

	// --- 项目文档接口 ---

	// 列出项目下的所有文档
	// 用户场景：查看项目中已上传的所有背景资料、章节草稿等。
	router.GET("/v1/projects/:id/documents", func(c *gin.Context) {
		docs, err := db.ListProjectDocuments(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"documents": docs})
	})

	// 上传或更新项目文档
	// 业务逻辑：根据 Kind（类型）和 ProjectID 唯一标识文档，存在则更新，不存在则创建。
	// 用户场景：保存“角色A的详细设定”或更新“第三章的剧情摘要”。
	router.PUT("/v1/projects/:id/documents/:kind", func(c *gin.Context) {
		var req projectDocumentUpsertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		kind := firstNonEmpty(req.Kind, c.Param("kind"))
		doc, err := projects.UpsertProjectDocument(c.Request.Context(), store.UpsertProjectDocumentParams{
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

	// --- 核心运行接口 ---

	// 执行 AI 代理运行
	// 这是系统最核心的业务逻辑入口。
	// 业务流程：
	// 1. 解析请求参数。
	// 2. 确定使用的模型（优先使用请求指定的，否则使用数据库默认模型）。
	// 3. 加载模型配置（API Key, URL 等）。
	// 4. 加载项目上下文（项目元数据 + 关联文档）。
	// 5. 初始化运行时环境。
	// 6. 执行干跑（Dry Run）或实际运行。
	// 7. 记录运行结果到数据库。
	// 用户场景：用户发送指令，AI 结合项目背景生成小说章节、代码或分析结果。
	router.POST("/v1/runs", func(c *gin.Context) {
		logger := logging.FromContext(c.Request.Context())
		var req runRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		logger.Info("run.request.accepted",
			zap.Bool("dry_run", req.DryRun),
			zap.Bool("explicit_project", strings.TrimSpace(req.Project) != ""),
			zap.Bool("explicit_model", strings.TrimSpace(req.Model) != ""),
			zap.Int("input_bytes", len(req.Input)),
		)

		// 1. 确定模型 ID：优先使用请求参数，其次使用数据库中设置的默认模型
		defaultModelID, err := models.GetDefaultModelID(c.Request.Context())
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		selectedModelID := firstNonEmpty(req.Model, defaultModelID)
		logger.Info("run.model.selected",
			zap.String("selected_profile_id", project.Slug(selectedModelID)),
			zap.String("default_profile_id", defaultModelID),
			zap.Bool("explicit_model", strings.TrimSpace(req.Model) != ""),
		)

		// 如果仍然没有模型 ID，报错
		if strings.TrimSpace(selectedModelID) == "" {
			writeHTTPError(c, http.StatusBadRequest, fmt.Errorf("model is required; pass /v1/runs.model or set a database default model first"))
			return
		}

		// 2. 加载模型配置：先读 Redis 缓存，未命中时回源 PostgreSQL 并回填缓存。
		activeModel, runtimeModel, err := loadRuntimeModelConfig(c.Request.Context(), models, selectedModelID)
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		logger.Info("run.model.loaded",
			zap.String("profile_id", activeModel.ID),
			zap.String("provider", activeModel.Provider),
			zap.String("model_id", activeModel.ModelID),
			zap.String("base_url", activeModel.BaseURL),
			zap.Int("timeout_seconds", runtimeModel.TimeoutSeconds),
		)

		// 3. 加载项目上下文：获取项目信息及关联文档，构建 Prompt 上下文
		projectID := strings.TrimSpace(req.Project)
		projectContext, activeProject, err := loadProjectContext(c.Request.Context(), projects, projectID)
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		projectLogFields := []zap.Field{
			zap.String("project_id", project.Slug(projectID)),
			zap.Int("context_bytes", len(projectContext)),
		}
		if activeProject != nil {
			projectLogFields = append(projectLogFields,
				zap.String("active_project_id", activeProject.ID),
				zap.String("storage_provider", activeProject.StorageProvider),
				zap.String("storage_bucket", activeProject.StorageBucket),
				zap.String("storage_prefix", activeProject.StoragePrefix),
			)
		}
		logger.Info("run.project_context.loaded", projectLogFields...)

		// 4. 初始化运行时实例
		rt, err := runtime.New(cfg, runtimeModel)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		rt.ProjectDocs = runtimeProjectDocumentProvider{Projects: projects}
		if activeProject != nil {
			rt.ProjectID = activeProject.ID
		}
		rt.ProjectContext = projectContext
		rt.Debug = debug
		if req.Debug != nil {
			rt.Debug = *req.Debug
		}

		// 5. 处理干跑模式 (Dry Run)
		// 业务逻辑：仅构建上下文和准备环境，不实际调用 LLM API，用于调试 Prompt。
		if req.DryRun {
			logger.Info("run.dry_run.start", zap.String("profile_id", activeModel.ID), zap.String("project_id", rt.ProjectID))
			if err := rt.DryRun(req.Input); err != nil {
				writeHTTPError(c, http.StatusInternalServerError, err)
				return
			}
			logger.Info("run.dry_run.completed", zap.String("run_id", rt.Store.RunID), zap.String("run_dir", rt.Store.Root))
			c.JSON(http.StatusOK, gin.H{
				"dry_run": true,
				"run_id":  rt.Store.RunID,
				"run_dir": rt.Store.Root,
				"project": activeProject,
				"model":   activeModel,
			})
			return
		}

		// 6. 执行实际运行
		// 在数据库中创建一条“进行中”的运行记录
		dbRun, err := db.CreateRun(c.Request.Context(), projectID, req.Input, nil)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		logger.Info("run.pg.create", zap.Int64("db_run_id", dbRun.ID), zap.String("project_id", project.Slug(projectID)))

		// 设置超时控制，防止 LLM 调用无限挂起
		timeout := time.Duration(runtimeModel.TimeoutSeconds+30) * time.Second
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// 调用 Runtime 执行 AI 任务
		res, err := rt.Run(ctx, req.Input)
		if err != nil {
			// 如果执行失败，更新数据库记录状态为“失败”并记录错误信息
			_, _ = db.FailRun(c.Request.Context(), dbRun.ID, err.Error())
			logger.Error("run.runtime.failed", zap.Int64("db_run_id", dbRun.ID), zap.Error(err))
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}

		// 如果执行成功，更新数据库记录状态为“完成”并保存结果路径
		finished, err := db.FinishRun(c.Request.Context(), dbRun.ID, res.FinalText, res.RunDir)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		logger.Info("run.completed",
			zap.Int64("db_run_id", finished.ID),
			zap.String("run_id", res.RunID),
			zap.String("run_dir", res.RunDir),
			zap.Int("final_text_bytes", len(res.FinalText)),
		)

		// 返回最终结果给前端
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

type modelConfigStore struct {
	DB    appStore
	Cache *cache.RedisConfigCache
}

type projectConfigStore struct {
	DB    appStore
	Cache *cache.RedisConfigCache
	Files *projectfs.Provider
}

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

func (s projectConfigStore) CreateProject(ctx context.Context, arg store.CreateProjectParams) (store.Project, error) {
	logger := logging.FromContext(ctx)
	p, err := s.DB.CreateProject(ctx, arg)
	if err != nil {
		return store.Project{}, err
	}
	logger.Info("project.pg.create",
		zap.String("project_id", p.ID),
		zap.String("storage_provider", p.StorageProvider),
		zap.String("storage_bucket", p.StorageBucket),
		zap.String("storage_prefix", p.StoragePrefix),
	)
	if err := s.syncFilesystemProject(ctx, p); err != nil {
		return store.Project{}, err
	}
	s.CacheProject(ctx, p)
	return p, nil
}

func (s projectConfigStore) GetProject(ctx context.Context, projectID string) (store.Project, error) {
	logger := logging.FromContext(ctx)
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return store.Project{}, fmt.Errorf("project is required")
	}
	if s.Cache != nil {
		p, err := s.Cache.GetProject(ctx, projectID)
		if err == nil {
			logger.Info("project.cache.hit", zap.String("project_id", p.ID))
			return p, nil
		}
		if !errors.Is(err, cache.ErrMiss) {
			logger.Warn("project.cache.error", zap.String("project_id", project.Slug(projectID)), zap.Error(err))
		} else {
			logger.Info("project.cache.miss", zap.String("project_id", project.Slug(projectID)))
		}
	} else {
		logger.Debug("project.cache.skip", zap.String("project_id", project.Slug(projectID)))
	}
	p, err := s.DB.GetProject(ctx, projectID)
	if err != nil {
		return store.Project{}, err
	}
	logger.Info("project.pg.get",
		zap.String("project_id", p.ID),
		zap.String("storage_provider", p.StorageProvider),
		zap.String("storage_bucket", p.StorageBucket),
		zap.String("storage_prefix", p.StoragePrefix),
	)
	if s.Cache != nil {
		if err := s.Cache.SetProject(ctx, p); err != nil {
			logger.Warn("project.cache.backfill.failed", zap.String("project_id", p.ID), zap.Error(err))
		} else {
			logger.Debug("project.cache.backfill.done", zap.String("project_id", p.ID))
		}
	}
	return p, nil
}

func (s projectConfigStore) ListProjects(ctx context.Context, limit, offset int32) ([]store.Project, error) {
	logger := logging.FromContext(ctx)
	items, err := s.DB.ListProjects(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	logger.Info("project.pg.list", zap.Int32("limit", limit), zap.Int32("offset", offset), zap.Int("count", len(items)))
	for _, item := range items {
		s.CacheProject(ctx, item)
	}
	return items, nil
}

func (s projectConfigStore) UpdateProject(ctx context.Context, arg store.UpdateProjectParams) (store.Project, error) {
	logger := logging.FromContext(ctx)
	previous, previousErr := s.DB.GetProject(ctx, arg.ID)
	p, err := s.DB.UpdateProject(ctx, arg)
	if err != nil {
		return store.Project{}, err
	}
	logger.Info("project.pg.update",
		zap.String("project_id", p.ID),
		zap.String("storage_provider", p.StorageProvider),
		zap.String("storage_bucket", p.StorageBucket),
		zap.String("storage_prefix", p.StoragePrefix),
	)
	if previousErr == nil {
		if err := s.Files.TryMoveProject(previous, p); err != nil {
			logger.Warn("project.fs.move.failed",
				zap.String("project_id", p.ID),
				zap.String("previous_storage_prefix", previous.StoragePrefix),
				zap.String("storage_prefix", p.StoragePrefix),
				zap.Error(err),
			)
		} else if previous.StoragePrefix != "" && previous.StoragePrefix != p.StoragePrefix {
			logger.Info("project.fs.move.done",
				zap.String("project_id", p.ID),
				zap.String("previous_storage_prefix", previous.StoragePrefix),
				zap.String("storage_prefix", p.StoragePrefix),
			)
		}
	}
	if err := s.syncFilesystemProject(ctx, p); err != nil {
		return store.Project{}, err
	}
	s.CacheProject(ctx, p)
	return p, nil
}

func (s projectConfigStore) DeleteProject(ctx context.Context, projectID string) error {
	previous, previousErr := s.DB.GetProject(ctx, projectID)
	var docs []store.ProjectDocument
	if previousErr == nil {
		if projectDocs, err := s.DB.ListProjectDocuments(ctx, projectID); err == nil {
			docs = projectDocs
		}
	}
	if err := s.DB.DeleteProject(ctx, projectID); err != nil {
		return err
	}
	logger := logging.FromContext(ctx)
	logger.Info("project.pg.delete", zap.String("project_id", project.Slug(projectID)))
	if previousErr == nil && s.Files != nil && strings.EqualFold(previous.StorageProvider, "filesystem") {
		previous.Status = "deleted"
		if err := s.Files.SyncProject(previous, docs); err != nil {
			logger.Warn("project.fs.meta.delete_sync.failed",
				zap.String("project_id", previous.ID),
				zap.String("storage_prefix", previous.StoragePrefix),
				zap.Error(err),
			)
		} else {
			logger.Info("project.fs.meta.delete_sync.done",
				zap.String("project_id", previous.ID),
				zap.String("storage_prefix", previous.StoragePrefix),
			)
		}
	}
	if s.Cache != nil {
		syncCacheAsync(ctx, "project.cache.delete", func(ctx context.Context) error {
			return s.Cache.DeleteProject(ctx, projectID)
		}, zap.String("project_id", project.Slug(projectID)))
	}
	return nil
}

func (s projectConfigStore) UpsertProjectDocument(ctx context.Context, arg store.UpsertProjectDocumentParams) (store.ProjectDocument, error) {
	doc, err := s.DB.UpsertProjectDocument(ctx, arg)
	if err != nil {
		return store.ProjectDocument{}, err
	}
	logger := logging.FromContext(ctx)
	logger.Info("project.document.pg.upsert",
		zap.String("project_id", project.Slug(arg.ProjectID)),
		zap.String("kind", doc.Kind),
		zap.Int("body_bytes", len(doc.Body)),
	)
	projectItem, err := s.GetProject(ctx, arg.ProjectID)
	if err != nil {
		return store.ProjectDocument{}, err
	}
	if err := s.syncFilesystemDocument(ctx, projectItem, doc); err != nil {
		return store.ProjectDocument{}, err
	}
	return doc, nil
}

func (s projectConfigStore) ListProjectDocuments(ctx context.Context, projectID string) ([]store.ProjectDocument, error) {
	docs, err := s.DB.ListProjectDocuments(ctx, projectID)
	if err != nil {
		return nil, err
	}
	logging.FromContext(ctx).Info("project.documents.pg.list",
		zap.String("project_id", project.Slug(projectID)),
		zap.Int("count", len(docs)),
	)
	return docs, nil
}

func (s projectConfigStore) CacheProject(ctx context.Context, p store.Project) {
	if s.Cache == nil {
		return
	}
	syncCacheAsync(ctx, "project.cache.set", func(ctx context.Context) error {
		return s.Cache.SetProject(ctx, p)
	}, zap.String("project_id", p.ID))
}

func (s projectConfigStore) syncFilesystemProject(ctx context.Context, p store.Project) error {
	if s.Files == nil || !strings.EqualFold(strings.TrimSpace(p.StorageProvider), "filesystem") {
		return nil
	}
	docs, err := s.DB.ListProjectDocuments(ctx, p.ID)
	if err != nil {
		return err
	}
	if err := s.Files.SyncProject(p, docs); err != nil {
		return err
	}
	logging.FromContext(ctx).Info("project.fs.meta.synced",
		zap.String("project_id", p.ID),
		zap.String("storage_prefix", p.StoragePrefix),
		zap.Int("document_count", len(docs)),
	)
	return nil
}

func (s projectConfigStore) syncFilesystemDocument(ctx context.Context, p store.Project, doc store.ProjectDocument) error {
	if s.Files == nil || !strings.EqualFold(strings.TrimSpace(p.StorageProvider), "filesystem") {
		return nil
	}
	if err := s.Files.SyncDocument(p, doc); err != nil {
		return err
	}
	logging.FromContext(ctx).Info("project.fs.document.synced",
		zap.String("project_id", p.ID),
		zap.String("kind", doc.Kind),
		zap.String("storage_prefix", p.StoragePrefix),
	)
	return s.syncFilesystemProject(ctx, p)
}

func (s modelConfigStore) GetDefaultModelID(ctx context.Context) (string, error) {
	logger := logging.FromContext(ctx)
	if s.Cache != nil {
		modelID, err := s.Cache.GetDefaultModelID(ctx)
		if err == nil {
			logger.Info("default_model.cache.hit", zap.String("model_id", modelID))
			return modelID, nil
		}
		if !errors.Is(err, cache.ErrMiss) {
			logger.Warn("default_model.cache.error", zap.Error(err))
		} else {
			logger.Info("default_model.cache.miss")
		}
	} else {
		logger.Debug("default_model.cache.skip")
	}
	modelID, err := s.DB.GetDefaultModelID(ctx)
	if err != nil {
		return "", err
	}
	logger.Info("default_model.pg.get", zap.String("model_id", modelID))
	if s.Cache != nil {
		if err := s.Cache.SetDefaultModelID(ctx, modelID); err != nil {
			logger.Warn("default_model.cache.backfill.failed", zap.Error(err))
		} else {
			logger.Debug("default_model.cache.backfill.done", zap.String("model_id", modelID))
		}
	}
	return modelID, nil
}

func (s modelConfigStore) SetDefaultModelID(ctx context.Context, modelID string) error {
	if err := s.DB.SetDefaultModelID(ctx, modelID); err != nil {
		return err
	}
	logging.FromContext(ctx).Info("default_model.pg.set", zap.String("model_id", project.Slug(modelID)))
	if s.Cache != nil {
		syncCacheAsync(ctx, "default_model.cache.set", func(ctx context.Context) error {
			return s.Cache.SetDefaultModelID(ctx, project.Slug(modelID))
		}, zap.String("model_id", project.Slug(modelID)))
	}
	return nil
}

func (s modelConfigStore) ClearDefaultModelID(ctx context.Context) error {
	if err := s.DB.ClearDefaultModelID(ctx); err != nil {
		return err
	}
	logging.FromContext(ctx).Info("default_model.pg.clear")
	if s.Cache != nil {
		syncCacheAsync(ctx, "default_model.cache.delete", func(ctx context.Context) error {
			return s.Cache.DeleteDefaultModelID(ctx)
		})
	}
	return nil
}

func (s modelConfigStore) GetModelProfile(ctx context.Context, modelID string) (store.ModelProfile, error) {
	logger := logging.FromContext(ctx)
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return store.ModelProfile{}, fmt.Errorf("model is required")
	}
	if s.Cache != nil {
		profile, err := s.Cache.GetModelProfile(ctx, modelID)
		if err == nil {
			logger.Info("model.cache.hit", zap.String("profile_id", profile.ID), zap.String("model_id", profile.ModelID))
			return profile, nil
		}
		if !errors.Is(err, cache.ErrMiss) {
			logger.Warn("model.cache.error", zap.String("profile_id", project.Slug(modelID)), zap.Error(err))
		} else {
			logger.Info("model.cache.miss", zap.String("profile_id", project.Slug(modelID)))
		}
	} else {
		logger.Debug("model.cache.skip", zap.String("profile_id", project.Slug(modelID)))
	}
	profile, err := s.DB.GetModelProfile(ctx, modelID)
	if err != nil {
		return store.ModelProfile{}, err
	}
	logger.Info("model.pg.get", zap.String("profile_id", profile.ID), zap.String("model_id", profile.ModelID), zap.String("provider", profile.Provider))
	if s.Cache != nil {
		if err := s.Cache.SetModelProfile(ctx, profile); err != nil {
			logger.Warn("model.cache.backfill.failed", zap.String("profile_id", profile.ID), zap.Error(err))
		} else {
			logger.Debug("model.cache.backfill.done", zap.String("profile_id", profile.ID))
		}
	}
	return profile, nil
}

func (s modelConfigStore) CacheModelProfile(ctx context.Context, profile store.ModelProfile) {
	if s.Cache == nil {
		return
	}
	syncCacheAsync(ctx, "model.cache.set", func(ctx context.Context) error {
		return s.Cache.SetModelProfile(ctx, profile)
	}, zap.String("profile_id", profile.ID), zap.String("model_id", profile.ModelID))
}

func (s modelConfigStore) DeleteModelProfile(ctx context.Context, modelID string) {
	if s.Cache == nil {
		return
	}
	syncCacheAsync(ctx, "model.cache.delete", func(ctx context.Context) error {
		return s.Cache.DeleteModelProfile(ctx, modelID)
	}, zap.String("profile_id", project.Slug(modelID)))
}

func syncCacheAsync(parent context.Context, op string, fn func(context.Context) error, fields ...zap.Field) {
	logger := logging.FromContext(parent).With(fields...)
	go func() {
		baseCtx := logging.WithLogger(context.Background(), logger)
		ctx, cancel := context.WithTimeout(baseCtx, 5*time.Second)
		defer cancel()
		logger.Debug("cache.sync.start", zap.String("op", op))
		if err := fn(ctx); err != nil {
			logger.Warn("cache.sync.failed", zap.String("op", op), zap.Error(err))
			return
		}
		logger.Debug("cache.sync.done", zap.String("op", op))
	}()
}

func loadRuntimeModelConfig(ctx context.Context, models modelConfigStore, modelID string) (*store.ModelProfile, runtime.ModelConfig, error) {
	profile, err := models.GetModelProfile(ctx, modelID)
	if err != nil {
		return nil, runtime.ModelConfig{}, err
	}
	if profile.Status != "" && profile.Status != "active" {
		return nil, runtime.ModelConfig{}, fmt.Errorf("model profile %q is not active", profile.ID)
	}
	modelCfg := runtime.ModelConfig{
		Provider:       profile.Provider,
		ID:             profile.ModelID,
		BaseURL:        profile.BaseURL,
		APIKey:         profile.APIKey,
		APIKeyEnv:      profile.APIKeyEnv,
		ContextWindow:  profile.ContextWindow,
		MaxOutput:      profile.MaxOutputTokens,
		Temperature:    profile.Temperature,
		TimeoutSeconds: profile.TimeoutSeconds,
	}
	return &profile, modelCfg, nil
}

// openConfiguredStore 初始化数据库连接并执行自动迁移。
func openConfiguredStore(ctx context.Context, cfg config.Config) (*store.Store, error) {
	logger := logging.FromContext(ctx)
	databaseURL, err := cfg.Database.ConnectionString()
	if err != nil {
		return nil, err
	}
	logger.Info("database.open.start",
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("database", cfg.Database.Name),
		zap.String("user", cfg.Database.User),
		zap.Bool("auto_migrate", cfg.Database.AutoMigrate),
	)
	if cfg.Database.AutoMigrate {
		logger.Info("database.migrate.start", zap.String("migrations_dir", cfg.Database.MigrationsDir))
		if err := store.Migrate(databaseURL, cfg.Database.MigrationsDir); err != nil {
			return nil, fmt.Errorf("migrate database: %w", err)
		}
		logger.Info("database.migrate.done", zap.String("migrations_dir", cfg.Database.MigrationsDir))
	}
	db, err := store.Open(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	logger.Info("database.open.done")
	return db, nil
}

// loadProjectContext 通过 ContextProvider 构建项目上下文，HTTP 层不直接拼装项目文档。
func loadProjectContext(ctx context.Context, db workflow.ProjectStore, projectID string) (string, *store.Project, error) {
	provider := workflow.PostgresContextProvider{Store: db}
	pack, err := provider.BuildContext(ctx, projectID, "")
	if err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(pack.Project.ID) == "" {
		return pack.Text, nil, nil
	}
	return pack.Text, &store.Project{
		ID:              pack.Project.ID,
		Name:            pack.Project.Name,
		Description:     pack.Project.Description,
		Status:          pack.Project.Status,
		StorageProvider: pack.Project.StorageProvider,
		StorageBucket:   pack.Project.StorageBucket,
		StoragePrefix:   pack.Project.StoragePrefix,
	}, nil
}

// pagination 从 HTTP 请求中提取分页参数。
// 默认限制为 50 条，偏移量为 0。
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

// writeHTTPError 统一处理 HTTP 错误响应。
// 它将错误信息格式化为 JSON 返回给客户端。
func requestLoggerMiddleware(base *zap.Logger) gin.HandlerFunc {
	if base == nil {
		base = zap.NewNop()
	}
	return func(c *gin.Context) {
		start := time.Now()
		requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if requestID == "" {
			requestID = logging.NewRequestID()
		}
		c.Writer.Header().Set("X-Request-ID", requestID)

		reqLogger := base.With(
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
		)
		c.Request = c.Request.WithContext(logging.WithLogger(c.Request.Context(), reqLogger))
		reqLogger.Info("http.request.start",
			zap.String("client_ip", c.ClientIP()),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("user_agent", c.Request.UserAgent()),
		)

		c.Next()

		fields := []zap.Field{
			zap.Int("status", c.Writer.Status()),
			zap.String("route", c.FullPath()),
			zap.Duration("latency", time.Since(start)),
			zap.Int("response_bytes", c.Writer.Size()),
		}
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("gin_errors", c.Errors.String()))
		}
		switch {
		case c.Writer.Status() >= http.StatusInternalServerError:
			reqLogger.Error("http.request.completed", fields...)
		case c.Writer.Status() >= http.StatusBadRequest:
			reqLogger.Warn("http.request.completed", fields...)
		default:
			reqLogger.Info("http.request.completed", fields...)
		}
	}
}

func writeHTTPError(c *gin.Context, status int, err error) {
	msg := "request failed"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		msg = err.Error()
	}
	logging.FromContext(c.Request.Context()).Warn("http.error",
		zap.Int("status", status),
		zap.String("error", msg),
	)
	c.JSON(status, gin.H{"error": msg})
}
