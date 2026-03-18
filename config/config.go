package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config is the root configuration structure loaded from config.yaml / env vars.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Auth     AuthConfig     `mapstructure:"auth"`
	LDAP     LDAPConfig     `mapstructure:"ldap"`
	SMTP     SMTPConfig     `mapstructure:"smtp"`
	Telegram TelegramConfig `mapstructure:"telegram"`
	Slack    SlackConfig    `mapstructure:"slack"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
}

type ServerConfig struct {
	Address string `mapstructure:"address"` // e.g. ":3000"
}

type DatabaseConfig struct {
	Type string `mapstructure:"type"` // sqlite | mysql | postgres
	DSN  string `mapstructure:"dsn"`  // full DSN or path for sqlite
}

type AuthConfig struct {
	JWTSecret          string `mapstructure:"jwt_secret"`
	JWTExpiryHours     int    `mapstructure:"jwt_expiry_hours"`
	RefreshExpiryHours int    `mapstructure:"refresh_expiry_hours"`
	AllowLocalLogin    bool   `mapstructure:"allow_local_login"`
}

type LDAPConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	URL         string `mapstructure:"url"`
	BindDN      string `mapstructure:"bind_dn"`
	BindPass    string `mapstructure:"bind_password"`
	BaseDN      string `mapstructure:"base_dn"`
	UserFilter  string `mapstructure:"user_filter"`
	TLS         string `mapstructure:"tls"` // none | tls | starttls
	AttrEmail   string `mapstructure:"attr_email"`
	AttrName    string `mapstructure:"attr_display_name"`
	DefaultRole string `mapstructure:"default_role"` // admin | user
}

type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	TLS      bool   `mapstructure:"tls"`
}

type TelegramConfig struct {
	BotToken string `mapstructure:"bot_token"`
}

type SlackConfig struct {
	WebhookURL string `mapstructure:"webhook_url"`
}

type MetricsConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	BearerToken string `mapstructure:"bearer_token"` // optional auth for /metrics
}

// Load reads config from config.yaml and environment variables.
// Env vars override file values. Prefix: STATUSY_
// Example: STATUSY_DATABASE_TYPE=postgres
func Load() (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("server.address", ":3000")
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.dsn", "statusy.db")
	v.SetDefault("auth.jwt_expiry_hours", 24)
	v.SetDefault("auth.refresh_expiry_hours", 168) // 7 days
	v.SetDefault("auth.allow_local_login", true)
	v.SetDefault("ldap.tls", "none")
	v.SetDefault("ldap.attr_email", "mail")
	v.SetDefault("ldap.attr_display_name", "cn")
	v.SetDefault("ldap.default_role", "user")
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("smtp.port", 587)
	v.SetDefault("smtp.tls", true)

	// Config file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("data") // Docker: /app/data/config.yaml (persisted volume)
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/statusy")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
		// No config file is fine — use defaults + env vars
	}

	// Environment variable overrides
	v.SetEnvPrefix("STATUSY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicitly bind env vars so they always override config file values.
	// Viper's AutomaticEnv() does NOT override keys already set in the file —
	// BindEnv forces the env var to win regardless.
	_ = v.BindEnv("auth.jwt_secret", "STATUSY_AUTH_JWT_SECRET")
	_ = v.BindEnv("database.dsn", "STATUSY_DATABASE_DSN")
	_ = v.BindEnv("server.address", "STATUSY_SERVER_ADDRESS")

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	if cfg.Auth.JWTSecret == "" {
		return nil, fmt.Errorf("auth.jwt_secret must be set (or STATUSY_AUTH_JWT_SECRET env var)")
	}

	return cfg, nil
}
