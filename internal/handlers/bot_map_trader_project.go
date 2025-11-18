package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/repository"
)

// Bot Handler
type BotHandler struct {
	repo *repository.BotRepository
}

func NewBotHandler(repo *repository.BotRepository) *BotHandler {
	return &BotHandler{repo: repo}
}

func (h *BotHandler) List(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	bots, count, err := h.repo.FindAll(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bots"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": bots,
		"pagination": gin.H{
			"total":  count,
			"offset": offset,
			"limit":  limit,
		},
	})
}

func (h *BotHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bot ID"})
		return
	}

	bot, err := h.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bot not found"})
		return
	}

	c.JSON(http.StatusOK, bot)
}

// Map Handler
type MapHandler struct {
	repo *repository.MapRepository
}

func NewMapHandler(repo *repository.MapRepository) *MapHandler {
	return &MapHandler{repo: repo}
}

func (h *MapHandler) List(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	maps, count, err := h.repo.FindAll(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch maps"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": maps,
		"pagination": gin.H{
			"total":  count,
			"offset": offset,
			"limit":  limit,
		},
	})
}

func (h *MapHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid map ID"})
		return
	}

	mapModel, err := h.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Map not found"})
		return
	}

	c.JSON(http.StatusOK, mapModel)
}

// Trader Handler
type TraderHandler struct {
	repo *repository.TraderRepository
}

func NewTraderHandler(repo *repository.TraderRepository) *TraderHandler {
	return &TraderHandler{repo: repo}
}

func (h *TraderHandler) List(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	traders, count, err := h.repo.FindAll(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch traders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": traders,
		"pagination": gin.H{
			"total":  count,
			"offset": offset,
			"limit":  limit,
		},
	})
}

func (h *TraderHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid trader ID"})
		return
	}

	trader, err := h.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trader not found"})
		return
	}

	c.JSON(http.StatusOK, trader)
}

// Project Handler
type ProjectHandler struct {
	repo *repository.ProjectRepository
}

func NewProjectHandler(repo *repository.ProjectRepository) *ProjectHandler {
	return &ProjectHandler{repo: repo}
}

func (h *ProjectHandler) List(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	projects, count, err := h.repo.FindAll(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": projects,
		"pagination": gin.H{
			"total":  count,
			"offset": offset,
			"limit":  limit,
		},
	})
}

func (h *ProjectHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	project, err := h.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	c.JSON(http.StatusOK, project)
}

