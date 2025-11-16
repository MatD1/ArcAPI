package graph

import (
	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/repository"
	"github.com/mat/arcapi/internal/services"
)

// SetupGraphQLRoutes sets up GraphQL routes in the Gin router
// This function should be called from main.go after initializing all dependencies
func SetupGraphQLRoutes(
	r *gin.RouterGroup,
	userRepo *repository.UserRepository,
	questRepo *repository.QuestRepository,
	itemRepo *repository.ItemRepository,
	skillNodeRepo *repository.SkillNodeRepository,
	hideoutModuleRepo *repository.HideoutModuleRepository,
	enemyTypeRepo *repository.EnemyTypeRepository,
	alertRepo *repository.AlertRepository,
	questProgressRepo *repository.UserQuestProgressRepository,
	hideoutModuleProgressRepo *repository.UserHideoutModuleProgressRepository,
	skillNodeProgressRepo *repository.UserSkillNodeProgressRepository,
	blueprintProgressRepo *repository.UserBlueprintProgressRepository,
	authService *services.AuthService,
	dataCacheService *services.DataCacheService,
) {
	// Create resolver
	resolver := NewResolver(
		userRepo,
		questRepo,
		itemRepo,
		skillNodeRepo,
		hideoutModuleRepo,
		enemyTypeRepo,
		alertRepo,
		questProgressRepo,
		hideoutModuleProgressRepo,
		skillNodeProgressRepo,
		blueprintProgressRepo,
		authService,
		dataCacheService,
	)
	
	// Try to create GraphQL handler (will fail if code not generated)
	graphqlHandler := NewGraphQLHandler(resolver, authService)
	
	// If handler creation failed, use simple handler
	if graphqlHandler == nil {
		simpleHandler := NewGraphQLHandlerSimple(resolver, authService)
		
		// GraphQL endpoint (POST only for security)
		r.POST("/graphql", GraphQLAuthMiddleware(authService), simpleHandler.GraphQLHandler)
		
		// GraphQL Playground (development only - consider protecting with admin auth)
		r.GET("/graphql/playground", simpleHandler.PlaygroundHandler)
		
		return
	}
	
	// GraphQL endpoint (POST only for security)
	r.POST("/graphql", GraphQLAuthMiddleware(authService), graphqlHandler.GraphQLHandler)
	
	// GraphQL Playground (development only - consider protecting with admin auth)
	r.GET("/graphql/playground", graphqlHandler.PlaygroundHandler)
}

