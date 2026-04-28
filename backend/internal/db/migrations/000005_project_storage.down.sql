ALTER TABLE projects
  DROP COLUMN IF EXISTS storage_prefix,
  DROP COLUMN IF EXISTS storage_bucket,
  DROP COLUMN IF EXISTS storage_provider;
