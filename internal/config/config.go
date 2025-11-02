package config

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// Database
	DBHost     string `envconfig:"DB_HOST" default:"localhost"`
	DBPort     int    `envconfig:"DB_PORT" default:"5432"`
	DBUser     string `envconfig:"DB_USER" default:"postgres"`
	DBPassword string `envconfig:"DB_PASSWORD" required:"true"`
	DBName     string `envconfig:"DB_NAME" default:"arcapi"`
	DBSSLMode  string `envconfig:"DB_SSL_MODE" default:"disable"`

	// Redis
	RedisAddr     string `envconfig:"REDIS_ADDR" default:"localhost:6379"`
	RedisPassword string `envconfig:"REDIS_PASSWORD" default:""`

	// JWT
	JWTSecret      string `envconfig:"JWT_SECRET" required:"true"`
	JWTExpiryHours int    `envconfig:"JWT_EXPIRY_HOURS" default:"72"`

	// OAuth
	GitHubClientID      string `envconfig:"GITHUB_CLIENT_ID" default:""`
	GitHubClientSecret  string `envconfig:"GITHUB_CLIENT_SECRET" default:""`
	OAuthEnabled        bool   `envconfig:"OAUTH_ENABLED" default:"true"`
	OAuthRedirectURL    string `envconfig:"OAUTH_REDIRECT_URL" default:"http://localhost:8080/api/v1/auth/github/callback"`
	FrontendCallbackURL string `envconfig:"FRONTEND_CALLBACK_URL" default:"http://localhost:8080/dashboard/api/auth/github/callback/"`

	// Sync
	SyncCron string `envconfig:"SYNC_CRON" default:"*/15 * * * *"`

	// Server
	APIPort  string `envconfig:"PORT" default:"8080"` // Railway uses PORT env var
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists (ignore error if it doesn't)
	_ = godotenv.Load()

	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

func (c *Config) IsOAuthEnabled() bool {
	return c.OAuthEnabled && c.GitHubClientID != "" && c.GitHubClientSecret != ""
}
