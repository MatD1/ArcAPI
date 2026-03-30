package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
)

type HideoutModuleHandler struct {
	repo *repository.HideoutModuleRepository
}

func NewHideoutModuleHandler(repo *repository.HideoutModuleRepository) *HideoutModuleHandler {
	return &HideoutModuleHandler{repo: repo}
}

// List returns all hideout modules (paginated)
// @Summary List hideout modules
// @Description Fetch hideout modules with optional pagination. If ?all=true is passed, returns all modules unpaginated.
// @Tags hideout-modules
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param all query bool false "Return all modules" default(false)
// @Success 200 {object} PaginatedResponse{data=[]models.HideoutModule} "Successfully fetched hideout modules"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /hideout-modules [get]
func (h *HideoutModuleHandler) List(c *gin.Context) {
	if c.Query("all") == "true" {
		h.ListAll(c)
		return
	}

	page := 1
	limit := 20

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
	hideoutModules, count, err := h.repo.FindAll(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch hideout modules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": hideoutModules,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": count,
		},
	})
}

func (h *HideoutModuleHandler) ListAll(c *gin.Context) {
	hideoutModules, count, err := h.repo.FindAll(0, 999999)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch hideout modules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  hideoutModules,
		"total": count,
	})
}

// Get returns a single hideout module by ID
// @Summary Get a single hideout module
// @Description Fetch a hideout module by its numeric ID
// @Tags hideout-modules
// @Accept json
// @Produce json
// @Param id path int true "Hideout Module ID"
// @Success 200 {object} models.HideoutModule "Successfully fetched the hideout module"
// @Failure 400 {object} ErrorResponse "Invalid hideout module ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 404 {object} ErrorResponse "Hideout module not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /hideout-modules/{id} [get]
func (h *HideoutModuleHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hideout module ID"})
		return
	}

	hideoutModule, err := h.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hideout module not found"})
		return
	}

	c.JSON(http.StatusOK, hideoutModule)
}

// Create adds a new hideout module
// @Summary Create a hideout module
// @Description Add a new hideout module to the database
// @Tags hideout-modules
// @Accept json
// @Produce json
// @Param hideoutModule body models.HideoutModule true "Hideout Module object"
// @Success 201 {object} models.HideoutModule "Successfully created the hideout module"
// @Failure 400 {object} ErrorResponse "Invalid input data"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /hideout-modules [post]
func (h *HideoutModuleHandler) Create(c *gin.Context) {
	var hideoutModule models.HideoutModule
	if err := c.ShouldBindJSON(&hideoutModule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if hideoutModule.ExternalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "external_id is required"})
		return
	}

	err := h.repo.Create(&hideoutModule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create hideout module"})
		return
	}

	c.JSON(http.StatusCreated, hideoutModule)
}

// Update modifies an existing hideout module
// @Summary Update a hideout module
// @Description Update an existing hideout module by its ID
// @Tags hideout-modules
// @Accept json
// @Produce json
// @Param id path int true "Hideout Module ID"
// @Param hideoutModule body models.HideoutModule true "Updated hideout module object"
// @Success 200 {object} models.HideoutModule "Successfully updated the hideout module"
// @Failure 400 {object} ErrorResponse "Invalid input or ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /hideout-modules/{id} [put]
func (h *HideoutModuleHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hideout module ID"})
		return
	}

	var hideoutModule models.HideoutModule
	if err := c.ShouldBindJSON(&hideoutModule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hideoutModule.ID = uint(id)
	err = h.repo.Update(&hideoutModule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update hideout module"})
		return
	}

	c.JSON(http.StatusOK, hideoutModule)
}

// Delete removes a hideout module
// @Summary Delete a hideout module
// @Description Delete an existing hideout module by its ID
// @Tags hideout-modules
// @Accept json
// @Produce json
// @Param id path int true "Hideout Module ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse "Invalid hideout module ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /hideout-modules/{id} [delete]
func (h *HideoutModuleHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hideout module ID"})
		return
	}

	err = h.repo.Delete(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete hideout module"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
