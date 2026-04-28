-- name: GetAppSetting :one
-- 根据指定的键（key）获取单个应用配置项。
-- 返回配置的键、值以及创建和更新时间。
SELECT key, value, created_at, updated_at
FROM app_settings
WHERE key = $1;

-- name: UpsertAppSetting :one
-- 插入或更新应用配置项。
-- 如果指定的 key 已存在，则更新其 value；否则插入新记录。
-- 返回操作后的完整配置记录（包括键、值及时间戳）。
INSERT INTO app_settings (key, value)
VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value
RETURNING key, value, created_at, updated_at;

-- name: DeleteAppSetting :exec
-- 根据指定的键（key）删除应用配置项。
-- 此操作不返回任何数据，仅执行删除动作。
DELETE FROM app_settings
WHERE key = $1;