package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/services"
)

type AuthContext struct {
	User     interface{}
	APIKey   interface{}
	JWTToken interface{}
}

const AuthContextKey = "auth_context"

// AuthenticateRequest validates request using Supabase JWT or API Key.
// It returns the associated user and the raw credentials (token or key).
func AuthenticateRequest(c *gin.Context, authService *services.AuthService, supabaseService *services.SupabaseAuthService, cfg *config.Config) (*models.User, string, error) {
	// 1. Try API Key first (common for programmatic access)
	apiKeyString := c.GetHeader("X-API-Key")
	if apiKeyString != "" {
		apiKey, err := authService.ValidateAPIKey(apiKeyString)
		if err == nil {
			user, err := authService.UserRepo().FindByID(apiKey.UserID)
			if err == nil {
				return user, apiKeyString, nil
			}
		}
	}

	// 2. Try Authorization: Bearer <token> (Supabase JWT)
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenString := parts[1]
			user, err := ValidateTokenString(tokenString, authService, supabaseService, cfg)
			if err == nil {
				return user, tokenString, nil
			}
		}
	}

	return nil, "", fmt.Errorf("authentication required (Supabase JWT or X-API-Key)")
}

// ValidateTokenString validates a raw token string using Supabase.
func ValidateTokenString(tokenString string, authService *services.AuthService, supabaseService *services.SupabaseAuthService, cfg *config.Config) (*models.User, error) {
	if supabaseService == nil {
		return nil, fmt.Errorf("supabase auth service not available")
	}

	claims, err := supabaseService.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return authService.SyncSupabaseUser(claims)
}

// JWTAuthMiddleware validates authentication for read operations
func JWTAuthMiddleware(authService *services.AuthService, cfg *config.Config, supabaseService *services.SupabaseAuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, token, err := AuthenticateRequest(c, authService, supabaseService, cfg)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		c.Set(AuthContextKey, &AuthContext{
			User:     user,
			APIKey:   nil,
			JWTToken: token,
		})
		c.Set("user", user)
		c.Set("user_id", user.ID)

		c.Next()
	}
}

// WriteAuthMiddleware only allows admin users to perform write operations
// Regular users are restricted to read-only access, even with API keys
func WriteAuthMiddleware(authService *services.AuthService, cfg *config.Config, supabaseService *services.SupabaseAuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, token, err := AuthenticateRequest(c, authService, supabaseService, cfg)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// Only admin users can perform write operations
		if user.Role != models.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Write operations are restricted to admin users only"})
			c.Abort()
			return
		}

		// Admin users can write with JWT only (no API key required)
		c.Set(AuthContextKey, &AuthContext{
			User:     user,
			APIKey:   nil,
			JWTToken: token,
		})
		c.Set("user", user)
		c.Set("user_id", user.ID)

		c.Next()
	}
}

// AuthMiddleware validates both API key and JWT token (legacy, kept for backward compatibility)
func AuthMiddleware(authService *services.AuthService, cfg *config.Config, supabaseService *services.SupabaseAuthService) gin.HandlerFunc {
	return WriteAuthMiddleware(authService, cfg, supabaseService)
}

// ProgressAuthMiddleware allows all authenticated users to read and update their own progress
// Progress endpoints are always accessible regardless of can_access_data status
// Users can only access/modify their own progress (handled by handlers using authenticated user ID)
func ProgressAuthMiddleware(authService *services.AuthService, cfg *config.Config, supabaseService *services.SupabaseAuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, token, err := AuthenticateRequest(c, authService, supabaseService, cfg)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// Progress endpoints are always accessible - no can_access_data check needed
		// Users can only access/modify their own progress (handlers ensure this)

		// Store auth context
		c.Set(AuthContextKey, &AuthContext{
			User:     user,
			APIKey:   nil,
			JWTToken: token,
		})
		c.Set("user", user)
		c.Set("user_id", user.ID)

		c.Next()
	}
}

// AdminMiddleware checks if user has admin role
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx, exists := c.Get(AuthContextKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		ctx := authCtx.(*AuthContext)
		user, ok := ctx.User.(*models.User)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid user context"})
			c.Abort()
			return
		}

		if user.Role != models.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}
