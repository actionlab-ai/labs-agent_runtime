package httpapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"novel-agent-runtime/internal/cache"
	"novel-agent-runtime/internal/logging"
	"novel-agent-runtime/internal/project"
	"novel-agent-runtime/internal/store"
)

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
