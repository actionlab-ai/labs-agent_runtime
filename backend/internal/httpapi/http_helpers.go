package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"novel-agent-runtime/internal/config"
	"novel-agent-runtime/internal/logging"
	"novel-agent-runtime/internal/runtime"
	"novel-agent-runtime/internal/store"
	"novel-agent-runtime/internal/workflow"
)

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

func errRequiredModel() error {
	return fmt.Errorf("model is required; pass /v1/runs.model or set a database default model first")
}
