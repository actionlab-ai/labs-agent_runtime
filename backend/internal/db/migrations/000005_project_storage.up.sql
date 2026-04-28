ALTER TABLE projects
  ADD COLUMN IF NOT EXISTS storage_provider text NOT NULL DEFAULT 'filesystem',
  ADD COLUMN IF NOT EXISTS storage_bucket text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS storage_prefix text NOT NULL DEFAULT '';
