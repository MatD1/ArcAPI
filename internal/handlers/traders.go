package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/services"
)

type TradersHandler struct {
	tradersService *services.TradersService
}

func NewTradersHandler(tradersService *services.TradersService) *TradersHandler {
	return &TradersHandler{
		tradersService: tradersService,
	}
}

// GetTraders returns the traders data from cache or external API
func (h *TradersHandler) GetTraders(c *gin.Context) {
	data, err := h.tradersService.GetTraders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch traders data"})
		return
	}

	c.JSON(http.StatusOK, data)
}
