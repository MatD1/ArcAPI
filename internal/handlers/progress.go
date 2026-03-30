package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
)

type ProgressHandler struct {
	questProgressRepo         *repository.UserQuestProgressRepository
	hideoutModuleProgressRepo *repository.UserHideoutModuleProgressRepository
	skillNodeProgressRepo     *repository.UserSkillNodeProgressRepository
	blueprintProgressRepo     *repository.UserBlueprintProgressRepository
	questRepo                 *repository.QuestRepository
	hideoutModuleRepo         *repository.HideoutModuleRepository
	skillNodeRepo             *repository.SkillNodeRepository
	itemRepo                  *repository.ItemRepository
	userRepo                  *repository.UserRepository
}

func NewProgressHandler(
	questProgressRepo *repository.UserQuestProgressRepository,
	hideoutModuleProgressRepo *repository.UserHideoutModuleProgressRepository,
	skillNodeProgressRepo *repository.UserSkillNodeProgressRepository,
	blueprintProgressRepo *repository.UserBlueprintProgressRepository,
	questRepo *repository.QuestRepository,
	hideoutModuleRepo *repository.HideoutModuleRepository,
	skillNodeRepo *repository.SkillNodeRepository,
	itemRepo *repository.ItemRepository,
	userRepo *repository.UserRepository,
) *ProgressHandler {
	return &ProgressHandler{
		questProgressRepo:         questProgressRepo,
		hideoutModuleProgressRepo: hideoutModuleProgressRepo,
		skillNodeProgressRepo:     skillNodeProgressRepo,
		blueprintProgressRepo:     blueprintProgressRepo,
		questRepo:                 questRepo,
		hideoutModuleRepo:         hideoutModuleRepo,
		skillNodeRepo:             skillNodeRepo,
		itemRepo:                  itemRepo,
		userRepo:                  userRepo,
	}
}

