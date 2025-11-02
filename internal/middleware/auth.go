package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/services"
)

type AuthContext struct {
	User     interface{}
	APIKey   interface{}
	JWTToken interface{}
}

const AuthContextKey = "auth_context"

// AuthMiddleware validates both API key and JWT token
func AuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get API key from header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			c.Abort()
			return
		}

		// Validate API key
		key, err := authService.ValidateAPIKey(apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}

		// Get JWT token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "JWT token required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate JWT
		user, err := authService.ValidateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired JWT token"})
			c.Abort()
			return
		}

		// Store auth context
		c.Set(AuthContextKey, &AuthContext{
			User:     user,
			APIKey:   key,
			JWTToken: tokenString,
		})
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("api_key_id", key.ID)

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
