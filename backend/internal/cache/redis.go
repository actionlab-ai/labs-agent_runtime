package cache

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"novel-agent-runtime/internal/config"
	"novel-agent-runtime/internal/project"
	"novel-agent-runtime/internal/store"
)

var ErrMiss = errors.New("cache miss")

type RedisConfigCache struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

type modelProfileRecord struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Provider        string          `json:"provider"`
	ModelID         string          `json:"model_id"`
	BaseURL         string          `json:"base_url"`
	APIKey          string          `json:"api_key"`
	APIKeySet       bool            `json:"api_key_set"`
	APIKeyEnv       string          `json:"api_key_env"`
	ContextWindow   int             `json:"context_window"`
	MaxOutputTokens int             `json:"max_output_tokens"`
	Temperature     float64         `json:"temperature"`
	TimeoutSeconds  int             `json:"timeout_seconds"`
	Status          string          `json:"status"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func OpenRedisConfigCache(ctx context.Context, cfg config.RedisConfig) (*RedisConfigCache, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	client := openRedisClient(cfg)
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}
	prefix := strings.Trim(strings.TrimSpace(cfg.KeyPrefix), ":")
	if prefix == "" {
		prefix = "novelrt"
	}
	ttl := time.Duration(cfg.TTLSeconds) * time.Second
	return &RedisConfigCache{client: client, prefix: prefix, ttl: ttl}, nil
}

func openRedisClient(cfg config.RedisConfig) redis.UniversalClient {
	if cfg.Mode == "standalone" {
		return redis.NewClient(&redis.Options{
			Addr:     cfg.Addrs[0],
			Password: cfg.Password,
		})
	}
	return redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    cfg.Addrs,
		Password: cfg.Password,
	})
}

func (c *RedisConfigCache) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *RedisConfigCache) GetProject(ctx context.Context, id string) (store.Project, error) {
	if c == nil {
		return store.Project{}, ErrMiss
	}
	raw, err := c.client.Get(ctx, c.key("project", project.Slug(id))).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return store.Project{}, ErrMiss
		}
		return store.Project{}, err
	}
	var p store.Project
	if err := json.Unmarshal(raw, &p); err != nil {
		return store.Project{}, err
	}
	return p, nil
}

func (c *RedisConfigCache) SetProject(ctx context.Context, p store.Project) error {
	if c == nil {
		return nil
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key("project", p.ID), raw, c.ttl).Err()
}

func (c *RedisConfigCache) DeleteProject(ctx context.Context, id string) error {
	if c == nil {
		return nil
	}
	return c.client.Del(ctx, c.key("project", project.Slug(id))).Err()
}

func (c *RedisConfigCache) GetModelProfile(ctx context.Context, id string) (store.ModelProfile, error) {
	if c == nil {
		return store.ModelProfile{}, ErrMiss
	}
	raw, err := c.client.Get(ctx, c.key("model", id)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return store.ModelProfile{}, ErrMiss
		}
		return store.ModelProfile{}, err
	}
	var rec modelProfileRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return store.ModelProfile{}, err
	}
	return rec.toModelProfile(), nil
}

func (c *RedisConfigCache) SetModelProfile(ctx context.Context, profile store.ModelProfile) error {
	if c == nil {
		return nil
	}
	raw, err := json.Marshal(modelProfileRecordFrom(profile))
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key("model", profile.ID), raw, c.ttl).Err()
}

func (c *RedisConfigCache) DeleteModelProfile(ctx context.Context, id string) error {
	if c == nil {
		return nil
	}
	return c.client.Del(ctx, c.key("model", id)).Err()
}

func (c *RedisConfigCache) GetDefaultModelID(ctx context.Context) (string, error) {
	if c == nil {
		return "", ErrMiss
	}
	v, err := c.client.Get(ctx, c.key("setting", "default_model_id")).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrMiss
		}
		return "", err
	}
	return strings.TrimSpace(v), nil
}

func (c *RedisConfigCache) SetDefaultModelID(ctx context.Context, modelID string) error {
	if c == nil {
		return nil
	}
	return c.client.Set(ctx, c.key("setting", "default_model_id"), strings.TrimSpace(modelID), c.ttl).Err()
}

func (c *RedisConfigCache) DeleteDefaultModelID(ctx context.Context) error {
	if c == nil {
		return nil
	}
	return c.client.Del(ctx, c.key("setting", "default_model_id")).Err()
}

func (c *RedisConfigCache) key(parts ...string) string {
	clean := make([]string, 0, len(parts)+1)
	clean = append(clean, c.prefix)
	for _, part := range parts {
		part = strings.Trim(strings.TrimSpace(part), ":")
		if part != "" {
			clean = append(clean, part)
		}
	}
	return strings.Join(clean, ":")
}

func modelProfileRecordFrom(p store.ModelProfile) modelProfileRecord {
	return modelProfileRecord{
		ID:              p.ID,
		Name:            p.Name,
		Provider:        p.Provider,
		ModelID:         p.ModelID,
		BaseURL:         p.BaseURL,
		APIKey:          p.APIKey,
		APIKeySet:       p.APIKeySet,
		APIKeyEnv:       p.APIKeyEnv,
		ContextWindow:   p.ContextWindow,
		MaxOutputTokens: p.MaxOutputTokens,
		Temperature:     p.Temperature,
		TimeoutSeconds:  p.TimeoutSeconds,
		Status:          p.Status,
		Metadata:        p.Metadata,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

func (r modelProfileRecord) toModelProfile() store.ModelProfile {
	return store.ModelProfile{
		ID:              r.ID,
		Name:            r.Name,
		Provider:        r.Provider,
		ModelID:         r.ModelID,
		BaseURL:         r.BaseURL,
		APIKey:          r.APIKey,
		APIKeySet:       r.APIKeySet,
		APIKeyEnv:       r.APIKeyEnv,
		ContextWindow:   r.ContextWindow,
		MaxOutputTokens: r.MaxOutputTokens,
		Temperature:     r.Temperature,
		TimeoutSeconds:  r.TimeoutSeconds,
		Status:          r.Status,
		Metadata:        r.Metadata,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}