// GetMyQuestProgress returns all quest progress for the current user
// GetMyQuestProgress returns all quest progress for the current user
// @Summary Get my quest progress
// @Description Fetch all quest completion states for the authenticated user.
// @Tags progress
// @Accept json
// @Produce json
// @Success 200 {object} map[string][]models.UserQuestProgress "Successfully fetched quest progress"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /progress/quests [get]
func (h *ProgressHandler) GetMyQuestProgress(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	userModel := user.(*models.User)

	progress, err := h.questProgressRepo.FindByUserID(userModel.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quest progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": progress})
}

// UpdateQuestProgress updates quest completion status for the current user
// Accepts external_id (e.g., "ss1") instead of internal database ID
// UpdateQuestProgress updates quest completion status for the current user
// @Summary Update my quest progress
// @Description Update the completion state of a specific quest using its external ID.
// @Tags progress
// @Accept json
// @Produce json
// @Param quest_id path string true "Quest External ID"
// @Param completion body map[string]bool true "Completion status (completed: true/false)"
// @Success 200 {object} models.UserQuestProgress "Successfully updated quest progress"
// @Failure 400 {object} ErrorResponse "Invalid input or ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 404 {object} ErrorResponse "Quest not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /progress/quests/{quest_id} [put]
func (h *ProgressHandler) UpdateQuestProgress(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	userModel := user.(*models.User)

	questExternalID := c.Param("quest_id")
	if questExternalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quest external_id is required"})
		return
	}

	// Look up quest by external_id
	quest, err := h.questRepo.FindByExternalID(questExternalID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quest not found"})
		return
	}

	var req struct {
		Completed bool `json:"completed" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	progress, err := h.questProgressRepo.Upsert(userModel.ID, quest.ID, req.Completed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quest progress"})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// GetMyHideoutModuleProgress returns all hideout module progress for the current user
// GetMyHideoutModuleProgress returns all hideout module progress for the current user
// @Summary Get my hideout module progress
// @Description Fetch all hideout module levels and unlock states for the authenticated user.
// @Tags progress
// @Accept json
// @Produce json
// @Success 200 {object} map[string][]models.UserHideoutModuleProgress "Successfully fetched hideout module progress"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /progress/hideout-modules [get]
func (h *ProgressHandler) GetMyHideoutModuleProgress(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	userModel := user.(*models.User)

	progress, err := h.hideoutModuleProgressRepo.FindByUserID(userModel.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch hideout module progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": progress})
}

// UpdateHideoutModuleProgress updates hideout module progress for the current user
// Accepts external_id (e.g., "module_001") instead of internal database ID
// UpdateHideoutModuleProgress updates hideout module progress for the current user
// @Summary Update my hideout module progress
// @Description Update the unlock state and level of a specific hideout module using its external ID.
// @Tags progress
// @Accept json
// @Produce json
// @Param module_id path string true "Module External ID"
// @Param progress body map[string]interface{} true "Progress data (unlocked, level)"
// @Success 200 {object} models.UserHideoutModuleProgress "Successfully updated hideout module progress"
// @Failure 400 {object} ErrorResponse "Invalid input or ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 404 {object} ErrorResponse "Module not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /progress/hideout-modules/{module_id} [put]
func (h *ProgressHandler) UpdateHideoutModuleProgress(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	userModel := user.(*models.User)

	moduleExternalID := c.Param("module_id")
	if moduleExternalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hideout module external_id is required"})
		return
	}

	// Look up hideout module by external_id
	module, err := h.hideoutModuleRepo.FindByExternalID(moduleExternalID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hideout module not found"})
		return
	}

	var req struct {
		Unlocked bool `json:"unlocked"`
		Level    int  `json:"level"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	progress, err := h.hideoutModuleProgressRepo.Upsert(userModel.ID, module.ID, req.Unlocked, req.Level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update hideout module progress"})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// GetMySkillNodeProgress returns all skill node progress for the current user
// GetMySkillNodeProgress returns all skill node progress for the current user
// @Summary Get my skill node progress
// @Description Fetch all skill node levels and unlock states for the authenticated user.
// @Tags progress
// @Accept json
// @Produce json
// @Success 200 {object} map[string][]models.UserSkillNodeProgress "Successfully fetched skill node progress"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /progress/skill-nodes [get]
func (h *ProgressHandler) GetMySkillNodeProgress(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	userModel := user.(*models.User)

	progress, err := h.skillNodeProgressRepo.FindByUserID(userModel.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch skill node progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": progress})
}

// UpdateSkillNodeProgress updates skill node progress for the current user
// Accepts external_id (e.g., "skill_001") instead of internal database ID
// UpdateSkillNodeProgress updates skill node progress for the current user
// @Summary Update my skill node progress
// @Description Update the unlock state and level of a specific skill node using its external ID.
// @Tags progress
// @Accept json
// @Produce json
// @Param skill_node_id path string true "Skill Node External ID"
// @Param progress body map[string]interface{} true "Progress data (unlocked, level)"
// @Success 200 {object} models.UserSkillNodeProgress "Successfully updated skill node progress"
// @Failure 400 {object} ErrorResponse "Invalid input or ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 404 {object} ErrorResponse "Skill node not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /progress/skill-nodes/{skill_node_id} [put]
func (h *ProgressHandler) UpdateSkillNodeProgress(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	userModel := user.(*models.User)

	skillNodeExternalID := c.Param("skill_node_id")
	if skillNodeExternalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Skill node external_id is required"})
		return
	}

	// Look up skill node by external_id
	skillNode, err := h.skillNodeRepo.FindByExternalID(skillNodeExternalID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Skill node not found"})
		return
	}

	var req struct {
		Unlocked bool `json:"unlocked"`
		Level    int  `json:"level"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	progress, err := h.skillNodeProgressRepo.Upsert(userModel.ID, skillNode.ID, req.Unlocked, req.Level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update skill node progress"})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// GetMyBlueprintProgress returns all blueprint progress for the current user
// GetMyBlueprintProgress returns all blueprint progress for the current user
// @Summary Get my blueprint progress
// @Description Fetch all blueprint consumption states for the authenticated user.
// @Tags progress
// @Accept json
// @Produce json
// @Success 200 {object} map[string][]models.UserBlueprintProgress "Successfully fetched blueprint progress"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /progress/blueprints [get]
func (h *ProgressHandler) GetMyBlueprintProgress(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	userModel := user.(*models.User)

	progress, err := h.blueprintProgressRepo.FindByUserID(userModel.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch blueprint progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": progress})
}

// UpdateBlueprintProgress updates blueprint consumption status for the current user
// Accepts external_id (e.g., "arc_motion_core") instead of internal database ID
// UpdateBlueprintProgress updates blueprint consumption status for the current user
// @Summary Update my blueprint progress
// @Description Update whether a specific blueprint has been consumed by the user.
// @Tags progress
// @Accept json
// @Produce json
// @Param item_id path string true "Item External ID (Blueprint)"
// @Param consumption body map[string]bool true "Consumption status (consumed: true/false)"
// @Success 200 {object} models.UserBlueprintProgress "Successfully updated blueprint progress"
// @Failure 400 {object} ErrorResponse "Invalid input or ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 404 {object} ErrorResponse "Blueprint not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /progress/blueprints/{item_id} [put]
func (h *ProgressHandler) UpdateBlueprintProgress(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	userModel := user.(*models.User)

	itemExternalID := c.Param("item_id")
	if itemExternalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Item external_id is required"})
		return
	}

	// Look up item (blueprint) by external_id
	item, err := h.itemRepo.FindByExternalID(itemExternalID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blueprint not found"})
		return
	}

	var req struct {
		Consumed bool `json:"consumed" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	progress, err := h.blueprintProgressRepo.Upsert(userModel.ID, item.ID, req.Consumed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update blueprint progress"})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// ========================================
// ADMIN ENDPOINTS - View/Manage All Users
// ========================================

// GetUserQuestProgress returns quest progress for a specific user (admin only)
// GetUserQuestProgress returns quest progress for a specific user (admin only)
// @Summary Get user quest progress
// @Description Fetch quest completion states for any user by their ID. Only admins can access this.
// @Tags management
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "Successfully fetched quest progress"
// @Failure 400 {object} ErrorResponse "Invalid user ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /admin/users/{id}/progress/quests [get]
func (h *ProgressHandler) GetUserQuestProgress(c *gin.Context) {
userIDStr := c.Param("id")
userID, err := parseUint(userIDStr)
if err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
return
}

// Verify user exists
if _, err := h.userRepo.FindByID(userID); err != nil {
c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
return
}

progress, err := h.questProgressRepo.FindByUserID(userID)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quest progress"})
return
}

c.JSON(http.StatusOK, gin.H{"data": progress, "user_id": userID})
}

// UpdateUserQuestProgress updates quest progress for a specific user (admin only)
// UpdateUserQuestProgress updates quest progress for a specific user (admin only)
// @Summary Update user quest progress
// @Description Update the completion state of a quest for any user. Only admins can perform this.
// @Tags management
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param quest_id path string true "Quest External ID"
// @Param completion body map[string]bool true "Completion status"
// @Success 200 {object} models.UserQuestProgress "Successfully updated user quest progress"
// @Failure 400 {object} ErrorResponse "Invalid input or ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Failure 404 {object} ErrorResponse "User or Quest not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /admin/users/{id}/progress/quests/{quest_id} [put]
func (h *ProgressHandler) UpdateUserQuestProgress(c *gin.Context) {
userIDStr := c.Param("id")
userID, err := parseUint(userIDStr)
if err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
return
}

questExternalID := c.Param("quest_id")
if questExternalID == "" {
c.JSON(http.StatusBadRequest, gin.H{"error": "Quest external_id is required"})
return
}

// Verify user exists
if _, err := h.userRepo.FindByID(userID); err != nil {
c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
return
}

// Look up quest by external_id
quest, err := h.questRepo.FindByExternalID(questExternalID)
if err != nil {
c.JSON(http.StatusNotFound, gin.H{"error": "Quest not found"})
return
}

var req struct {
Completed bool `json:"completed" binding:"required"`
}

if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
return
}

progress, err := h.questProgressRepo.Upsert(userID, quest.ID, req.Completed)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quest progress"})
return
}

c.JSON(http.StatusOK, progress)
}

// GetUserHideoutModuleProgress returns hideout module progress for a specific user (admin only)
// GetUserHideoutModuleProgress returns hideout module progress for a specific user (admin only)
// @Summary Get user hideout module progress
// @Description Fetch hideout module levels for any user. Only admins can access this.
// @Tags management
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "Successfully fetched hideout module progress"
// @Failure 400 {object} ErrorResponse "Invalid user ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /admin/users/{id}/progress/hideout-modules [get]
func (h *ProgressHandler) GetUserHideoutModuleProgress(c *gin.Context) {
userIDStr := c.Param("id")
userID, err := parseUint(userIDStr)
if err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
return
}

// Verify user exists
if _, err := h.userRepo.FindByID(userID); err != nil {
c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
return
}

progress, err := h.hideoutModuleProgressRepo.FindByUserID(userID)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch hideout module progress"})
return
}

