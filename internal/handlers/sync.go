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
// ForceSync triggers an immediate sync from GitHub
// @Summary Force data sync
// @Description Trigger an immediate synchronization of Game Data from GitHub. Only admins can force a sync.
// @Tags sync
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "Sync triggered successfully"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Failure 409 {object} ErrorResponse "Sync already running"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /admin/sync/force [post]
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
// SyncStatus returns the current sync status
// @Summary Get sync status
// @Description Check if a synchronization process is currently running. Only admins can check status.
// @Tags sync
// @Accept json
// @Produce json
// @Success 200 {object} map[string]bool "Sync status"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not an administrator"
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /admin/sync/status [get]
func (h *SyncHandler) SyncStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"is_running": h.syncService.IsRunning(),
	})
}

// GetSnapshot returns a full data snapshot for client hydration
// GetSnapshot returns a full data snapshot for client hydration
// @Summary Get data snapshot
// @Description Fetch a complete snapshot of all game data (quests, items, etc.) for client hydration.
// @Tags sync
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Full data snapshot"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /sync/snapshot [get]
func (h *SyncHandler) GetSnapshot(c *gin.Context) {
	snapshot, err := h.syncService.GetSnapshot()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate snapshot",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}
