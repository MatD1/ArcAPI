package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/services"
)

type GitHubDataHandler struct {
	dataService *services.GitHubDataService
}

func NewGitHubDataHandler(dataService *services.GitHubDataService) *GitHubDataHandler {
	return &GitHubDataHandler{dataService: dataService}
}

func (h *GitHubDataHandler) GetBots(c *gin.Context) {
	data, err := h.dataService.GetBots(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bots data"})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *GitHubDataHandler) GetMaps(c *gin.Context) {
	data, err := h.dataService.GetMaps(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch maps data"})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *GitHubDataHandler) GetTraders(c *gin.Context) {
	data, err := h.dataService.GetTraders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch traders data"})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *GitHubDataHandler) GetProjects(c *gin.Context) {
	data, err := h.dataService.GetProjects(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects data"})
		return
	}
	c.JSON(http.StatusOK, data)
}
