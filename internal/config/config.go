package config

import (
	"fmt"
	"os"
	"strings"

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

	// Redis - supports both URL format (redis://password@host:port) or separate config
	RedisURL      string `envconfig:"REDIS_URL" default:""`                // Single URL format: redis://password@host:port or redis://host:port
	RedisAddr     string `envconfig:"REDIS_ADDR" default:"localhost:6379"` // Fallback if REDIS_URL not set
	RedisPassword string `envconfig:"REDIS_PASSWORD" default:""`           // Fallback if REDIS_URL not set

	// Sync
	SyncCron string `envconfig:"SYNC_CRON" default:"*/15 * * * *"`

	// Server
	APIPort  string `envconfig:"PORT" default:"8080"` // Railway uses PORT env var
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`

	// Security
	AllowedOrigins string `envconfig:"ALLOWED_ORIGINS" default:""`

	// Rate Limiting
	RateLimitRequests      int `envconfig:"RATE_LIMIT_REQUESTS" default:"21"`
	RateLimitWindowSeconds int `envconfig:"RATE_LIMIT_WINDOW_SECONDS" default:"60"`
	RateLimitBurst         int `envconfig:"RATE_LIMIT_BURST" default:"8"`

	// Supabase Auth
	SupabaseURL            string `envconfig:"SUPABASE_URL" default:""`            // Main project URL (fallback: NEXT_PUBLIC_SUPABASE_URL)
	SupabaseJWKSURL        string `envconfig:"SUPABASE_JWKS_URL" default:""`        // Use if different from standard auth/v1/jwks
	SupabasePublishableKey string `envconfig:"SUPABASE_PUBLISHABLE_KEY" default:""` // Modern label (replacing "Anon Key")

	// GitHub
	GitHubToken string `envconfig:"GITHUB_TOKEN" default:""`
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists (ignore error if it doesn't)
	_ = godotenv.Load()

	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}

	// Manual fallbacks for Supabase terminology/format changes
	if cfg.SupabaseURL == "" {
		cfg.SupabaseURL = strings.TrimSpace(os.Getenv("NEXT_PUBLIC_SUPABASE_URL"))
	}
	if cfg.SupabasePublishableKey == "" {
		cfg.SupabasePublishableKey = strings.TrimSpace(os.Getenv("SUPABASE_ANON_KEY"))
		if cfg.SupabasePublishableKey == "" {
			cfg.SupabasePublishableKey = strings.TrimSpace(os.Getenv("NEXT_PUBLIC_SUPABASE_PUBLISHABLE_DEFAULT_KEY"))
		}
	}

	return &cfg, nil
}

func (c *Config) GetDSN() string {
	// Add connection timeout (5 seconds) and statement timeout (10 seconds) to prevent hanging
	// connect_timeout: how long to wait when establishing connection
	// statement_timeout: how long to wait for a query to complete
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=5 statement_timeout=10000",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

func (c *Config) IsOAuthEnabled() bool {
	return false // OAuth is now managed entirely by Supabase
}

func (c *Config) IsGitHubOAuthEnabled() bool {
	return false
}

func (c *Config) IsDiscordOAuthEnabled() bool {
	return false
}

func (c *Config) GetAllowedOrigins() []string {
	if c.AllowedOrigins == "" {
		return []string{}
	}
	origins := strings.Split(c.AllowedOrigins, ",")
	result := make([]string, 0, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
