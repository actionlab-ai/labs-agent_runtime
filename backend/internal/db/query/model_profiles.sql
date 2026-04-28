-- name: CreateModelProfile :one
-- 创建或更新模型配置档案（Model Profile）。
-- 如果指定的 ID 已存在，则执行更新操作：
-- 1. 更新所有基本字段（名称、提供商、模型ID等）。
-- 2. 合并 metadata 字段（保留原有元数据并添加新元数据）。
-- 3. 重置 deleted_at 为 NULL 并将状态设为 'active'（实现软删除恢复/重新激活逻辑）。
-- 返回完整的模型配置记录。
INSERT INTO model_profiles (
  id,
  name,
  provider,
  model_id,
  base_url,
  api_key,
  api_key_env,
  context_window,
  max_output_tokens,
  temperature,
  timeout_seconds,
  metadata
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    provider = EXCLUDED.provider,
    model_id = EXCLUDED.model_id,
    base_url = EXCLUDED.base_url,
    api_key = EXCLUDED.api_key,
    api_key_env = EXCLUDED.api_key_env,
    context_window = EXCLUDED.context_window,
    max_output_tokens = EXCLUDED.max_output_tokens,
    temperature = EXCLUDED.temperature,
    timeout_seconds = EXCLUDED.timeout_seconds,
    metadata = model_profiles.metadata || EXCLUDED.metadata,
    deleted_at = NULL,
    status = 'active'
RETURNING id, name, provider, model_id, base_url, api_key_env, context_window, max_output_tokens, temperature, timeout_seconds, status, metadata, created_at, updated_at, deleted_at, api_key;

-- name: GetModelProfile :one
-- 根据 ID 获取单个未删除的模型配置档案。
-- 仅返回 deleted_at 为 NULL 的有效记录。
SELECT id, name, provider, model_id, base_url, api_key_env, context_window, max_output_tokens, temperature, timeout_seconds, status, metadata, created_at, updated_at, deleted_at, api_key
FROM model_profiles
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListModelProfiles :many
-- 分页获取所有未删除的模型配置档案列表。
-- 按更新时间降序排列，优先显示最近修改的配置。
-- 支持通过 LIMIT 和 OFFSET 进行分页控制。
SELECT id, name, provider, model_id, base_url, api_key_env, context_window, max_output_tokens, temperature, timeout_seconds, status, metadata, created_at, updated_at, deleted_at, api_key
FROM model_profiles
WHERE deleted_at IS NULL
ORDER BY updated_at DESC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateModelProfile :one
-- 更新指定 ID 的模型配置档案。
-- 允许修改名称、提供商、模型参数、API密钥及状态等字段。
-- 仅当记录未被软删除（deleted_at IS NULL）时执行更新。
-- 返回更新后的完整记录。
UPDATE model_profiles
SET name = $2,
    provider = $3,
    model_id = $4,
    base_url = $5,
    api_key = $6,
    api_key_env = $7,
    context_window = $8,
    max_output_tokens = $9,
    temperature = $10,
    timeout_seconds = $11,
    status = $12,
    metadata = $13
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, name, provider, model_id, base_url, api_key_env, context_window, max_output_tokens, temperature, timeout_seconds, status, metadata, created_at, updated_at, deleted_at, api_key;

-- name: DeleteModelProfile :exec
-- 软删除指定 ID 的模型配置档案。
-- 将 deleted_at 设置为当前时间，并将状态更新为 'deleted'。
-- 仅对未删除的记录生效，确保物理数据保留但逻辑上不可见。
UPDATE model_profiles
SET deleted_at = now(), status = 'deleted'
WHERE id = $1 AND deleted_at IS NULL;