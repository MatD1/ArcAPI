package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
	"github.com/mat/arcapi/internal/services"
)

type AuthHandler struct {
	authService *services.AuthService
	userService *services.UserService
	cfg         *config.Config
	apiKeyRepo  *repository.APIKeyRepository
}

func NewAuthHandler(
	authService *services.AuthService,
	userService *services.UserService,
	cfg *config.Config,
	apiKeyRepo *repository.APIKeyRepository,
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
		cfg:         cfg,
		apiKeyRepo:  apiKeyRepo,
	}
}

// GetCurrentUser returns the authenticated user context
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	val, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	user, ok := val.(*models.User)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}