c.JSON(http.StatusOK, gin.H{"data": progress, "user_id": userID})
}

// UpdateUserHideoutModuleProgress updates hideout module progress for a specific user (admin only)
// UpdateUserHideoutModuleProgress updates hideout module progress for a specific user (admin only)
// @Summary Update user hideout module progress
// @Description Update module level/unlock state for any user. Only admins can perform this.
// @Tags management
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param module_id path string true "Module External ID"
// @Param progress body map[string]interface{} true "Progress data"
// @Success 200 {object} models.UserHideoutModuleProgress "Successfully updated user module progress"
// @Failure 400 {object} ErrorResponse "Invalid input or ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Failure 404 {object} ErrorResponse "User or Module not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /admin/users/{id}/progress/hideout-modules/{module_id} [put]
func (h *ProgressHandler) UpdateUserHideoutModuleProgress(c *gin.Context) {
userIDStr := c.Param("id")
userID, err := parseUint(userIDStr)
if err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
return
}

moduleExternalID := c.Param("module_id")
if moduleExternalID == "" {
c.JSON(http.StatusBadRequest, gin.H{"error": "Hideout module external_id is required"})
return
}

// Verify user exists
if _, err := h.userRepo.FindByID(userID); err != nil {
c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
return
}

// Look up hideout module by external_id
module, err := h.hideoutModuleRepo.FindByExternalID(moduleExternalID)
if err != nil {
c.JSON(http.StatusNotFound, gin.H{"error": "Hideout module not found"})
return
}

