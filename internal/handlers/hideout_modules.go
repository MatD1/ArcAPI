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

func (h *HideoutModuleHandler) List(c *gin.Context) {
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
