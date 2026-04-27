package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Path     string
	Root     string
	Model    RuntimeModel   `mapstructure:"model"`
	Runtime  RuntimeConfig  `mapstructure:"runtime"`
	Database DatabaseConfig `mapstructure:"database"`
}

type RuntimeModel struct {
	Provider       string  `mapstructure:"provider"`
	ID             string  `mapstructure:"id"`
	BaseURL        string  `mapstructure:"base_url"`
	APIKey         string  `mapstructure:"api_key"`
	APIKeyEnv      string  `mapstructure:"api_key_env"`
	ContextWindow  int     `mapstructure:"context_window"`
	MaxOutput      int     `mapstructure:"max_output_tokens"`
	Temperature    float64 `mapstructure:"temperature"`
	TimeoutSeconds int     `mapstructure:"timeout_seconds"`
}

type RuntimeConfig struct {
	SkillsDir            string  `mapstructure:"skills_dir"`
	RunsDir              string  `mapstructure:"runs_dir"`
	WorkspaceRoot        string  `mapstructure:"workspace_root"`
	DocumentOutputDir    string  `mapstructure:"document_output_dir"`
	ProjectID            string  `mapstructure:"project_id"`
	ProjectRoot          string  `mapstructure:"project_root"`
	MaxToolRounds        int     `mapstructure:"max_tool_rounds"`
	MaxSkillToolRounds   int     `mapstructure:"max_skill_tool_rounds"`
	ForceToolSearchFirst bool    `mapstructure:"force_tool_search_first"`
	ReturnSkillDirect    bool    `mapstructure:"return_skill_output_direct"`
	FallbackSkillSearch  bool    `mapstructure:"fallback_skill_search"`
	FallbackMinScore     float64 `mapstructure:"fallback_min_score"`
	MaxActivatedSkills   int     `mapstructure:"max_activated_skills"`
	MaxRetainedSkills    int     `mapstructure:"max_retained_skills"`
	ActivationMinScore   float64 `mapstructure:"activation_min_score"`
	ActivationScoreRatio float64 `mapstructure:"activation_score_ratio"`
}

type DatabaseConfig struct {
	URL           string `mapstructure:"url"`
	MigrationsDir string `mapstructure:"migrations_dir"`
	AutoMigrate   bool   `mapstructure:"auto_migrate"`
}

func Load(path string) (Config, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Config{}, fmt.Errorf("resolve config path: %w", err)
	}
	root := filepath.Dir(abs)

	v := viper.New()
	v.SetConfigFile(abs)
	v.SetConfigType("yaml")

	setDefaults(v)
	bindEnv(v)

	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", abs, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config %s: %w", abs, err)
	}

	cfg.Path = abs
	cfg.Root = root
	cfg.Model.Provider = firstNonEmpty(cfg.Model.Provider, "openai_compatible")
	cfg.Model.APIKeyEnv = firstNonEmpty(cfg.Model.APIKeyEnv, "NOVEL_MODEL_API_KEY")
	cfg.Runtime.SkillsDir = resolve(root, cfg.Runtime.SkillsDir)
	cfg.Runtime.RunsDir = resolve(root, cfg.Runtime.RunsDir)
	cfg.Runtime.WorkspaceRoot = resolve(root, firstNonEmpty(cfg.Runtime.WorkspaceRoot, ".."))
	cfg.Runtime.DocumentOutputDir = resolve(root, firstNonEmpty(cfg.Runtime.DocumentOutputDir, "../docs/08-generated-drafts"))
	if strings.TrimSpace(cfg.Runtime.ProjectRoot) != "" {
		cfg.Runtime.ProjectRoot = resolve(root, cfg.Runtime.ProjectRoot)
	}
	cfg.Database.MigrationsDir = resolve(root, firstNonEmpty(cfg.Database.MigrationsDir, "./internal/db/migrations"))

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("model.provider", "openai_compatible")
	v.SetDefault("model.api_key_env", "NOVEL_MODEL_API_KEY")
	v.SetDefault("model.api_key", "")
	v.SetDefault("model.context_window", 131072)
	v.SetDefault("model.max_output_tokens", 4096)
	v.SetDefault("model.temperature", 0.7)
	v.SetDefault("model.timeout_seconds", 180)

	v.SetDefault("runtime.skills_dir", "./skills")
	v.SetDefault("runtime.runs_dir", "./runs")
	v.SetDefault("runtime.workspace_root", "..")
	v.SetDefault("runtime.document_output_dir", "../docs/08-generated-drafts")
	v.SetDefault("runtime.project_id", "")
	v.SetDefault("runtime.project_root", "")
	v.SetDefault("runtime.max_tool_rounds", 4)
	v.SetDefault("runtime.max_skill_tool_rounds", 6)
	v.SetDefault("runtime.force_tool_search_first", true)
	v.SetDefault("runtime.return_skill_output_direct", true)
	v.SetDefault("runtime.fallback_skill_search", true)
	v.SetDefault("runtime.fallback_min_score", 0.18)
	v.SetDefault("runtime.max_activated_skills", 3)
	v.SetDefault("runtime.max_retained_skills", 6)
	v.SetDefault("runtime.activation_min_score", 0.18)
	v.SetDefault("runtime.activation_score_ratio", 0.55)
	v.SetDefault("database.url", "")
	v.SetDefault("database.migrations_dir", "./internal/db/migrations")
	v.SetDefault("database.auto_migrate", true)
}

