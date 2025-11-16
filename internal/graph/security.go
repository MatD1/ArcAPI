package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/services"
	"github.com/vektah/gqlparser/v2/ast"
)

const (
	// MaxQueryDepth limits the depth of nested queries to prevent deep recursion attacks
	MaxQueryDepth = 10
	
	// MaxQueryComplexity limits the total complexity score of a query
	MaxQueryComplexity = 1000
	
	// MaxQueryCost limits the estimated cost of a query
	MaxQueryCost = 500
)

// UserContextKey is the key for storing user in context
const UserContextKey = "user"

// GetUserFromContext retrieves the authenticated user from context
func GetUserFromContext(ctx context.Context) (*models.User, error) {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	if !ok || user == nil {
		return nil, fmt.Errorf("user not authenticated")
	}
	return user, nil
}

// RequireAuth ensures a user is authenticated
func RequireAuth(ctx context.Context) (*models.User, error) {
	return GetUserFromContext(ctx)
}

// RequireAdmin ensures a user is authenticated and is an admin
func RequireAdmin(ctx context.Context) (*models.User, error) {
	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if user.Role != models.RoleAdmin {
		return nil, fmt.Errorf("admin access required")
	}
	return user, nil
}

// DepthLimitDirective validates query depth
func DepthLimitDirective(ctx context.Context, obj interface{}, next graphql.Resolver, maxDepth int) (interface{}, error) {
	depth := calculateDepth(ctx)
	if depth > maxDepth {
		return nil, fmt.Errorf("query depth %d exceeds maximum allowed depth of %d", depth, maxDepth)
	}
	return next(ctx)
}

// calculateDepth calculates the depth of the current query
func calculateDepth(ctx context.Context) int {
	opCtx := graphql.GetOperationContext(ctx)
	if opCtx == nil || opCtx.Operation == nil {
		return 0
	}
	return calculateSelectionSetDepth(opCtx.Operation.SelectionSet, 0)
}

// calculateSelectionSetDepth recursively calculates depth
func calculateSelectionSetDepth(selectionSet ast.SelectionSet, currentDepth int) int {
	maxDepth := currentDepth
	for _, selection := range selectionSet {
		switch sel := selection.(type) {
		case *ast.Field:
			if sel.SelectionSet != nil {
				depth := calculateSelectionSetDepth(sel.SelectionSet, currentDepth+1)
				if depth > maxDepth {
					maxDepth = depth
				}
			}
		case *ast.FragmentSpread:
			// Fragment depth is handled by the fragment definition
			if currentDepth+1 > maxDepth {
				maxDepth = currentDepth + 1
			}
		case *ast.InlineFragment:
			if sel.SelectionSet != nil {
				depth := calculateSelectionSetDepth(sel.SelectionSet, currentDepth+1)
				if depth > maxDepth {
					maxDepth = depth
				}
			}
		}
	}
	return maxDepth
}

// ComplexityLimitDirective validates query complexity
func ComplexityLimitDirective(ctx context.Context, obj interface{}, next graphql.Resolver, maxComplexity int) (interface{}, error) {
	complexity := calculateComplexity(ctx)
	if complexity > maxComplexity {
		return nil, fmt.Errorf("query complexity %d exceeds maximum allowed complexity of %d", complexity, maxComplexity)
	}
	return next(ctx)
}

// calculateComplexity calculates the complexity score of a query
func calculateComplexity(ctx context.Context) int {
	opCtx := graphql.GetOperationContext(ctx)
	if opCtx == nil || opCtx.Operation == nil {
		return 0
	}
	return calculateSelectionSetComplexity(opCtx.Operation.SelectionSet, 1)
}

// calculateSelectionSetComplexity recursively calculates complexity
func calculateSelectionSetComplexity(selectionSet ast.SelectionSet, multiplier int) int {
	complexity := 0
	for _, selection := range selectionSet {
		switch sel := selection.(type) {
		case *ast.Field:
			// Base complexity for each field
			fieldComplexity := 1
			
			// Increase complexity for list fields (they can be expensive)
			if sel.Definition != nil {
				if strings.HasSuffix(sel.Definition.Type.String(), "!") || strings.Contains(sel.Definition.Type.String(), "[") {
					fieldComplexity = 10 // Lists are more expensive
				}
			}
			
			// Check for pagination arguments (limit)
			if sel.Arguments != nil {
				for _, arg := range sel.Arguments {
					if arg.Name == "limit" || arg.Name == "pagination" {
						// Check if value is an integer (pagination increases complexity)
						if arg.Value != nil {
							fieldComplexity *= 2 // Pagination increases cost
						}
					}
				}
			}
			
			complexity += fieldComplexity * multiplier
			
			// Recursively calculate nested fields
			if sel.SelectionSet != nil {
				complexity += calculateSelectionSetComplexity(sel.SelectionSet, multiplier*2)
			}
		case *ast.FragmentSpread:
			complexity += 1 * multiplier
		case *ast.InlineFragment:
			if sel.SelectionSet != nil {
				complexity += calculateSelectionSetComplexity(sel.SelectionSet, multiplier)
			}
		}
	}
	return complexity
}

// ValidateQueryDepth validates query depth before execution
func ValidateQueryDepth(ctx context.Context, maxDepth int) error {
	depth := calculateDepth(ctx)
	if depth > maxDepth {
		return fmt.Errorf("query depth %d exceeds maximum allowed depth of %d", depth, maxDepth)
	}
	return nil
}

// ValidateQueryComplexity validates query complexity before execution
func ValidateQueryComplexity(ctx context.Context, maxComplexity int) error {
	complexity := calculateComplexity(ctx)
	if complexity > maxComplexity {
		return fmt.Errorf("query complexity %d exceeds maximum allowed complexity of %d", complexity, maxComplexity)
	}
	return nil
}

// GraphQLResolverAuthMiddleware validates JWT token and adds user to context for GraphQL resolvers
func GraphQLResolverAuthMiddleware(authService *services.AuthService) func(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	return func(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
		// Get authorization header from request context
		reqCtx := graphql.GetRequestContext(ctx)
		if reqCtx == nil {
			return nil, fmt.Errorf("request context not found")
		}
		
		// Get headers from request
		headers := reqCtx.Headers
		authHeader := headers.Get("Authorization")
		if authHeader == "" {
			return nil, fmt.Errorf("authorization header required")
		}
		
		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return nil, fmt.Errorf("invalid authorization header format")
		}
		
		tokenString := parts[1]
		
		// Validate JWT
		user, err := authService.ValidateJWT(tokenString)
		if err != nil {
			return nil, fmt.Errorf("invalid or expired JWT token: %w", err)
		}
		
		// Add user to context
		ctx = context.WithValue(ctx, UserContextKey, user)
		
		return next(ctx)
	}
}

// GraphQLAdminMiddleware ensures user is admin
func GraphQLAdminMiddleware(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	user, err := RequireAdmin(ctx)
	if err != nil {
		return nil, err
	}
	_ = user // Use user to avoid unused variable
	return next(ctx)
}

// ValidateOperation validates GraphQL operation before execution
func ValidateOperation(ctx context.Context) error {
	opCtx := graphql.GetOperationContext(ctx)
	if opCtx == nil {
		return fmt.Errorf("operation context not found")
	}
	
	// Validate query depth
	if err := ValidateQueryDepth(ctx, MaxQueryDepth); err != nil {
		return err
	}
	
	// Validate query complexity
	if err := ValidateQueryComplexity(ctx, MaxQueryComplexity); err != nil {
		return err
	}
	
	return nil
}

