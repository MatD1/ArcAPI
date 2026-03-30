package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/middleware"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
	"github.com/mat/arcapi/internal/services"
)

type ManagementHandler struct {
	authService       *services.AuthService
	apiKeyRepo        *repository.APIKeyRepository
	auditLogRepo      *repository.AuditLogRepository
	userRepo          *repository.UserRepository
	hideoutModuleRepo *repository.HideoutModuleRepository
}

func NewManagementHandler(
	authService *services.AuthService,
	apiKeyRepo *repository.APIKeyRepository,
	auditLogRepo *repository.AuditLogRepository,
	userRepo *repository.UserRepository,
	hideoutModuleRepo *repository.HideoutModuleRepository,
) *ManagementHandler {
	return &ManagementHandler{
		authService:       authService,
		apiKeyRepo:        apiKeyRepo,
		auditLogRepo:      auditLogRepo,
		userRepo:          userRepo,
		hideoutModuleRepo: hideoutModuleRepo,
	}
}

// CreateAPIKey creates a new API key (admin only)
// CreateAPIKey creates a new API key (admin only)
// @Summary Create API key
// @Description Generate a new API key for the current user. Only admins can create keys.
// @Tags management
// @Accept json
// @Produce json
// @Param name body string true "Key name"
// @Success 201 {object} map[string]string "Successfully created API key"
// @Failure 400 {object} ErrorResponse "Invalid input data"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /management/api-keys [post]
func (h *ManagementHandler) CreateAPIKey(c *gin.Context) {
	// Get current user from context
	authCtx, _ := c.Get(middleware.AuthContextKey)
	ctx := authCtx.(*middleware.AuthContext)
	user := ctx.User.(*models.User)

	// Security check: Only admins can create API keys
	if user.Role != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only administrators can create API keys"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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
// ListAPIKeys lists all API keys (admins see all, users see only their own)
// @Summary List API keys
// @Description Fetch API keys. Admins see all keys, regular users see only their own.
// @Tags management
// @Accept json
// @Produce json
// @Success 200 {array} models.APIKey "Successfully fetched API keys"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /management/api-keys [get]
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
// RevokeAPIKey revokes an API key
// @Summary Revoke API key
// @Description Deactivate an API key by its ID
// @Tags management
// @Accept json
// @Produce json
// @Param id path int true "API Key ID"
// @Success 200 {object} map[string]string "Successfully revoked API key"
// @Failure 400 {object} ErrorResponse "Invalid key ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "API key not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /management/api-keys/{id}/revoke [post]
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

// QueryLogs queries audit logs with filters
// QueryLogs queries audit logs with filters
// @Summary Query audit logs
// @Description Fetch audit logs with various filters and pagination
// @Tags management
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Logs per page" default(50)
// @Param api_key_id query int false "Filter by API key ID"
// @Param user_id query int false "Filter by User ID"
// @Param method query string false "Filter by HTTP method"
// @Param endpoint query string false "Filter by endpoint"
// @Param start_time query string false "Filter by start time (RFC3339)"
// @Param end_time query string false "Filter by end time (RFC3339)"
// @Success 200 {object} PaginatedResponse{data=[]models.AuditLog} "Successfully fetched logs"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /management/logs [get]
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
// UpdateUserAccess controls whether a user can access data (admin only)
// @Summary Update user data access
// @Description Enable or disable data access for a user. Only admins can update access.
// @Tags management
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param access body map[string]bool true "Access payload (can_access_data)"
// @Success 200 {object} map[string]interface{} "Successfully updated user access"
// @Failure 400 {object} ErrorResponse "Invalid user ID or input"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /management/users/{id}/access [put]
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

	// Get current user from context
	authCtx, _ := c.Get(middleware.AuthContextKey)
	ctx := authCtx.(*middleware.AuthContext)
	currentUser := ctx.User.(*models.User)

	// Get target user
	targetUser, err := h.userRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Security check: Only admins can update user access (enforced by AdminMiddleware, but double-check)
	if currentUser.Role != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can update user access"})
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

// UpdateUserRole updates a user's role (admin only)
// UpdateUserRole updates a user's role (admin only)
// @Summary Update user role
// @Description Change a user's role to 'admin' or 'user'. Only admins can update roles.
// @Tags management
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param role body map[string]string true "Role payload (role: 'admin' or 'user')"
// @Success 200 {object} map[string]interface{} "Successfully updated user role"
// @Failure 400 {object} ErrorResponse "Invalid user ID or role"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /management/users/{id}/role [put]
func (h *ManagementHandler) UpdateUserRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required,oneof=admin user"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current user from context
	authCtx, _ := c.Get(middleware.AuthContextKey)
	ctx := authCtx.(*middleware.AuthContext)
	currentUser := ctx.User.(*models.User)

	// Get target user
	targetUser, err := h.userRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Security check: Only admins can update user roles (enforced by AdminMiddleware, but double-check)
	if currentUser.Role != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can update user roles"})
		return
	}

	// Prevent admins from demoting themselves
	if currentUser.ID == targetUser.ID && req.Role != string(models.RoleAdmin) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot change your own admin role"})
		return
	}

	// Update role
	targetUser.Role = models.UserRole(req.Role)
	err = h.userRepo.Update(targetUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user role"})
		return
	}

	// Invalidate cached auth data for this user to ensure changes take effect immediately
	h.authService.InvalidateUserCache(targetUser.ID)

	c.JSON(http.StatusOK, gin.H{
		"message": "User role updated",
		"user":    targetUser,
	})
}

