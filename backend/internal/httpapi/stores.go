package httpapi

import (
	"novel-agent-runtime/internal/cache"
	"novel-agent-runtime/internal/projectfs"
)

type modelConfigStore struct {
	DB    appStore
	Cache *cache.RedisConfigCache
}

type projectConfigStore struct {
	DB    appStore
	Cache *cache.RedisConfigCache
	Files *projectfs.Provider
}
