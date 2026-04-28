-- name: CreateProject :one
INSERT INTO projects (id, name, description, storage_provider, storage_bucket, storage_prefix, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    storage_provider = EXCLUDED.storage_provider,
    storage_bucket = EXCLUDED.storage_bucket,
    storage_prefix = EXCLUDED.storage_prefix,
    metadata = projects.metadata || EXCLUDED.metadata,
    deleted_at = NULL
RETURNING id, name, description, status, storage_provider, storage_bucket, storage_prefix, metadata, created_at, updated_at, deleted_at;

-- name: GetProject :one
SELECT id, name, description, status, storage_provider, storage_bucket, storage_prefix, metadata, created_at, updated_at, deleted_at
FROM projects
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListProjects :many
SELECT id, name, description, status, storage_provider, storage_bucket, storage_prefix, metadata, created_at, updated_at, deleted_at
FROM projects
WHERE deleted_at IS NULL
ORDER BY updated_at DESC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateProject :one
UPDATE projects
SET name = $2,
    description = $3,
    status = $4,
    storage_provider = $5,
    storage_bucket = $6,
    storage_prefix = $7,
    metadata = $8
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, name, description, status, storage_provider, storage_bucket, storage_prefix, metadata, created_at, updated_at, deleted_at;

-- name: DeleteProject :exec
UPDATE projects
SET deleted_at = now(), status = 'deleted'
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpsertProjectDocument :one
INSERT INTO project_documents (project_id, kind, title, body, metadata)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (project_id, kind) WHERE deleted_at IS NULL DO UPDATE
SET title = EXCLUDED.title,
    body = EXCLUDED.body,
    metadata = project_documents.metadata || EXCLUDED.metadata
RETURNING id, project_id, kind, title, body, metadata, created_at, updated_at, deleted_at;

-- name: ListProjectDocuments :many
SELECT id, project_id, kind, title, body, metadata, created_at, updated_at, deleted_at
FROM project_documents
WHERE project_id = $1 AND deleted_at IS NULL
ORDER BY
  CASE kind
    WHEN 'project_brief' THEN 1
    WHEN 'reader_contract' THEN 2
    WHEN 'style_guide' THEN 3
    WHEN 'taboo' THEN 4
    WHEN 'world_rules' THEN 5
    WHEN 'power_system' THEN 6
    WHEN 'factions' THEN 7
    WHEN 'locations' THEN 8
    WHEN 'mainline' THEN 9
    WHEN 'current_state' THEN 10
    ELSE 100
  END,
  updated_at DESC;
