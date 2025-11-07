package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
)

type EnemyTypeHandler struct {
	repo *repository.EnemyTypeRepository
}

func NewEnemyTypeHandler(repo *repository.EnemyTypeRepository) *EnemyTypeHandler {
	return &EnemyTypeHandler{repo: repo}
}

func (h *EnemyTypeHandler) List(c *gin.Context) {
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
	enemyTypes, count, err := h.repo.FindAll(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch enemy types"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": enemyTypes,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": count,
		},
	})
}

func (h *EnemyTypeHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid enemy type ID"})
		return
	}

	enemyType, err := h.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Enemy type not found"})
		return
	}

	c.JSON(http.StatusOK, enemyType)
}

func (h *EnemyTypeHandler) Create(c *gin.Context) {
	var enemyType models.EnemyType
	if err := c.ShouldBindJSON(&enemyType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if enemyType.ExternalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "external_id is required"})
		return
	}

	err := h.repo.Create(&enemyType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create enemy type"})
		return
	}

	c.JSON(http.StatusCreated, enemyType)
}

func (h *EnemyTypeHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid enemy type ID"})
		return
	}

	var enemyType models.EnemyType
	if err := c.ShouldBindJSON(&enemyType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	enemyType.ID = uint(id)
	err = h.repo.Update(&enemyType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update enemy type"})
		return
	}

	c.JSON(http.StatusOK, enemyType)
}

func (h *EnemyTypeHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid enemy type ID"})
		return
	}

	err = h.repo.Delete(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete enemy type"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
