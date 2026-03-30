package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
	"github.com/mat/arcapi/internal/services"
)

type QuestHandler struct {
	repo             *repository.QuestRepository
	dataCacheService *services.DataCacheService
}

func NewQuestHandler(repo *repository.QuestRepository) *QuestHandler {
	return &QuestHandler{repo: repo}
}

func NewQuestHandlerWithCache(repo *repository.QuestRepository, dataCacheService *services.DataCacheService) *QuestHandler {
	return &QuestHandler{
		repo:             repo,
		dataCacheService: dataCacheService,
	}
}

// List returns all quests
// @Summary List all quests
// @Description Fetch all quests from the database or cache
// @Tags quests
// @Accept json
// @Produce json
// @Success 200 {object} PaginatedResponse{data=[]models.Quest} "Successfully fetched quests"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /quests [get]
func (h *QuestHandler) List(c *gin.Context) {
	// Return all quests without pagination
	var quests []models.Quest
	var count int64
	var err error

	// Use cache service if available
	if h.dataCacheService != nil {
		quests, count, err = h.dataCacheService.GetQuests()
	} else {
		// Fallback to direct database query
		quests, count, err = h.repo.FindAll(0, 1000000)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  quests,
		"total": count,
	})
}

// Get returns a single quest by ID
// @Summary Get a single quest
// @Description Fetch a quest by its numeric ID
// @Tags quests
// @Accept json
// @Produce json
// @Param id path int true "Quest ID"
// @Success 200 {object} models.Quest "Successfully fetched the quest"
// @Failure 400 {object} ErrorResponse "Invalid quest ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 404 {object} ErrorResponse "Quest not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /quests/{id} [get]
func (h *QuestHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quest ID"})
		return
	}

	quest, err := h.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quest not found"})
		return
	}

	c.JSON(http.StatusOK, quest)
}

// Create adds a new quest
// @Summary Create a quest
// @Description Add a new quest to the database
// @Tags quests
// @Accept json
// @Produce json
// @Param quest body models.Quest true "Quest object"
// @Success 201 {object} models.Quest "Successfully created the quest"
// @Failure 400 {object} ErrorResponse "Invalid input data"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /quests [post]
func (h *QuestHandler) Create(c *gin.Context) {
	var quest models.Quest
	if err := c.ShouldBindJSON(&quest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if quest.ExternalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "external_id is required"})
		return
	}

	err := h.repo.Create(&quest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create quest"})
		return
	}

	// Invalidate cache on create
	if h.dataCacheService != nil {
		h.dataCacheService.InvalidateQuestsCache()
	}

	c.JSON(http.StatusCreated, quest)
}

// Update modifies an existing quest
// @Summary Update a quest
// @Description Update an existing quest by its ID
// @Tags quests
// @Accept json
// @Produce json
// @Param id path int true "Quest ID"
// @Param quest body models.Quest true "Updated quest object"
// @Success 200 {object} models.Quest "Successfully updated the quest"
// @Failure 400 {object} ErrorResponse "Invalid input or ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /quests/{id} [put]
func (h *QuestHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quest ID"})
		return
	}

	var quest models.Quest
	if err := c.ShouldBindJSON(&quest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	quest.ID = uint(id)
	err = h.repo.Update(&quest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quest"})
		return
	}

	// Invalidate cache on update
	if h.dataCacheService != nil {
		h.dataCacheService.InvalidateQuestsCache()
	}

	c.JSON(http.StatusOK, quest)
}

// Delete removes a quest
// @Summary Delete a quest
// @Description Delete an existing quest by its ID
// @Tags quests
// @Accept json
// @Produce json
// @Param id path int true "Quest ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse "Invalid quest ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /quests/{id} [delete]
func (h *QuestHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quest ID"})
		return
	}

	err = h.repo.Delete(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete quest"})
		return
	}

	// Invalidate cache on delete
	if h.dataCacheService != nil {
		h.dataCacheService.InvalidateQuestsCache()
	}

	c.JSON(http.StatusNoContent, nil)
}
