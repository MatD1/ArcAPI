package config

import (
	"fmt"
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

	// JWT
	JWTSecret      string `envconfig:"JWT_SECRET" required:"true"`
	JWTExpiryHours int    `envconfig:"JWT_EXPIRY_HOURS" default:"72"`
	RefreshTokenExpiryDays int `envconfig:"REFRESH_TOKEN_EXPIRY_DAYS" default:"14"`

	// OAuth - GitHub
	GitHubClientID     string `envconfig:"GITHUB_CLIENT_ID" default:""`
	GitHubClientSecret string `envconfig:"GITHUB_CLIENT_SECRET" default:""`

	// OAuth - Discord
	DiscordClientID     string `envconfig:"DISCORD_CLIENT_ID" default:""`
	DiscordClientSecret string `envconfig:"DISCORD_CLIENT_SECRET" default:""`

	OAuthEnabled        bool   `envconfig:"OAUTH_ENABLED" default:"true"`
	OAuthRedirectURL    string `envconfig:"OAUTH_REDIRECT_URL" default:"http://localhost:8080/api/v1/auth/github/callback"`
	DiscordRedirectURL  string `envconfig:"DISCORD_REDIRECT_URL" default:"http://localhost:8080/api/v1/auth/discord/callback"`
	FrontendCallbackURL string `envconfig:"FRONTEND_CALLBACK_URL" default:"http://localhost:8080/dashboard/api/auth/github/callback/"`

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

	// Authentik OIDC
	AuthentikEnabled     bool   `envconfig:"AUTHENTIK_ENABLED" default:"false"`
	AuthentikIssuer      string `envconfig:"AUTHENTIK_ISSUER" default:""`
	AuthentikClientID    string `envconfig:"AUTHENTIK_CLIENT_ID" default:""`
	AuthentikClientSecret string `envconfig:"AUTHENTIK_CLIENT_SECRET" default:""`
	AuthentikJWKSURL     string `envconfig:"AUTHENTIK_JWKS_URL" default:""`
	AuthentikAuthURL     string `envconfig:"AUTHENTIK_AUTH_URL" default:""`
	AuthentikTokenURL    string `envconfig:"AUTHENTIK_TOKEN_URL" default:""`
	AuthentikLogoutURL   string `envconfig:"AUTHENTIK_LOGOUT_URL" default:""`
	AuthentikAdminGroup  string `envconfig:"AUTHENTIK_ADMIN_GROUP" default:"arcdb-admins"`
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
	return c.OAuthEnabled && (c.GitHubClientID != "" && c.GitHubClientSecret != "" || c.DiscordClientID != "" && c.DiscordClientSecret != "")
}

func (c *Config) IsGitHubOAuthEnabled() bool {
	return c.OAuthEnabled && c.GitHubClientID != "" && c.GitHubClientSecret != ""
}

func (c *Config) IsDiscordOAuthEnabled() bool {
	return c.OAuthEnabled && c.DiscordClientID != "" && c.DiscordClientSecret != ""
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
