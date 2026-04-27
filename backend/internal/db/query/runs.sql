-- name: CreateRun :one
INSERT INTO runs (project_id, session_id, input, status, metadata)
VALUES ($1, $2, $3, 'running', $4)
RETURNING id, project_id, session_id, input, final_text, run_dir, status, error, metadata, created_at, updated_at;

-- name: FinishRun :one
UPDATE runs
SET final_text = $2,
    run_dir = $3,
    status = 'completed',
    error = ''
WHERE id = $1
RETURNING id, project_id, session_id, input, final_text, run_dir, status, error, metadata, created_at, updated_at;

-- name: FailRun :one
UPDATE runs
SET status = 'failed',
    error = $2
WHERE id = $1
RETURNING id, project_id, session_id, input, final_text, run_dir, status, error, metadata, created_at, updated_at;
