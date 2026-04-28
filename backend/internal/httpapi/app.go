package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"novel-agent-runtime/internal/cache"
	"novel-agent-runtime/internal/config"
	"novel-agent-runtime/internal/logging"
	"novel-agent-runtime/internal/projectfs"
)

type routeDeps struct {
	cfg      config.Config
	debug    bool
	db       appStore
	models   modelConfigStore
	projects projectConfigStore
}

func Serve(cfg config.Config, addr string, debug bool) error {
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
	deps := routeDeps{
		cfg:   cfg,
		debug: debug,
		db:    db,
		models: modelConfigStore{
			DB:    db,
			Cache: configCache,
		},
		projects: projectConfigStore{
			DB:    db,
			Cache: configCache,
			Files: projectFiles,
		},
	}

	registerUtilityRoutes(router, deps)
	registerWorkflowRoutes(router, cfg, debug, db, deps.models, deps.projects)
	registerProjectRoutes(router, deps)
	registerModelRoutes(router, deps)
	registerSettingRoutes(router, deps)
	registerProjectDocumentRoutes(router, deps)
	registerRunRoutes(router, deps)

	return router
}
