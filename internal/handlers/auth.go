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
// GetCurrentUser returns the authenticated user context
// @Summary Get current user
// @Description Returns the details of the currently authenticated user
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]models.User "Successfully fetched user details"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /me [get]
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

// Login handles authentication via API key
// Login handles authentication via API key
// @Summary Login with API key
// @Description Authenticate a user using an API key. Returns user details and the key as a token.
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body map[string]string true "API Key"
// @Success 200 {object} map[string]interface{} "Authenticated successfully"
// @Failure 400 {object} ErrorResponse "API key is required"
// @Failure 401 {object} ErrorResponse "Invalid API key or user not found"
// @Router /login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		APIKey string `json:"api_key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API key is required"})
		return
	}

	// Validate API key
	apiKey, err := h.authService.ValidateAPIKey(req.APIKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
		return
	}

	// Get associated user
	user, err := h.userService.GetByID(apiKey.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// For API key login, we return a simpler response for now
	// The frontend handles the Supabase/JWT flow separately
	c.JSON(http.StatusOK, gin.H{
		"message": "Authenticated successfully",
		"user":    user,
		"token":   req.APIKey, // Return the key itself as the token to preserve context
	})
}
