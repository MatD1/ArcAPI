package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type ConfigHandler struct{}

func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{}
}

// GetFrontendConfig returns frontend configuration (public config only)
func (h *ConfigHandler) GetFrontendConfig(c *gin.Context) {
	// Only return public configuration that's safe to expose to the frontend
	config := gin.H{
		"supabase": gin.H{
			"enabled": os.Getenv("NEXT_PUBLIC_SUPABASE_ENABLED") == "true",
			"url":     os.Getenv("NEXT_PUBLIC_SUPABASE_URL"),
			"anonKey": os.Getenv("NEXT_PUBLIC_SUPABASE_ANON_KEY"),
		},
	}

	c.JSON(http.StatusOK, config)
}

