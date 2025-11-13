package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/repository"
	"github.com/mat/arcapi/internal/services"
)

type HealthHandler struct {
	db           *repository.DB
	cacheService *services.CacheService
}

func NewHealthHandler(db *repository.DB, cacheService *services.CacheService) *HealthHandler {
	return &HealthHandler{
		db:           db,
		cacheService: cacheService,
	}
}

// HealthCheck performs a comprehensive health check
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	status := gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	checks := gin.H{}
	allHealthy := true

	// Check database
	sqlDB, err := h.db.DB.DB()
	if err != nil {
		checks["database"] = gin.H{"status": "error", "error": err.Error()}
		allHealthy = false
	} else {
		if err := sqlDB.Ping(); err != nil {
			checks["database"] = gin.H{"status": "error", "error": err.Error()}
			allHealthy = false
		} else {
			stats := sqlDB.Stats()
			checks["database"] = gin.H{
				"status":          "healthy",
				"open_connections": stats.OpenConnections,
				"max_open":        stats.MaxOpenConnections,
			}
		}
	}

	// Check Redis cache (if available)
	if h.cacheService != nil {
		ctx := h.cacheService.Context()
		client := h.cacheService.Client()
		if err := client.Ping(ctx).Err(); err != nil {
			checks["cache"] = gin.H{"status": "error", "error": err.Error()}
			// Cache is optional, so don't mark as unhealthy
		} else {
			checks["cache"] = gin.H{"status": "healthy"}
		}
	} else {
		checks["cache"] = gin.H{"status": "disabled"}
	}

	status["checks"] = checks

	if !allHealthy {
		status["status"] = "degraded"
		c.JSON(http.StatusServiceUnavailable, status)
		return
	}

	c.JSON(http.StatusOK, status)
}

// ReadinessCheck performs a lightweight readiness check
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	// Quick database check
	sqlDB, err := h.db.DB.DB()
	if err != nil || sqlDB.Ping() != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"error":  "database unavailable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// LivenessCheck performs a minimal liveness check
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}
