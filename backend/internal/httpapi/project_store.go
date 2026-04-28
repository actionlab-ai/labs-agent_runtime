package httpapi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"novel-agent-runtime/internal/cache"
	"novel-agent-runtime/internal/logging"
	"novel-agent-runtime/internal/project"
	"novel-agent-runtime/internal/store"
)

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
	logger.Info("project.documents.pg.upsert",
		zap.String("project_id", doc.ProjectID),
		zap.String("kind", doc.Kind),
		zap.Int("body_bytes", len(doc.Body)),
	)
	if p, err := s.DB.GetProject(ctx, arg.ProjectID); err == nil {
		if err := s.syncFilesystemDocument(ctx, p, doc); err != nil {
			return store.ProjectDocument{}, err
		}
		s.CacheProject(ctx, p)
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
