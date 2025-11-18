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
	// Read from APPWRITE_* env vars (Railway) or NEXT_PUBLIC_APPWRITE_* (build-time)
	appwriteEnabled := os.Getenv("APPWRITE_ENABLED") == "true" || os.Getenv("NEXT_PUBLIC_APPWRITE_ENABLED") == "true"
	appwriteEndpoint := os.Getenv("APPWRITE_ENDPOINT")
	if appwriteEndpoint == "" {
		appwriteEndpoint = os.Getenv("NEXT_PUBLIC_APPWRITE_ENDPOINT")
	}
	appwriteProjectID := os.Getenv("APPWRITE_PROJECT_ID")
	if appwriteProjectID == "" {
		appwriteProjectID = os.Getenv("NEXT_PUBLIC_APPWRITE_PROJECT_ID")
	}
	appwriteDatabaseID := os.Getenv("APPWRITE_DATABASE_ID")
	if appwriteDatabaseID == "" {
		appwriteDatabaseID = os.Getenv("NEXT_PUBLIC_APPWRITE_DATABASE_ID")
	}
	// Note: databaseId should be the actual Appwrite database ID (not the name)
	// The database ID is a unique identifier found in the Appwrite console
	// No default value - must be explicitly configured

	// GraphQL is enabled by default in Appwrite, but can be disabled via env var
	appwriteGraphQLEnabled := os.Getenv("APPWRITE_GRAPHQL_ENABLED")
	if appwriteGraphQLEnabled == "" {
		appwriteGraphQLEnabled = os.Getenv("NEXT_PUBLIC_APPWRITE_GRAPHQL_ENABLED")
	}
	// Default to true if not specified (GraphQL is enabled by default in Appwrite)
	graphqlEnabled := appwriteGraphQLEnabled == "" || appwriteGraphQLEnabled == "true"

	config := gin.H{
		"appwrite": gin.H{
			"enabled":        appwriteEnabled && appwriteEndpoint != "" && appwriteProjectID != "",
			"endpoint":       appwriteEndpoint,
			"projectId":      appwriteProjectID,
			"databaseId":     appwriteDatabaseID,
			"graphqlEnabled": graphqlEnabled,
		},
	}

	c.JSON(http.StatusOK, config)
}
