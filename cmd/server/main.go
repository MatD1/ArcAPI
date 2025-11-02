package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/handlers"
	"github.com/mat/arcapi/internal/middleware"
	"github.com/mat/arcapi/internal/repository"
	"github.com/mat/arcapi/internal/services"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := repository.NewDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize Redis cache (optional, continue if it fails)
	cacheService, err := services.NewCacheService(cfg)
	if err != nil {
		log.Printf("Warning: Redis not available, continuing without cache: %v", err)
		cacheService = nil
	} else {
		defer cacheService.Close()
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	jwtTokenRepo := repository.NewJWTTokenRepository(db)
	missionRepo := repository.NewMissionRepository(db)
	itemRepo := repository.NewItemRepository(db)
	skillNodeRepo := repository.NewSkillNodeRepository(db)
	hideoutModuleRepo := repository.NewHideoutModuleRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)

	// Initialize services
	authService := services.NewAuthService(userRepo, apiKeyRepo, jwtTokenRepo, cacheService, cfg)
	userService := services.NewUserService(userRepo)

	// Initialize sync service
	syncService := services.NewSyncService(
		missionRepo,
		itemRepo,
		skillNodeRepo,
		hideoutModuleRepo,
		cfg,
	)

	// Start sync service
	if err := syncService.Start(); err != nil {
		log.Fatalf("Failed to start sync service: %v", err)
	}
	defer syncService.Stop()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, userService, cfg)
	missionHandler := handlers.NewMissionHandler(missionRepo)
	itemHandler := handlers.NewItemHandler(itemRepo)
	skillNodeHandler := handlers.NewSkillNodeHandler(skillNodeRepo)
	hideoutModuleHandler := handlers.NewHideoutModuleHandler(hideoutModuleRepo)
	managementHandler := handlers.NewManagementHandler(
		authService,
		apiKeyRepo,
		jwtTokenRepo,
		auditLogRepo,
	)

	// Setup router
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// Logger middleware (must be before auth middleware to log all requests)
	r.Use(middleware.LoggerMiddleware(auditLogRepo))

	// Public routes
	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.GET("/github/login", authHandler.GitHubLogin)
			auth.GET("/github/callback", authHandler.GitHubCallback)
			auth.POST("/login", authHandler.LoginWithAPIKey)
		}

		// Protected routes (require API key + JWT)
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(authService))
		{
			// Missions
			protected.GET("/missions", missionHandler.List)
			protected.GET("/missions/:id", missionHandler.Get)
			protected.POST("/missions", missionHandler.Create)
			protected.PUT("/missions/:id", missionHandler.Update)
			protected.DELETE("/missions/:id", missionHandler.Delete)

			// Items
			protected.GET("/items", itemHandler.List)
			protected.GET("/items/:id", itemHandler.Get)
			protected.POST("/items", itemHandler.Create)
			protected.PUT("/items/:id", itemHandler.Update)
			protected.DELETE("/items/:id", itemHandler.Delete)

			// Skill Nodes
			protected.GET("/skill-nodes", skillNodeHandler.List)
			protected.GET("/skill-nodes/:id", skillNodeHandler.Get)
			protected.POST("/skill-nodes", skillNodeHandler.Create)
			protected.PUT("/skill-nodes/:id", skillNodeHandler.Update)
			protected.DELETE("/skill-nodes/:id", skillNodeHandler.Delete)

			// Hideout Modules
			protected.GET("/hideout-modules", hideoutModuleHandler.List)
			protected.GET("/hideout-modules/:id", hideoutModuleHandler.Get)
			protected.POST("/hideout-modules", hideoutModuleHandler.Create)
			protected.PUT("/hideout-modules/:id", hideoutModuleHandler.Update)
			protected.DELETE("/hideout-modules/:id", hideoutModuleHandler.Delete)

			// Management API (admin only)
			admin := protected.Group("/admin")
			admin.Use(middleware.AdminMiddleware())
			{
				admin.POST("/api-keys", managementHandler.CreateAPIKey)
				admin.GET("/api-keys", managementHandler.ListAPIKeys)
				admin.DELETE("/api-keys/:id", managementHandler.RevokeAPIKey)
				admin.POST("/jwts/revoke", managementHandler.RevokeJWT)
				admin.GET("/jwts", managementHandler.ListJWTs)
				admin.GET("/logs", managementHandler.QueryLogs)
			}
		}
	}

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Start server
	go func() {
		if err := r.Run(":" + cfg.APIPort); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Server starting on port %s", cfg.APIPort)

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
}
