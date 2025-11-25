package graph

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/middleware"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/services"
)

// GraphQLHandler handles GraphQL requests
// Note: srv will be set after code generation
type GraphQLHandler struct {
	srv         interface{} // Will be *handler.Server after code generation
	authService *services.AuthService
}

// NewGraphQLHandler creates a new GraphQL handler with security middleware
// NOTE: This function requires gqlgen code generation to work.
// After running `go run github.com/99designs/gqlgen generate`, update this function
// to use the generated NewExecutableSchema function.
//
// Example after generation:
//
//	cfg := Config{Resolvers: resolver}
//	srv := handler.NewDefaultServer(NewExecutableSchema(cfg))
//	setupSecurityMiddleware(srv, authService)
//	return &GraphQLHandler{srv: srv, authService: authService}
func NewGraphQLHandler(resolver *Resolver, authService *services.AuthService) *GraphQLHandler {
	// TODO: After code generation, uncomment and update:
	// cfg := Config{Resolvers: resolver}
	// srv := handler.NewDefaultServer(NewExecutableSchema(cfg))
	// setupSecurityMiddleware(srv, authService)
	// return &GraphQLHandler{srv: srv, authService: authService}

	// Temporary: return nil until code is generated
	// Use NewGraphQLHandlerSimple for a placeholder implementation
	return nil
}

// GraphQLHandler handles POST requests to /graphql
func (h *GraphQLHandler) GraphQLHandler(c *gin.Context) {
	// This will be implemented after code generation
	// For now, return error
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "GraphQL handler not initialized. Run: make generate-graphql",
	})
}

// PlaygroundHandler serves GraphQL playground (only in development)
func (h *GraphQLHandler) PlaygroundHandler(c *gin.Context) {
	// Only allow playground in development mode
	// In production, you might want to disable this or protect it with admin auth
	playground.Handler("GraphQL Playground", "/api/v1/graphql").ServeHTTP(c.Writer, c.Request)
}

// GraphQLAuthMiddleware validates API requests before hitting the GraphQL handler
func GraphQLAuthMiddleware(authService *services.AuthService, cfg *config.Config, oidcService *services.OIDCService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, token, err := middleware.AuthenticateRequest(c, authService, oidcService, cfg)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// Add user to Gin context (will be accessible in GraphQL resolvers via request context)
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("token", token)

		c.Next()
	}
}

// RequireAuthFromContext extracts user from GraphQL context
func RequireAuthFromContext(ctx context.Context) (*models.User, error) {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	if !ok || user == nil {
		return nil, fmt.Errorf("user not authenticated")
	}
	return user, nil
}
