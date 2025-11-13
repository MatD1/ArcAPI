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

// JWTAuthMiddleware validates JWT token only (for read operations)
func JWTAuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		// All authenticated users have read access by default (no CanAccessData check needed)

		// Store auth context
		c.Set(AuthContextKey, &AuthContext{
			User:     user,
			APIKey:   nil,
			JWTToken: tokenString,
		})
		c.Set("user", user)
		c.Set("user_id", user.ID)

		c.Next()
	}
}

// WriteAuthMiddleware only allows admin users to perform write operations
// Regular users are restricted to read-only access, even with API keys
func WriteAuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get JWT token from Authorization header (required)
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
			JWTToken: tokenString,
		})
		c.Set("user", user)
		c.Set("user_id", user.ID)

		c.Next()
	}
}

// AuthMiddleware validates both API key and JWT token (legacy, kept for backward compatibility)
func AuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return WriteAuthMiddleware(authService)
}

// ProgressAuthMiddleware allows all authenticated users to read and update their own progress
// Progress endpoints are always accessible regardless of can_access_data status
// Users can only access/modify their own progress (handled by handlers using authenticated user ID)
func ProgressAuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		// Progress endpoints are always accessible - no can_access_data check needed
		// Users can only access/modify their own progress (handlers ensure this)

		// Store auth context
		c.Set(AuthContextKey, &AuthContext{
			User:     user,
			APIKey:   nil,
			JWTToken: tokenString,
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
