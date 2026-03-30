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

// List returns all alerts (paginated)
// @Summary List all alerts
// @Description Fetch all alerts with optional pagination
// @Tags alerts
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} PaginatedResponse{data=[]models.Alert} "Successfully fetched alerts"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /alerts [get]
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

// GetActive returns all active alerts
// @Summary List active alerts
// @Description Fetch all alerts that are currently marked as active
// @Tags alerts
// @Accept json
// @Produce json
// @Success 200 {object} PaginatedResponse{data=[]models.Alert} "Successfully fetched active alerts"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /alerts/active [get]
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

// Get returns a single alert by ID
// @Summary Get a single alert
// @Description Fetch an alert by its numeric ID
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path int true "Alert ID"
// @Success 200 {object} models.Alert "Successfully fetched the alert"
// @Failure 400 {object} ErrorResponse "Invalid alert ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 404 {object} ErrorResponse "Alert not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /alerts/{id} [get]
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

// Create adds a new alert
// @Summary Create an alert
// @Description Add a new alert to the database
// @Tags alerts
// @Accept json
// @Produce json
// @Param alert body models.Alert true "Alert object"
// @Success 201 {object} models.Alert "Successfully created the alert"
// @Failure 400 {object} ErrorResponse "Invalid input data"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /alerts [post]
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

// Update modifies an existing alert
// @Summary Update an alert
// @Description Update an existing alert by its ID
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path int true "Alert ID"
// @Param alert body models.Alert true "Updated alert object"
// @Success 200 {object} models.Alert "Successfully updated the alert"
// @Failure 400 {object} ErrorResponse "Invalid input or ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /alerts/{id} [put]
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

// Delete removes an alert
// @Summary Delete an alert
// @Description Delete an existing alert by its ID
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path int true "Alert ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse "Invalid alert ID"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /alerts/{id} [delete]
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
