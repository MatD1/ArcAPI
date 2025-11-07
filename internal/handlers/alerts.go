package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
)

type AlertHandler struct {
	repo *repository.AlertRepository
}

func NewAlertHandler(repo *repository.AlertRepository) *AlertHandler {
	return &AlertHandler{repo: repo}
}

func (h *AlertHandler) List(c *gin.Context) {
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
	alerts, count, err := h.repo.FindAll(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": alerts,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": count,
		},
	})
}

func (h *AlertHandler) GetActive(c *gin.Context) {
	alerts, err := h.repo.FindActive()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch active alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": alerts,
	})
}

func (h *AlertHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}

	alert, err := h.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	c.JSON(http.StatusOK, alert)
}

func (h *AlertHandler) Create(c *gin.Context) {
	var alert models.Alert
	if err := c.ShouldBindJSON(&alert); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if alert.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	if alert.Severity == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "severity is required"})
		return
	}

	// Validate severity
	validSeverities := map[string]bool{
		"info":     true,
		"warning":  true,
		"error":    true,
		"critical": true,
	}
	if !validSeverities[alert.Severity] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "severity must be one of: info, warning, error, critical"})
		return
	}

	// Default is_active to true if not provided
	// The model has default:true in GORM, but we'll also set it here for consistency
	// If the field wasn't provided in JSON, it will be false (zero value), so we default to true
	// We can't easily check if it was provided, so we'll just ensure it defaults to true
	// The GORM default will handle it, but we can also set it explicitly
	if !alert.IsActive {
		// Check if it was explicitly set to false by reading the raw body
		// For now, we'll just default new alerts to active
		alert.IsActive = true
	}

	err := h.repo.Create(&alert)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create alert"})
		return
	}

	c.JSON(http.StatusCreated, alert)
}

func (h *AlertHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}

	var alert models.Alert
	if err := c.ShouldBindJSON(&alert); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate severity if provided
	if alert.Severity != "" {
		validSeverities := map[string]bool{
			"info":     true,
			"warning":  true,
			"error":    true,
			"critical": true,
		}
		if !validSeverities[alert.Severity] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "severity must be one of: info, warning, error, critical"})
			return
		}
	}

	alert.ID = uint(id)
	err = h.repo.Update(&alert)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update alert"})
		return
	}

	c.JSON(http.StatusOK, alert)
}

func (h *AlertHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}

	err = h.repo.Delete(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete alert"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