// ListUsers lists all users (admin only)
// ListUsers lists all users (admin only)
// @Summary List all users
// @Description Fetch all users with optional pagination. Only admins can list users.
// @Tags management
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Users per page" default(50)
// @Success 200 {object} PaginatedResponse{data=[]models.User} "Successfully fetched users"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /management/users [get]
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

// GetUser gets a user with their API keys and JWT tokens
// Admins can view any user, regular users can only view themselves
// GetUser gets a user with their API keys and JWT tokens
// @Summary Get user details
// @Description Fetch detailed user data including associated API keys. Admins see anyone, users see only themselves.
// @Tags management
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "Successfully fetched user details"
// @Failure 400 {object} ErrorResponse "Invalid user ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /management/users/{id} [get]
func (h *ManagementHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get current user from context
	authCtx, _ := c.Get(middleware.AuthContextKey)
	ctx := authCtx.(*middleware.AuthContext)
	currentUser := ctx.User.(*models.User)

	// Security check: Users can only view their own data unless they're admin
	if currentUser.Role != models.RoleAdmin && currentUser.ID != uint(id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only view your own user data"})
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

	c.JSON(http.StatusOK, gin.H{
		"user":     user,
		"api_keys": apiKeys,
	})
}

// UpdateUserProfile allows users to update their own profile
// Regular users can ONLY update their username (not email or other fields)
// Admins can update any user's profile including email
// UpdateUserProfile allows users to update their own profile
// @Summary Update user profile
// @Description Modify user profile details (email/username). Admins can update any user, regular users only their own username.
// @Tags management
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param profile body map[string]string true "Profile update payload (email, username)"
// @Success 200 {object} map[string]interface{} "Successfully updated user profile"
// @Failure 400 {object} ErrorResponse "Invalid user ID or input"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /management/users/{id} [put]
func (h *ManagementHandler) UpdateUserProfile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get current user from context
	authCtx, _ := c.Get(middleware.AuthContextKey)
	ctx := authCtx.(*middleware.AuthContext)
	currentUser := ctx.User.(*models.User)

	// Security check: Users can only update their own profile unless they're admin
	if currentUser.Role != models.RoleAdmin && currentUser.ID != uint(id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own profile"})
		return
	}

	var req struct {
		Email    *string `json:"email"`
		Username *string `json:"username"`
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

	// Permission check: Regular users can only update username, admins can update everything
	if currentUser.Role != models.RoleAdmin {
		// Regular user can only update username
		if req.Email != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your username. Contact an administrator to change your email."})
			return
		}
		if req.Username != nil {
			targetUser.Username = *req.Username
		}
	} else {
		// Admin can update everything
		if req.Email != nil {
			targetUser.Email = *req.Email
		}
		if req.Username != nil {
			targetUser.Username = *req.Username
		}
	}

	err = h.userRepo.Update(targetUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User profile updated",
		"user":    targetUser,
	})
}

// DeleteUser deletes a user and all associated data (admin only)
// DeleteUser deletes a user and all associated data (admin only)
// @Summary Delete user
// @Description Permanently delete a user and all their associated data. Only admins can delete users.
// @Tags management
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]string "Successfully deleted user"
// @Failure 400 {object} ErrorResponse "Invalid user ID or attempt to delete self"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /management/users/{id} [delete]
func (h *ManagementHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get target user
	targetUser, err := h.userRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Prevent deleting yourself
	authCtx, _ := c.Get(middleware.AuthContextKey)
	ctx := authCtx.(*middleware.AuthContext)
	currentUser := ctx.User.(*models.User)

	if targetUser.ID == currentUser.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete your own account"})
		return
	}

	// Delete user (cascade will handle related records)
	err = h.userRepo.Delete(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// CleanupDuplicateHideoutModules removes duplicate hideout modules, keeping the one with the lowest ID
// CleanupDuplicateHideoutModules removes duplicate hideout modules, keeping the one with the lowest ID
// @Summary Cleanup duplicate hideout modules
// @Description Identify and remove duplicate hideout modules by their external_id. Keeps only the one with the lowest numeric ID. Only admins can perform this.
// @Tags management
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Cleanup report"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /management/hideout-modules/cleanup [post]
func (h *ManagementHandler) CleanupDuplicateHideoutModules(c *gin.Context) {
	// Find all hideout modules (using a large limit to get all)
	allModules, _, err := h.hideoutModuleRepo.FindAll(0, 1000000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch hideout modules"})
		return
	}

	// Group by external_id
	externalIDMap := make(map[string][]models.HideoutModule)
	for _, module := range allModules {
		externalIDMap[module.ExternalID] = append(externalIDMap[module.ExternalID], module)
	}

	var deletedCount int
	var keptCount int

	// For each external_id with duplicates, keep the one with lowest ID and delete the rest
	for _, modules := range externalIDMap {
		if len(modules) > 1 {
			// Find the one with lowest ID to keep
			lowestID := modules[0].ID
			lowestIndex := 0
			for i, m := range modules {
				if m.ID < lowestID {
					lowestID = m.ID
					lowestIndex = i
				}
			}

			// Delete all except the one with lowest ID
			for i, m := range modules {
				if i != lowestIndex {
					if err := h.hideoutModuleRepo.Delete(m.ID); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{
							"error": fmt.Sprintf("Failed to delete duplicate module %d: %v", m.ID, err),
						})
						return
					}
					deletedCount++
				}
			}
			keptCount++
		} else {
			keptCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Cleanup completed",
		"deleted":      deletedCount,
		"kept":         keptCount,
		"total_before": len(allModules),
		"total_after":  len(allModules) - deletedCount,
	})
}
