package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
)

type ProgressHandler struct {
	questProgressRepo         *repository.UserQuestProgressRepository
	hideoutModuleProgressRepo *repository.UserHideoutModuleProgressRepository
	skillNodeProgressRepo     *repository.UserSkillNodeProgressRepository
	questRepo                 *repository.QuestRepository
	hideoutModuleRepo         *repository.HideoutModuleRepository
	skillNodeRepo             *repository.SkillNodeRepository
}

func NewProgressHandler(
	questProgressRepo *repository.UserQuestProgressRepository,
	hideoutModuleProgressRepo *repository.UserHideoutModuleProgressRepository,
	skillNodeProgressRepo *repository.UserSkillNodeProgressRepository,
	questRepo *repository.QuestRepository,
	hideoutModuleRepo *repository.HideoutModuleRepository,
	skillNodeRepo *repository.SkillNodeRepository,
) *ProgressHandler {
	return &ProgressHandler{
		questProgressRepo:         questProgressRepo,
		hideoutModuleProgressRepo: hideoutModuleProgressRepo,
		skillNodeProgressRepo:     skillNodeProgressRepo,
		questRepo:                 questRepo,
		hideoutModuleRepo:         hideoutModuleRepo,
		skillNodeRepo:             skillNodeRepo,
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
// Accepts external_id (e.g., "ss1") instead of internal database ID
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
		Unlocked bool `json:"unlocked" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	progress, err := h.skillNodeProgressRepo.Upsert(userModel.ID, skillNode.ID, req.Unlocked)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update skill node progress"})
		return
	}

	c.JSON(http.StatusOK, progress)
}
