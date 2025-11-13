package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/services"
)

type SyncHandler struct {
	syncService *services.SyncService
}

func NewSyncHandler(syncService *services.SyncService) *SyncHandler {
	return &SyncHandler{
		syncService: syncService,
	}
}

// ForceSync triggers an immediate sync from GitHub
func (h *SyncHandler) ForceSync(c *gin.Context) {
	if h.syncService.IsRunning() {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Sync is already running. Please wait for it to complete.",
		})
		return
	}

	err := h.syncService.ForceSync()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to trigger sync",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Sync triggered successfully",
		"status":  "running",
	})
}

// SyncStatus returns the current sync status
func (h *SyncHandler) SyncStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"is_running": h.syncService.IsRunning(),
	})
}
