CREATE TABLE IF NOT EXISTS model_profiles (
  id text PRIMARY KEY,
  name text NOT NULL,
  provider text NOT NULL DEFAULT 'openai_compatible',
  model_id text NOT NULL,
  base_url text NOT NULL,
  api_key_env text NOT NULL DEFAULT '',
  context_window integer NOT NULL DEFAULT 131072,
  max_output_tokens integer NOT NULL DEFAULT 4096,
  temperature double precision NOT NULL DEFAULT 0.7,
  timeout_seconds integer NOT NULL DEFAULT 180,
  status text NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

DROP TRIGGER IF EXISTS model_profiles_set_updated_at ON model_profiles;
CREATE TRIGGER model_profiles_set_updated_at
  BEFORE UPDATE ON model_profiles
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
