ALTER TABLE model_profiles
  ADD COLUMN IF NOT EXISTS api_key text NOT NULL DEFAULT '';
