package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
)

type SkillNodeHandler struct {
	repo *repository.SkillNodeRepository
}

func NewSkillNodeHandler(repo *repository.SkillNodeRepository) *SkillNodeHandler {
	return &SkillNodeHandler{repo: repo}
}

// List returns all skill nodes (paginated)
// @Summary List skill nodes
// @Description Fetch skill nodes with optional pagination. If ?all=true is passed, returns all skill nodes unpaginated.
// @Tags skill-nodes
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param all query bool false "Return all nodes" default(false)
// @Success 200 {object} PaginatedResponse{data=[]models.SkillNode} "Successfully fetched skill nodes"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /skill-nodes [get]
func (h *SkillNodeHandler) List(c *gin.Context) {
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
	skillNodes, count, err := h.repo.FindAll(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch skill nodes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": skillNodes,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": count,
		},
	})
}

func (h *SkillNodeHandler) ListAll(c *gin.Context) {
	skillNodes, count, err := h.repo.FindAll(0, 999999)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch skill nodes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  skillNodes,
		"total": count,
	})
}

// Get returns a single skill node by ID
// @Summary Get a single skill node
// @Description Fetch a skill node by its numeric ID
// @Tags skill-nodes
// @Accept json
// @Produce json
// @Param id path int true "Skill Node ID"
// @Success 200 {object} models.SkillNode "Successfully fetched the skill node"
// @Failure 400 {object} ErrorResponse "Invalid skill node ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 404 {object} ErrorResponse "Skill node not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /skill-nodes/{id} [get]
func (h *SkillNodeHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid skill node ID"})
		return
	}

	skillNode, err := h.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Skill node not found"})
		return
	}

	c.JSON(http.StatusOK, skillNode)
}

// Create adds a new skill node
// @Summary Create a skill node
// @Description Add a new skill node to the database
// @Tags skill-nodes
// @Accept json
// @Produce json
// @Param skillNode body models.SkillNode true "Skill Node object"
// @Success 201 {object} models.SkillNode "Successfully created the skill node"
// @Failure 400 {object} ErrorResponse "Invalid input data"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /skill-nodes [post]
func (h *SkillNodeHandler) Create(c *gin.Context) {
	var skillNode models.SkillNode
	if err := c.ShouldBindJSON(&skillNode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if skillNode.ExternalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "external_id is required"})
		return
	}

	err := h.repo.Create(&skillNode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create skill node"})
		return
	}

	c.JSON(http.StatusCreated, skillNode)
}

// Update modifies an existing skill node
// @Summary Update a skill node
// @Description Update an existing skill node by its ID
// @Tags skill-nodes
// @Accept json
// @Produce json
// @Param id path int true "Skill Node ID"
// @Param skillNode body models.SkillNode true "Updated skill node object"
// @Success 200 {object} models.SkillNode "Successfully updated the skill node"
// @Failure 400 {object} ErrorResponse "Invalid input or ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /skill-nodes/{id} [put]
func (h *SkillNodeHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid skill node ID"})
		return
	}

	var skillNode models.SkillNode
	if err := c.ShouldBindJSON(&skillNode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	skillNode.ID = uint(id)
	err = h.repo.Update(&skillNode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update skill node"})
		return
	}

	c.JSON(http.StatusOK, skillNode)
}

// Delete removes a skill node
// @Summary Delete a skill node
// @Description Delete an existing skill node by its ID
// @Tags skill-nodes
// @Accept json
// @Produce json
// @Param id path int true "Skill Node ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse "Invalid skill node ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /skill-nodes/{id} [delete]
func (h *SkillNodeHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid skill node ID"})
		return
	}

	err = h.repo.Delete(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete skill node"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
