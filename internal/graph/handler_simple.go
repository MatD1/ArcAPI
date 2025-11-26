package graph

import (
	"context"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/services"
	"github.com/vektah/gqlparser/v2/ast"
)

// GraphQLHandlerSimple is a simplified handler that works before code generation
// After running gqlgen generate, use NewGraphQLHandler instead
type GraphQLHandlerSimple struct {
	resolver    *Resolver
	authService *services.AuthService
}

// NewGraphQLHandlerSimple creates a simple GraphQL handler
// This is a placeholder until gqlgen code is generated
func NewGraphQLHandlerSimple(resolver *Resolver, authService *services.AuthService) *GraphQLHandlerSimple {
	return &GraphQLHandlerSimple{
		resolver:    resolver,
		authService: authService,
	}
}

// GraphQLHandler handles POST requests to /graphql
// This will be replaced after code generation
func (h *GraphQLHandlerSimple) GraphQLHandler(c *gin.Context) {
	// For now, return a message indicating code generation is needed
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "GraphQL code generation required. Run: go run github.com/99designs/gqlgen generate",
		"instructions": []string{
			"1. Install gqlgen: go install github.com/99designs/gqlgen@latest",
			"2. Generate code: cd internal/graph && go run github.com/99designs/gqlgen generate",
			"3. Restart the server",
		},
	})
}

// PlaygroundHandler serves GraphQL playground
func (h *GraphQLHandlerSimple) PlaygroundHandler(c *gin.Context) {
	playground.Handler("GraphQL Playground", "/api/v1/graphql").ServeHTTP(c.Writer, c.Request)
}

// createGraphQLServer creates a GraphQL server with security middleware
// This function will be used after code generation
func createGraphQLServer(resolver *Resolver, authService *services.AuthService) *handler.Server {
	// This will be implemented after code generation
	// For now, return nil
	return nil
}

// setupSecurityMiddleware configures security middleware for GraphQL
func setupSecurityMiddleware(srv *handler.Server, authService *services.AuthService) {
	// Add query complexity analysis
	srv.Use(extension.FixedComplexityLimit(MaxQueryComplexity))

	// Add query caching (LRU cache for parsed queries)
	// Cache stores *ast.QueryDocument objects
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	// Configure transports (only POST for security - no GET to prevent CSRF)
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10,
	})

	// Add request validation middleware
	srv.Use(extension.Introspection{})

	// Add custom middleware for authentication and validation
	srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		// Get user from request headers
		reqCtx := graphql.GetRequestContext(ctx)
		if reqCtx != nil {
			headers := reqCtx.Headers
			authHeader := headers.Get("Authorization")

			// Health check doesn't require auth
			opCtx := graphql.GetOperationContext(ctx)
			if opCtx != nil && opCtx.Operation != nil {
				// Check if this is a health query
				for _, sel := range opCtx.Operation.SelectionSet {
					if field, ok := sel.(*ast.Field); ok && field.Name == "health" {
						return next(ctx)
					}
				}
			}

			if authHeader != "" {
				// Extract and validate token
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					user, err := authService.ValidateJWT(parts[1])
					if err == nil {
						// Add user to context
						ctx = context.WithValue(ctx, UserContextKey, user)
					}
				}
			}
		}

		// Validate operation before execution
		if err := ValidateOperation(ctx); err != nil {
			return graphql.OneShot(graphql.ErrorResponse(ctx, "%s", err.Error()))
		}

		return next(ctx)
	})
}
