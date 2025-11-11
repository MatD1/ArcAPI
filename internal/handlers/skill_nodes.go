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
