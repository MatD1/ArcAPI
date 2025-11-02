package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
)

type MissionHandler struct {
	repo *repository.MissionRepository
}

func NewMissionHandler(repo *repository.MissionRepository) *MissionHandler {
	return &MissionHandler{repo: repo}
}

func (h *MissionHandler) List(c *gin.Context) {
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
	missions, count, err := h.repo.FindAll(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch missions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": missions,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": count,
		},
	})
}

func (h *MissionHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mission ID"})
		return
	}

	mission, err := h.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mission not found"})
		return
	}

	c.JSON(http.StatusOK, mission)
}

func (h *MissionHandler) Create(c *gin.Context) {
	var mission models.Mission
	if err := c.ShouldBindJSON(&mission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if mission.ExternalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "external_id is required"})
		return
	}

	err := h.repo.Create(&mission)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create mission"})
		return
	}

	c.JSON(http.StatusCreated, mission)
}

func (h *MissionHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mission ID"})
		return
	}

	var mission models.Mission
	if err := c.ShouldBindJSON(&mission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mission.ID = uint(id)
	err = h.repo.Update(&mission)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update mission"})
		return
	}

	c.JSON(http.StatusOK, mission)
}

func (h *MissionHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mission ID"})
		return
	}

	err = h.repo.Delete(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete mission"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
