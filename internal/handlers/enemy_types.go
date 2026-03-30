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

// List returns all enemy types (paginated)
// @Summary List enemy types
// @Description Fetch enemy types with optional pagination
// @Tags enemy-types
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} PaginatedResponse{data=[]models.EnemyType} "Successfully fetched enemy types"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /enemy-types [get]
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

// Get returns a single enemy type by ID
// @Summary Get a single enemy type
// @Description Fetch an enemy type by its numeric ID
// @Tags enemy-types
// @Accept json
// @Produce json
// @Param id path int true "Enemy Type ID"
// @Success 200 {object} models.EnemyType "Successfully fetched the enemy type"
// @Failure 400 {object} ErrorResponse "Invalid enemy type ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 404 {object} ErrorResponse "Enemy type not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /enemy-types/{id} [get]
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

// Create adds a new enemy type
// @Summary Create an enemy type
// @Description Add a new enemy type to the database
// @Tags enemy-types
// @Accept json
// @Produce json
// @Param enemyType body models.EnemyType true "Enemy Type object"
// @Success 201 {object} models.EnemyType "Successfully created the enemy type"
// @Failure 400 {object} ErrorResponse "Invalid input data"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /enemy-types [post]
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

// Update modifies an existing enemy type
// @Summary Update an enemy type
// @Description Update an existing enemy type by its ID
// @Tags enemy-types
// @Accept json
// @Produce json
// @Param id path int true "Enemy Type ID"
// @Param enemyType body models.EnemyType true "Updated enemy type object"
// @Success 200 {object} models.EnemyType "Successfully updated the enemy type"
// @Failure 400 {object} ErrorResponse "Invalid input or ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /enemy-types/{id} [put]
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

// Delete removes an enemy type
// @Summary Delete an enemy type
// @Description Delete an existing enemy type by its ID
// @Tags enemy-types
// @Accept json
// @Produce json
// @Param id path int true "Enemy Type ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse "Invalid enemy type ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /enemy-types/{id} [delete]
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
