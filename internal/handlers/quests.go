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
