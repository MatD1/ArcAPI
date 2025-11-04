package handlers

import (
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
}

func NewProgressHandler(
	questProgressRepo *repository.UserQuestProgressRepository,
	hideoutModuleProgressRepo *repository.UserHideoutModuleProgressRepository,
	skillNodeProgressRepo *repository.UserSkillNodeProgressRepository,
) *ProgressHandler {
	return &ProgressHandler{
		questProgressRepo:         questProgressRepo,
		hideoutModuleProgressRepo: hideoutModuleProgressRepo,
		skillNodeProgressRepo:     skillNodeProgressRepo,
	}
}

// GetMyQuestProgress returns all quest progress for the current user
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
func (h *ProgressHandler) UpdateQuestProgress(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	userModel := user.(*models.User)

	questIDStr := c.Param("quest_id")
	questID, err := strconv.ParseUint(questIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quest ID"})
		return
	}

	var req struct {
		Completed bool `json:"completed" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	progress, err := h.questProgressRepo.Upsert(userModel.ID, uint(questID), req.Completed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quest progress"})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// GetMyHideoutModuleProgress returns all hideout module progress for the current user
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
func (h *ProgressHandler) UpdateHideoutModuleProgress(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	userModel := user.(*models.User)

	moduleIDStr := c.Param("module_id")
	moduleID, err := strconv.ParseUint(moduleIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hideout module ID"})
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

	progress, err := h.hideoutModuleProgressRepo.Upsert(userModel.ID, uint(moduleID), req.Unlocked, req.Level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update hideout module progress"})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// GetMySkillNodeProgress returns all skill node progress for the current user
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
func (h *ProgressHandler) UpdateSkillNodeProgress(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	userModel := user.(*models.User)

	skillNodeIDStr := c.Param("skill_node_id")
	skillNodeID, err := strconv.ParseUint(skillNodeIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid skill node ID"})
		return
	}

	var req struct {
		Unlocked bool `json:"unlocked" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	progress, err := h.skillNodeProgressRepo.Upsert(userModel.ID, uint(skillNodeID), req.Unlocked)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update skill node progress"})
		return
	}

	c.JSON(http.StatusOK, progress)
}
