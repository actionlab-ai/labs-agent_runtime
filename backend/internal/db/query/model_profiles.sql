-- name: CreateModelProfile :one
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
SELECT id, name, provider, model_id, base_url, api_key_env, context_window, max_output_tokens, temperature, timeout_seconds, status, metadata, created_at, updated_at, deleted_at, api_key
FROM model_profiles
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListModelProfiles :many
SELECT id, name, provider, model_id, base_url, api_key_env, context_window, max_output_tokens, temperature, timeout_seconds, status, metadata, created_at, updated_at, deleted_at, api_key
FROM model_profiles
WHERE deleted_at IS NULL
ORDER BY updated_at DESC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateModelProfile :one
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
UPDATE model_profiles
SET deleted_at = now(), status = 'deleted'
WHERE id = $1 AND deleted_at IS NULL;