var req struct {
Unlocked bool `json:"unlocked"`
Level    int  `json:"level"`
}

if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
return
}

progress, err := h.hideoutModuleProgressRepo.Upsert(userID, module.ID, req.Unlocked, req.Level)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update hideout module progress"})
return
}

c.JSON(http.StatusOK, progress)
}

// GetUserSkillNodeProgress returns skill node progress for a specific user (admin only)
// GetUserSkillNodeProgress returns skill node progress for a specific user (admin only)
// @Summary Get user skill node progress
// @Description Fetch skill node levels for any user. Only admins can access this.
// @Tags management
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "Successfully fetched skill node progress"
// @Failure 400 {object} ErrorResponse "Invalid user ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /admin/users/{id}/progress/skill-nodes [get]
func (h *ProgressHandler) GetUserSkillNodeProgress(c *gin.Context) {
userIDStr := c.Param("id")
userID, err := parseUint(userIDStr)
if err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
return
}

// Verify user exists
if _, err := h.userRepo.FindByID(userID); err != nil {
c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
return
}

progress, err := h.skillNodeProgressRepo.FindByUserID(userID)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch skill node progress"})
return
}

c.JSON(http.StatusOK, gin.H{"data": progress, "user_id": userID})
}

// UpdateUserSkillNodeProgress updates skill node progress for a specific user (admin only)
// UpdateUserSkillNodeProgress updates skill node progress for a specific user (admin only)
// @Summary Update user skill node progress
// @Description Update skill node level/unlock state for any user. Only admins can perform this.
// @Tags management
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param skill_node_id path string true "Skill Node External ID"
// @Param progress body map[string]interface{} true "Progress data"
// @Success 200 {object} models.UserSkillNodeProgress "Successfully updated user skill node progress"
// @Failure 400 {object} ErrorResponse "Invalid input or ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Failure 404 {object} ErrorResponse "User or Skill Node not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /admin/users/{id}/progress/skill-nodes/{skill_node_id} [put]
func (h *ProgressHandler) UpdateUserSkillNodeProgress(c *gin.Context) {
userIDStr := c.Param("id")
userID, err := parseUint(userIDStr)
if err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
return
}

skillNodeExternalID := c.Param("skill_node_id")
if skillNodeExternalID == "" {
c.JSON(http.StatusBadRequest, gin.H{"error": "Skill node external_id is required"})
return
}

// Verify user exists
if _, err := h.userRepo.FindByID(userID); err != nil {
c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
return
}

// Look up skill node by external_id
skillNode, err := h.skillNodeRepo.FindByExternalID(skillNodeExternalID)
if err != nil {
c.JSON(http.StatusNotFound, gin.H{"error": "Skill node not found"})
return
}

var req struct {
Unlocked bool `json:"unlocked"`
Level    int  `json:"level"`
}

if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
return
}

progress, err := h.skillNodeProgressRepo.Upsert(userID, skillNode.ID, req.Unlocked, req.Level)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update skill node progress"})
return
}

c.JSON(http.StatusOK, progress)
}

// GetUserBlueprintProgress returns blueprint progress for a specific user (admin only)
// GetUserBlueprintProgress returns blueprint progress for a specific user (admin only)
// @Summary Get user blueprint progress
// @Description Fetch blueprint consumption states for any user. Only admins can access this.
// @Tags management
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "Successfully fetched blueprint progress"
// @Failure 400 {object} ErrorResponse "Invalid user ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /admin/users/{id}/progress/blueprints [get]
func (h *ProgressHandler) GetUserBlueprintProgress(c *gin.Context) {
userIDStr := c.Param("id")
userID, err := parseUint(userIDStr)
if err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
return
}

// Verify user exists
if _, err := h.userRepo.FindByID(userID); err != nil {
c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
return
}