func bindEnv(v *viper.Viper) {
	v.SetEnvPrefix("NOVEL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	_ = v.BindEnv("database.url", "DATABASE_URL", "NOVEL_DATABASE_URL")

	// Explicit binds make supported overrides discoverable and stable.
	for _, key := range []string{
		"model.provider",
		"model.id",
		"model.base_url",
		"model.api_key",
		"model.api_key_env",
		"model.context_window",
		"model.max_output_tokens",
		"model.temperature",
		"model.timeout_seconds",
		"runtime.skills_dir",
		"runtime.runs_dir",
		"runtime.workspace_root",
		"runtime.document_output_dir",
		"runtime.project_id",
		"runtime.project_root",
		"runtime.max_tool_rounds",
		"runtime.max_skill_tool_rounds",
		"runtime.force_tool_search_first",
		"runtime.return_skill_output_direct",
		"runtime.fallback_skill_search",
		"runtime.fallback_min_score",
		"runtime.max_activated_skills",
		"runtime.max_retained_skills",
		"runtime.activation_min_score",
		"runtime.activation_score_ratio",
		"database.migrations_dir",
		"database.auto_migrate",
	} {
		_ = v.BindEnv(key)
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Model.ID) == "" {
		return fmt.Errorf("config model.id is required")
	}
	if strings.TrimSpace(c.Model.BaseURL) == "" {
		return fmt.Errorf("config model.base_url is required")
	}
	if c.Model.ContextWindow <= 0 {
		return fmt.Errorf("config model.context_window must be positive")
	}
	if c.Model.MaxOutput <= 0 {
		return fmt.Errorf("config model.max_output_tokens must be positive")
	}
	if c.Model.TimeoutSeconds <= 0 {
		return fmt.Errorf("config model.timeout_seconds must be positive")
	}
	if strings.TrimSpace(c.Runtime.SkillsDir) == "" {
		return fmt.Errorf("config runtime.skills_dir is required")
	}
	if strings.TrimSpace(c.Runtime.RunsDir) == "" {
		return fmt.Errorf("config runtime.runs_dir is required")
	}
	if strings.TrimSpace(c.Runtime.WorkspaceRoot) == "" {
		return fmt.Errorf("config runtime.workspace_root is required")
	}
	if strings.TrimSpace(c.Runtime.DocumentOutputDir) == "" {
		return fmt.Errorf("config runtime.document_output_dir is required")
	}
	if c.Runtime.MaxToolRounds <= 0 {
		return fmt.Errorf("config runtime.max_tool_rounds must be positive")
	}
	if c.Runtime.MaxSkillToolRounds <= 0 {
		return fmt.Errorf("config runtime.max_skill_tool_rounds must be positive")
	}
	if c.Runtime.MaxActivatedSkills <= 0 {
		return fmt.Errorf("config runtime.max_activated_skills must be positive")
	}
	if c.Runtime.MaxRetainedSkills <= 0 {
		return fmt.Errorf("config runtime.max_retained_skills must be positive")
	}
	if c.Runtime.MaxRetainedSkills < c.Runtime.MaxActivatedSkills {
		return fmt.Errorf("config runtime.max_retained_skills must be >= runtime.max_activated_skills")
	}
	if c.Runtime.ActivationMinScore < 0 {
		return fmt.Errorf("config runtime.activation_min_score must be >= 0")
	}
	if c.Runtime.ActivationScoreRatio <= 0 || c.Runtime.ActivationScoreRatio > 1 {
		return fmt.Errorf("config runtime.activation_score_ratio must be in (0, 1]")
	}
	if _, err := os.Stat(c.Runtime.SkillsDir); err != nil {
		return fmt.Errorf("runtime.skills_dir not accessible %s: %w", c.Runtime.SkillsDir, err)
	}
	if _, err := os.Stat(c.Runtime.WorkspaceRoot); err != nil {
		return fmt.Errorf("runtime.workspace_root not accessible %s: %w", c.Runtime.WorkspaceRoot, err)
	}
	return nil
}

func resolve(root, p string) string {
	if p == "" {
		return root
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Clean(filepath.Join(root, p))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
