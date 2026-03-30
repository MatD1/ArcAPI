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

// List returns all bots (paginated)
// @Summary List bots
// @Description Fetch bots with optional pagination (offset/limit)
// @Tags bots
// @Accept json
// @Produce json
// @Param offset query int false "Offset" default(0)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} PaginatedResponse{data=[]models.Bot} "Successfully fetched bots"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /bots [get]
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

// Get returns a single bot by ID
// @Summary Get a single bot
// @Description Fetch a bot by its numeric ID
// @Tags bots
// @Accept json
// @Produce json
// @Param id path int true "Bot ID"
// @Success 200 {object} models.Bot "Successfully fetched the bot"
// @Failure 400 {object} ErrorResponse "Invalid bot ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 404 {object} ErrorResponse "Bot not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /bots/{id} [get]
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

// List returns all maps (paginated)
// @Summary List maps
// @Description Fetch maps with optional pagination
// @Tags maps
// @Accept json
// @Produce json
// @Param offset query int false "Offset" default(0)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} PaginatedResponse{data=[]models.Map} "Successfully fetched maps"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /maps [get]
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

// Get returns a single map by ID
// @Summary Get a single map
// @Description Fetch a map by its numeric ID
// @Tags maps
// @Accept json
// @Produce json
// @Param id path int true "Map ID"
// @Success 200 {object} models.Map "Successfully fetched the map"
// @Failure 400 {object} ErrorResponse "Invalid map ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 404 {object} ErrorResponse "Map not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /maps/{id} [get]
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

// List returns all traders (paginated)
// @Summary List traders
// @Description Fetch traders with optional pagination
// @Tags traders
// @Accept json
// @Produce json
// @Param offset query int false "Offset" default(0)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} PaginatedResponse{data=[]models.Trader} "Successfully fetched traders"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /traders [get]
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

// Get returns a single trader by ID
// @Summary Get a single trader
// @Description Fetch a trader by its numeric ID
// @Tags traders
// @Accept json
// @Produce json
// @Param id path int true "Trader ID"
// @Success 200 {object} models.Trader "Successfully fetched the trader"
// @Failure 400 {object} ErrorResponse "Invalid trader ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 404 {object} ErrorResponse "Trader not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /traders/{id} [get]
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

// List returns all projects (paginated)
// @Summary List projects
// @Description Fetch projects with optional pagination
// @Tags projects
// @Accept json
// @Produce json
// @Param offset query int false "Offset" default(0)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} PaginatedResponse{data=[]models.Project} "Successfully fetched projects"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /projects [get]
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

// Get returns a single project by ID
// @Summary Get a single project
// @Description Fetch a project by its numeric ID
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Success 200 {object} models.Project "Successfully fetched the project"
// @Failure 400 {object} ErrorResponse "Invalid project ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 404 {object} ErrorResponse "Project not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /projects/{id} [get]
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

