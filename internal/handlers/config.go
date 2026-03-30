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
// GetFrontendConfig returns frontend configuration (public config only)
// @Summary Get frontend configuration
// @Description Returns public configuration settings for the frontend (e.g. Supabase details)
// @Tags config
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Successfully fetched frontend configuration"
// @Router /config [get]
func (h *ConfigHandler) GetFrontendConfig(c *gin.Context) {
	// Supabase Configuration
	supabaseEnabled := os.Getenv("SUPABASE_ENABLED") == "true" || os.Getenv("NEXT_PUBLIC_SUPABASE_ENABLED") == "true"
	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseURL == "" {
		supabaseURL = os.Getenv("NEXT_PUBLIC_SUPABASE_URL")
	}
	supabaseAnonKey := os.Getenv("SUPABASE_PUBLISHABLE_KEY")
	if supabaseAnonKey == "" {
		supabaseAnonKey = os.Getenv("NEXT_PUBLIC_SUPABASE_PUBLISHABLE_DEFAULT_KEY")
	}

	config := gin.H{
		"supabase": gin.H{
			"enabled":   supabaseEnabled && supabaseURL != "" && supabaseAnonKey != "",
			"url":       supabaseURL,
			"anonKey":   supabaseAnonKey,
		},
	}

	c.JSON(http.StatusOK, config)
}
