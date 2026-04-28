package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// Config 表示应用程序的主配置结构，包含路径信息及运行时和数据库的子配置。
type Config struct {
	Path     string
	Root     string
	Runtime  RuntimeConfig  `mapstructure:"runtime"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

// RuntimeConfig 定义了运行时环境的配置，包括目录路径、项目设置及技能调用的控制参数。
type RuntimeConfig struct {
	SkillsDir            string  `mapstructure:"skills_dir"`
	RunsDir              string  `mapstructure:"runs_dir"`
	WorkspaceRoot        string  `mapstructure:"workspace_root"`
	DocumentOutputDir    string  `mapstructure:"document_output_dir"`
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

// DatabaseConfig 定义了数据库连接相关的配置信息。
type DatabaseConfig struct {
	URL                   string `mapstructure:"url"`
	Host                  string `mapstructure:"host"`
	Port                  int    `mapstructure:"port"`
	Name                  string `mapstructure:"name"`
	User                  string `mapstructure:"user"`
	Password              string `mapstructure:"password"`
	SSLMode               string `mapstructure:"sslmode"`
	ConnectTimeoutSeconds int    `mapstructure:"connect_timeout_seconds"`
	MigrationsDir         string `mapstructure:"migrations_dir"`
	AutoMigrate           bool   `mapstructure:"auto_migrate"`
}

// RedisConfig 定义 Redis Cluster 缓存连接配置。Redis 只做共享缓存，不作为配置主存储。
type RedisConfig struct {
	Enabled    bool     `mapstructure:"enabled"`
	Required   bool     `mapstructure:"required"`
	Mode       string   `mapstructure:"mode"`
	Addrs      []string `mapstructure:"addrs"`
	Password   string   `mapstructure:"password"`
	KeyPrefix  string   `mapstructure:"key_prefix"`
	TTLSeconds int      `mapstructure:"ttl_seconds"`
}

type LoggingConfig struct {
	Level       string `mapstructure:"level"`
	Encoding    string `mapstructure:"encoding"`
	Development bool   `mapstructure:"development"`
}

// Load 从指定路径加载配置文件，解析 YAML 内容，应用默认值和环境变量绑定，
// 处理相对路径，并验证配置的有效性。
//
// 参数:
//   - path: 配置文件的路径。
//
// 返回值:
//   - Config: 解析并验证后的配置对象。
//   - error: 如果在读取、解析或验证过程中发生错误，则返回错误信息。
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

	// 解析运行时配置中的相对路径，并应用默认值
	cfg.Runtime.SkillsDir = resolve(root, cfg.Runtime.SkillsDir)
	cfg.Runtime.RunsDir = resolve(root, cfg.Runtime.RunsDir)
	cfg.Runtime.WorkspaceRoot = resolve(root, firstNonEmpty(cfg.Runtime.WorkspaceRoot, ".."))
	cfg.Runtime.DocumentOutputDir = resolve(root, firstNonEmpty(cfg.Runtime.DocumentOutputDir, "../docs/08-generated-drafts"))
	// 清理数据库配置中的空白字符，并应用默认值
	cfg.Database.Host = strings.TrimSpace(cfg.Database.Host)
	cfg.Database.Name = strings.TrimSpace(cfg.Database.Name)
	cfg.Database.User = strings.TrimSpace(cfg.Database.User)
	cfg.Database.Password = strings.TrimSpace(cfg.Database.Password)
	cfg.Database.SSLMode = firstNonEmpty(cfg.Database.SSLMode, "disable")
	cfg.Database.MigrationsDir = resolve(root, firstNonEmpty(cfg.Database.MigrationsDir, "./internal/db/migrations"))
	cfg.Redis.Password = strings.TrimSpace(cfg.Redis.Password)
	cfg.Redis.Mode = strings.ToLower(firstNonEmpty(cfg.Redis.Mode, "cluster"))
	cfg.Redis.KeyPrefix = firstNonEmpty(cfg.Redis.KeyPrefix, "novelrt")
	if cfg.Redis.TTLSeconds <= 0 {
		cfg.Redis.TTLSeconds = 300
	}
	cfg.Logging.Level = strings.ToLower(firstNonEmpty(cfg.Logging.Level, "info"))
	cfg.Logging.Encoding = strings.ToLower(firstNonEmpty(cfg.Logging.Encoding, "json"))

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// setDefaults 为 Viper 实例设置所有配置项的默认值。
//
// 参数:
//   - v: Viper 实例指针。
func setDefaults(v *viper.Viper) {
	v.SetDefault("runtime.skills_dir", "./skills")
	v.SetDefault("runtime.runs_dir", "./runs")
	v.SetDefault("runtime.workspace_root", "..")
	v.SetDefault("runtime.document_output_dir", "../docs/08-generated-drafts")
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
	v.SetDefault("database.host", "")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.name", "")
	v.SetDefault("database.user", "")
	v.SetDefault("database.password", "")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.connect_timeout_seconds", 5)
	v.SetDefault("database.migrations_dir", "./internal/db/migrations")
	v.SetDefault("database.auto_migrate", true)
	v.SetDefault("redis.enabled", false)
	v.SetDefault("redis.required", false)
	v.SetDefault("redis.mode", "cluster")
	v.SetDefault("redis.addrs", []string{})
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.key_prefix", "novelrt")
	v.SetDefault("redis.ttl_seconds", 300)
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.encoding", "json")
	v.SetDefault("logging.development", false)
}

// bindEnv 配置 Viper 以支持环境变量覆盖。
// 设置前缀为 "NOVEL"，并将配置键中的 "." 替换为 "_" 以匹配环境变量命名规范。
// 显式绑定关键配置项以确保其可以通过环境变量进行覆盖。
//
// 参数:
//   - v: Viper 实例指针。
func bindEnv(v *viper.Viper) {
	v.SetEnvPrefix("NOVEL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	_ = v.BindEnv("database.url", "DATABASE_URL", "NOVEL_DATABASE_URL")
	_ = v.BindEnv("database.host", "DATABASE_HOST", "NOVEL_DATABASE_HOST")
	_ = v.BindEnv("database.port", "DATABASE_PORT", "NOVEL_DATABASE_PORT")
	_ = v.BindEnv("database.name", "DATABASE_NAME", "NOVEL_DATABASE_NAME")
	_ = v.BindEnv("database.user", "DATABASE_USER", "NOVEL_DATABASE_USER")
	_ = v.BindEnv("database.password", "DATABASE_PASSWORD", "NOVEL_DATABASE_PASSWORD")
	_ = v.BindEnv("database.sslmode", "DATABASE_SSLMODE", "NOVEL_DATABASE_SSLMODE")
	_ = v.BindEnv("database.connect_timeout_seconds", "DATABASE_CONNECT_TIMEOUT_SECONDS", "NOVEL_DATABASE_CONNECT_TIMEOUT_SECONDS")
	_ = v.BindEnv("redis.password", "REDIS_PASSWORD", "NOVEL_REDIS_PASSWORD")

	// Explicit binds make supported overrides discoverable and stable.
	for _, key := range []string{
		"runtime.skills_dir",
		"runtime.runs_dir",
		"runtime.workspace_root",
		"runtime.document_output_dir",
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
		"database.host",
		"database.port",
		"database.name",
		"database.user",
		"database.password",
		"database.sslmode",
		"database.connect_timeout_seconds",
		"database.migrations_dir",
		"database.auto_migrate",
		"redis.enabled",
		"redis.required",
		"redis.mode",
		"redis.addrs",
		"redis.password",
		"redis.key_prefix",
		"redis.ttl_seconds",
		"logging.level",
		"logging.encoding",
		"logging.development",
	} {
		_ = v.BindEnv(key)
	}
}

// Validate 验证配置对象的完整性和有效性。
// 检查必填字段是否为空，数值字段是否符合范围要求，以及指定的目录是否可访问。
//
// 返回值:
//   - error: 如果验证失败，返回描述具体错误的错误信息；否则返回 nil。
func (c Config) Validate() error {
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
	if _, err := c.Database.ConnectionString(); err != nil {
		return err
	}
	if c.Redis.Enabled && len(c.Redis.Addrs) == 0 {
		return fmt.Errorf("redis.addrs is required when redis.enabled is true")
	}
	if c.Redis.Enabled && c.Redis.Mode != "cluster" && c.Redis.Mode != "standalone" {
		return fmt.Errorf("redis.mode must be cluster or standalone")
	}
	if c.Redis.TTLSeconds < 0 {
		return fmt.Errorf("redis.ttl_seconds must be >= 0")
	}
	if c.Logging.Level != "" {
		switch c.Logging.Level {
		case "debug", "info", "warn", "error":
		default:
			return fmt.Errorf("logging.level must be debug, info, warn, or error")
		}
	}
	if c.Logging.Encoding != "" {
		switch c.Logging.Encoding {
		case "json", "console":
		default:
			return fmt.Errorf("logging.encoding must be json or console")
		}
	}
	return nil
}

// ConnectionString 生成数据库连接字符串。
// 如果配置中直接提供了 URL，则直接返回该 URL。
// 否则，根据 Host、Port、Name、User 等字段构建 PostgreSQL 连接字符串。
//
// 返回值:
//   - string: 数据库连接字符串。
//   - error: 如果缺少必要字段或格式错误，返回错误信息。
func (d DatabaseConfig) ConnectionString() (string, error) {
	if strings.TrimSpace(d.URL) != "" {
		return strings.TrimSpace(d.URL), nil
	}
	if strings.TrimSpace(d.Host) == "" {
		return "", fmt.Errorf("database.host is required when database.url is empty")
	}
	if d.Port <= 0 {
		return "", fmt.Errorf("database.port must be positive")
	}
	if strings.TrimSpace(d.Name) == "" {
		return "", fmt.Errorf("database.name is required when database.url is empty")
	}
	if strings.TrimSpace(d.User) == "" {
		return "", fmt.Errorf("database.user is required when database.url is empty")
	}
	connURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.User, d.Password),
		Host:   d.Host + ":" + strconv.Itoa(d.Port),
		Path:   "/" + d.Name,
	}
	query := url.Values{}
	query.Set("sslmode", firstNonEmpty(d.SSLMode, "disable"))
	if d.ConnectTimeoutSeconds > 0 {
		query.Set("connect_timeout", strconv.Itoa(d.ConnectTimeoutSeconds))
	}
	connURL.RawQuery = query.Encode()
	return connURL.String(), nil
}

// resolve 解析路径。
// 如果路径为空，返回根路径。
// 如果路径是绝对路径，直接返回。
// 否则，将路径与根路径拼接并清理后返回。
//
// 参数:
//   - root: 根目录路径。
//   - p: 待解析的路径。
//
// 返回值:
//   - string: 解析后的绝对路径。
func resolve(root, p string) string {
	if p == "" {
		return root
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Clean(filepath.Join(root, p))
}

// firstNonEmpty 返回参数列表中第一个非空（去除空白字符后）的字符串。
// 如果所有字符串都为空，则返回空字符串。
//
// 参数:
//   - values: 可变长度的字符串列表。
//
// 返回值:
//   - string: 第一个非空字符串，或空字符串。
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