progress, err := h.blueprintProgressRepo.FindByUserID(userID)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch blueprint progress"})
return
}

c.JSON(http.StatusOK, gin.H{"data": progress, "user_id": userID})
}

// UpdateUserBlueprintProgress updates blueprint progress for a specific user (admin only)
// UpdateUserBlueprintProgress updates blueprint progress for a specific user (admin only)
// @Summary Update user blueprint progress
// @Description Update blueprint consumption state for any user. Only admins can perform this.
// @Tags management
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param item_id path string true "Item External ID (Blueprint)"
// @Param consumption body map[string]bool true "Consumption status"
// @Success 200 {object} models.UserBlueprintProgress "Successfully updated user blueprint progress"
// @Failure 400 {object} ErrorResponse "Invalid input or ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Failure 404 {object} ErrorResponse "User or Blueprint not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /admin/users/{id}/progress/blueprints/{item_id} [put]
func (h *ProgressHandler) UpdateUserBlueprintProgress(c *gin.Context) {
userIDStr := c.Param("id")
userID, err := parseUint(userIDStr)
if err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
return
}

itemExternalID := c.Param("item_id")
if itemExternalID == "" {
c.JSON(http.StatusBadRequest, gin.H{"error": "Item external_id is required"})
return
}

// Verify user exists
if _, err := h.userRepo.FindByID(userID); err != nil {
c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
return
}

// Look up item (blueprint) by external_id
item, err := h.itemRepo.FindByExternalID(itemExternalID)
if err != nil {
c.JSON(http.StatusNotFound, gin.H{"error": "Blueprint not found"})
return
}

var req struct {
Consumed bool `json:"consumed" binding:"required"`
}

if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
return
}

progress, err := h.blueprintProgressRepo.Upsert(userID, item.ID, req.Consumed)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update blueprint progress"})
return
}

c.JSON(http.StatusOK, progress)
}

// GetAllUserProgress returns all progress types for a specific user (admin only)
// GetAllUserProgress returns all progress types for a specific user (admin only)
// @Summary Get all user progress
// @Description Fetch a comprehensive summary of all progress types (quests, modules, skills, blueprints) for any user. Only admins can access this.
// @Tags management
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "Successfully fetched comprehensive progress"
// @Failure 400 {object} ErrorResponse "Invalid user ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /admin/users/{id}/progress [get]
func (h *ProgressHandler) GetAllUserProgress(c *gin.Context) {
userIDStr := c.Param("id")
userID, err := parseUint(userIDStr)
if err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
return
}

// Verify user exists
user, err := h.userRepo.FindByID(userID)
if err != nil {
c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
return
}

// Fetch all progress types in parallel
type result struct {
quests         []models.UserQuestProgress
hideoutModules []models.UserHideoutModuleProgress
skillNodes     []models.UserSkillNodeProgress
blueprints     []models.UserBlueprintProgress
err            error
}

resultChan := make(chan result, 1)

go func() {
defer func() {
if r := recover(); r != nil {
log.Printf("PANIC recovered in GetAllUserProgress goroutine: %v", r)
resultChan <- result{err: http.ErrAbortHandler}
}
}()

var r result
var err error

r.quests, err = h.questProgressRepo.FindByUserID(userID)
if err != nil {
log.Printf("Warning: Failed to fetch quest progress for user %d: %v", userID, err)
}

r.hideoutModules, err = h.hideoutModuleProgressRepo.FindByUserID(userID)
if err != nil {
log.Printf("Warning: Failed to fetch hideout module progress for user %d: %v", userID, err)
}

r.skillNodes, err = h.skillNodeProgressRepo.FindByUserID(userID)
if err != nil {
log.Printf("Warning: Failed to fetch skill node progress for user %d: %v", userID, err)
}

r.blueprints, err = h.blueprintProgressRepo.FindByUserID(userID)
if err != nil {
log.Printf("Warning: Failed to fetch blueprint progress for user %d: %v", userID, err)
}

resultChan <- r
}()

r := <-resultChan

// Check if goroutine panicked
if r.err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user progress"})
return
}

c.JSON(http.StatusOK, gin.H{
"user": gin.H{
"id":       user.ID,
"username": user.Username,
"email":    user.Email,
},
"progress": gin.H{
"quests":          r.quests,
"hideout_modules": r.hideoutModules,
"skill_nodes":     r.skillNodes,
"blueprints":      r.blueprints,
},
})
}

// Helper function to parse uint from string
func parseUint(s string) (uint, error) {
id, err := strconv.ParseUint(s, 10, 32)
if err != nil {
return 0, err
}
return uint(id), nil
}
