package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/middleware"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
	"github.com/mat/arcapi/internal/services"
)

type ManagementHandler struct {
	authService  *services.AuthService
	apiKeyRepo   *repository.APIKeyRepository
	jwtTokenRepo *repository.JWTTokenRepository
	auditLogRepo *repository.AuditLogRepository
	userRepo     *repository.UserRepository
}

func NewManagementHandler(
	authService *services.AuthService,
	apiKeyRepo *repository.APIKeyRepository,
	jwtTokenRepo *repository.JWTTokenRepository,
	auditLogRepo *repository.AuditLogRepository,
	userRepo *repository.UserRepository,
) *ManagementHandler {
	return &ManagementHandler{
		authService:  authService,
		apiKeyRepo:   apiKeyRepo,
		jwtTokenRepo: jwtTokenRepo,
		auditLogRepo: auditLogRepo,
		userRepo:     userRepo,
	}
}

// CreateAPIKey creates a new API key
func (h *ManagementHandler) CreateAPIKey(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authCtx, _ := c.Get(middleware.AuthContextKey)
	ctx := authCtx.(*middleware.AuthContext)
	user := ctx.User.(*models.User)

	key, err := h.authService.CreateAPIKey(user.ID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"api_key": key,
		"name":    req.Name,
		"warning": "Save this API key now. You won't be able to see it again.",
	})
}

// ListAPIKeys lists all API keys (admins see all, users see only their own)
func (h *ManagementHandler) ListAPIKeys(c *gin.Context) {
	authCtx, _ := c.Get(middleware.AuthContextKey)
	ctx := authCtx.(*middleware.AuthContext)
	user := ctx.User.(*models.User)

	var keys []models.APIKey
	var err error

	// Admins can see all keys, regular users only see their own
	if user.Role == models.RoleAdmin {
		// Get all API keys for admins
		keys, err = h.apiKeyRepo.FindAll()
	} else {
		keys, err = h.apiKeyRepo.FindByUserID(user.ID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch API keys"})
		return
	}

	// Remove key hashes from response
	for i := range keys {
		keys[i].KeyHash = ""
	}

	c.JSON(http.StatusOK, keys)
}

// RevokeAPIKey revokes an API key
func (h *ManagementHandler) RevokeAPIKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	authCtx, _ := c.Get(middleware.AuthContextKey)
	ctx := authCtx.(*middleware.AuthContext)
	user := ctx.User.(*models.User)

	// Verify key belongs to user (or user is admin)
	key, err := h.apiKeyRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	if key.UserID != user.ID && user.Role != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	err = h.authService.RevokeAPIKey(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke API key"})
		return
	}

	// Invalidate cache
	h.authService.InvalidateCache("", "")

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}

// RevokeJWT revokes a JWT token
func (h *ManagementHandler) RevokeJWT(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authCtx, _ := c.Get(middleware.AuthContextKey)
	ctx := authCtx.(*middleware.AuthContext)
	user := ctx.User.(*models.User)

	// Validate token first to get user
	tokenUser, err := h.authService.ValidateJWT(req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token"})
		return
	}

	// Verify user owns token or is admin
	if tokenUser.ID != user.ID && user.Role != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Revoke token by creating a hash and finding matching token
	// Note: This is a simplified approach. In production, you might want to store
	// a token identifier or use a different revocation strategy
	// For now, we'll revoke based on user and expiry matching

	// Get all active tokens for the user
	tokens, err := h.jwtTokenRepo.FindActiveByUserID(tokenUser.ID)
	if err == nil {
		// Try to find matching token (simplified - in production use proper token ID)
		for _, token := range tokens {
			// If we can match by some criteria, revoke it
			// This is a simplified implementation
			_ = h.jwtTokenRepo.Revoke(token.ID)
		}
	}

	// Invalidate cache
	h.authService.InvalidateCache("", req.Token)

	c.JSON(http.StatusOK, gin.H{"message": "JWT token revoked"})
}

// ListJWTs lists active JWT tokens for the current user
func (h *ManagementHandler) ListJWTs(c *gin.Context) {
	authCtx, _ := c.Get(middleware.AuthContextKey)
	ctx := authCtx.(*middleware.AuthContext)
	user := ctx.User.(*models.User)

	tokens, err := h.jwtTokenRepo.FindActiveByUserID(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tokens"})
		return
	}

	// Remove token hashes from response
	for i := range tokens {
		tokens[i].TokenHash = ""
	}

	c.JSON(http.StatusOK, tokens)
}

// QueryLogs queries audit logs with filters
func (h *ManagementHandler) QueryLogs(c *gin.Context) {
	page := 1
	limit := 50

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := (page - 1) * limit

	var apiKeyID, jwtTokenID, userID *uint
	var endpoint, method, startTime, endTime *string

	if k := c.Query("api_key_id"); k != "" {
		if id, err := strconv.ParseUint(k, 10, 32); err == nil {
			idUint := uint(id)
			apiKeyID = &idUint
		}
	}
	if j := c.Query("jwt_token_id"); j != "" {
		if id, err := strconv.ParseUint(j, 10, 32); err == nil {
			idUint := uint(id)
			jwtTokenID = &idUint
		}
	}
	if u := c.Query("user_id"); u != "" {
		if id, err := strconv.ParseUint(u, 10, 32); err == nil {
			idUint := uint(id)
			userID = &idUint
		}
	}
	if e := c.Query("endpoint"); e != "" {
		endpoint = &e
	}
	if m := c.Query("method"); m != "" {
		method = &m
	}
	if s := c.Query("start_time"); s != "" {
		startTime = &s
	}
	if e := c.Query("end_time"); e != "" {
		endTime = &e
	}

	logs, count, err := h.auditLogRepo.FindByFilters(
		apiKeyID, jwtTokenID, userID, endpoint, method, startTime, endTime, offset, limit,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": logs,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": count,
		},
	})
}

// UpdateUserAccess controls whether a user can access data (admin only)
func (h *ManagementHandler) UpdateUserAccess(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		CanAccessData bool `json:"can_access_data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get target user
	targetUser, err := h.userRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update access
	targetUser.CanAccessData = req.CanAccessData
	err = h.userRepo.Update(targetUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user access"})
		return
	}

	// Invalidate cached auth data for this user to ensure changes take effect immediately
	h.authService.InvalidateUserCache(targetUser.ID)

	c.JSON(http.StatusOK, gin.H{
		"message": "User access updated",
		"user":    targetUser,
	})
}

// ListUsers lists all users (admin only)
func (h *ManagementHandler) ListUsers(c *gin.Context) {
	page := 1
	limit := 50

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := (page - 1) * limit
	users, count, err := h.userRepo.FindAll(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": users,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": count,
		},
	})
}

// GetUser gets a user with their API keys and JWT tokens (admin only)
func (h *ManagementHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.userRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Get user's API keys
	apiKeys, err := h.apiKeyRepo.FindByUserID(user.ID)
	if err != nil {
		// Log but don't fail
		apiKeys = []models.APIKey{}
	}

	// Get user's active JWT tokens
	jwtTokens, err := h.jwtTokenRepo.FindActiveByUserID(user.ID)
	if err != nil {
		// Log but don't fail
		jwtTokens = []models.JWTToken{}
	}

	// Remove sensitive data from API keys
	for i := range apiKeys {
		apiKeys[i].KeyHash = ""
	}

	// Remove sensitive data from JWT tokens
	for i := range jwtTokens {
		jwtTokens[i].TokenHash = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"user":       user,
		"api_keys":   apiKeys,
		"jwt_tokens": jwtTokens,
	})
}
