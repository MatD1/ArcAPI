package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/graph"
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

	// Initialize database with retry logic (handles cold starts)
	log.Println("Connecting to database...")
	db, err := repository.NewDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connection established successfully")
	defer func() {
		sqlDB, err := db.DB.DB()
		if err == nil {
			sqlDB.Close()
		}
	}()

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
	questRepo := repository.NewQuestRepository(db)
	itemRepo := repository.NewItemRepository(db)
	skillNodeRepo := repository.NewSkillNodeRepository(db)
	hideoutModuleRepo := repository.NewHideoutModuleRepository(db)
	enemyTypeRepo := repository.NewEnemyTypeRepository(db)
	alertRepo := repository.NewAlertRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)
	questProgressRepo := repository.NewUserQuestProgressRepository(db)
	hideoutModuleProgressRepo := repository.NewUserHideoutModuleProgressRepository(db)
	skillNodeProgressRepo := repository.NewUserSkillNodeProgressRepository(db)
	blueprintProgressRepo := repository.NewUserBlueprintProgressRepository(db)
	botRepo := repository.NewBotRepository(db)
	mapRepo := repository.NewMapRepository(db)
	traderRepo := repository.NewTraderRepository(db)
	projectRepo := repository.NewProjectRepository(db)

	// Initialize services
	authCodeRepo := repository.NewAuthorizationCodeRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	authService := services.NewAuthService(userRepo, apiKeyRepo, jwtTokenRepo, authCodeRepo, refreshTokenRepo, cacheService, cfg)
	var oidcService *services.OIDCService
	if cfg.AuthentikEnabled {
		oidcService, err = services.NewOIDCService(cfg)
		if err != nil {
			log.Fatalf("failed to initialize Authentik OIDC service: %v", err)
		}
	}
	userService := services.NewUserService(userRepo)

	// Initialize data cache service (only if cache is available)
	var dataCacheService *services.DataCacheService
	if cacheService != nil {
		dataCacheService = services.NewDataCacheService(cacheService, itemRepo, questRepo)
		dataCacheService.Start()
		log.Println("Data cache service started - will refresh items and quests every 15 minutes")
	}

	// Initialize sync service (with cache service if available)
	var syncService *services.SyncService
	if dataCacheService != nil {
		syncService = services.NewSyncServiceWithCache(
			questRepo,
			itemRepo,
			skillNodeRepo,
			hideoutModuleRepo,
			botRepo,
			mapRepo,
			traderRepo,
			projectRepo,
			dataCacheService,
			cfg,
		)
	} else {
		syncService = services.NewSyncService(
			questRepo,
			itemRepo,
			skillNodeRepo,
			hideoutModuleRepo,
			botRepo,
			mapRepo,
			traderRepo,
			projectRepo,
			cfg,
		)
	}

	// Start sync service
	if err := syncService.Start(); err != nil {
		log.Fatalf("Failed to start sync service: %v", err)
	}
	defer syncService.Stop()

	// Initialize traders service (only if cache is available)
	var tradersService *services.TradersService
	if cacheService != nil {
		tradersService = services.NewTradersService(cacheService)
		tradersService.Start()
		log.Println("Traders service started - will refresh every 15 minutes")
	}

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, userService, cfg, apiKeyRepo, oidcService)

	// Use cache-enabled handlers if cache is available
	var questHandler *handlers.QuestHandler
	if dataCacheService != nil {
		questHandler = handlers.NewQuestHandlerWithCache(questRepo, dataCacheService)
	} else {
		questHandler = handlers.NewQuestHandler(questRepo)
	}
	missionHandler := questHandler // Backward compatibility - uses questHandler internally

	var itemHandler *handlers.ItemHandler
	if dataCacheService != nil {
		itemHandler = handlers.NewItemHandlerWithCache(itemRepo, questRepo, hideoutModuleRepo, dataCacheService)
	} else {
		itemHandler = handlers.NewItemHandlerWithRepos(itemRepo, questRepo, hideoutModuleRepo)
	}
	skillNodeHandler := handlers.NewSkillNodeHandler(skillNodeRepo)
	hideoutModuleHandler := handlers.NewHideoutModuleHandler(hideoutModuleRepo)
	enemyTypeHandler := handlers.NewEnemyTypeHandler(enemyTypeRepo)
	alertHandler := handlers.NewAlertHandler(alertRepo)
	botHandler := handlers.NewBotHandler(botRepo)
	mapHandler := handlers.NewMapHandler(mapRepo)
	traderHandler := handlers.NewTraderHandler(traderRepo)
	projectHandler := handlers.NewProjectHandler(projectRepo)
	var tradersHandler *handlers.TradersHandler
	if tradersService != nil {
		tradersHandler = handlers.NewTradersHandler(tradersService)
	}
	managementHandler := handlers.NewManagementHandler(
		authService,
		apiKeyRepo,
		jwtTokenRepo,
		auditLogRepo,
		userRepo,
		hideoutModuleRepo,
	)
	syncHandler := handlers.NewSyncHandler(syncService)
	progressHandler := handlers.NewProgressHandler(
		questProgressRepo,
		hideoutModuleProgressRepo,
		skillNodeProgressRepo,
		blueprintProgressRepo,
		questRepo,
		hideoutModuleRepo,
		skillNodeRepo,
		itemRepo,
		userRepo,
	)
	exportHandler := handlers.NewExportHandler(
		questRepo,
		itemRepo,
		skillNodeRepo,
		hideoutModuleRepo,
		enemyTypeRepo,
		alertRepo,
		botRepo,
		mapRepo,
		traderRepo,
		projectRepo,
	)

	// Setup router
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// Request size limit (10MB max)
	r.Use(middleware.RequestSizeLimitMiddleware(10 * 1024 * 1024))

	// Security middleware (CORS, security headers)
	r.Use(middleware.SecurityMiddleware(cfg.GetAllowedOrigins()))

	// Logger middleware (must be before auth middleware to log all requests)
	r.Use(middleware.LoggerMiddleware(auditLogRepo))

	// Public routes
	api := r.Group("/api/v1")
	// Rate limiting middleware (applied to all API routes)
	api.Use(middleware.RateLimitMiddleware(cacheService, cfg.RateLimitRequests, cfg.RateLimitWindowSeconds))
	{
		auth := api.Group("/auth")
		{
			auth.GET("/github/login", authHandler.GitHubLogin)
			auth.GET("/github/callback", authHandler.GitHubCallback)
			auth.GET("/discord/login", authHandler.DiscordLogin)
			auth.GET("/discord/callback", authHandler.DiscordCallback)
			auth.GET("/exchange-token", authHandler.ExchangeTempToken) // Public endpoint to exchange temp token
			auth.POST("/login", authHandler.LoginWithAPIKey)
			auth.POST("/token", authHandler.TokenExchange)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/authentik/token", authHandler.AuthentikTokenExchange)
		}

		// Read-only routes (require JWT only)
		readOnly := api.Group("")
		readOnly.Use(middleware.JWTAuthMiddleware(authService, cfg, oidcService))
		{
			readOnly.GET("/me", authHandler.GetCurrentUser)
			readOnly.POST("/me/refresh-role", authHandler.RefreshUserRole)
			// Quests - Read
			readOnly.GET("/quests", questHandler.List)
			readOnly.GET("/quests/:id", questHandler.Get)
			// Backward compatibility
			readOnly.GET("/missions", missionHandler.List)
			readOnly.GET("/missions/:id", missionHandler.Get)

			// Items - Read
			readOnly.GET("/items", itemHandler.List)
			readOnly.GET("/items/:id", itemHandler.Get)
			readOnly.GET("/items/required", itemHandler.RequiredItems)   // Get all required items for quests and hideout modules
			readOnly.GET("/items/blueprints", itemHandler.GetBlueprints) // Get all blueprint items

			// Skill Nodes - Read
			readOnly.GET("/skill-nodes", skillNodeHandler.List)
			readOnly.GET("/skill-nodes/:id", skillNodeHandler.Get)

			// Hideout Modules - Read
			readOnly.GET("/hideout-modules", hideoutModuleHandler.List)
			readOnly.GET("/hideout-modules/:id", hideoutModuleHandler.Get)

			// Enemy Types - Read
			readOnly.GET("/enemy-types", enemyTypeHandler.List)
			readOnly.GET("/enemy-types/:id", enemyTypeHandler.Get)

			// Alerts - Read
			readOnly.GET("/alerts", alertHandler.List)
			readOnly.GET("/alerts/active", alertHandler.GetActive) // For mobile apps to fetch active alerts
			readOnly.GET("/alerts/:id", alertHandler.Get)

			// Traders - Read (cached from external API)
			if tradersHandler != nil {
				readOnly.GET("/traders", tradersHandler.GetTraders)
			}
			// Bots, Maps, Traders (repo), Projects - Read from database
			readOnly.GET("/bots", botHandler.List)
			readOnly.GET("/bots/:id", botHandler.Get)
			readOnly.GET("/maps", mapHandler.List)
			readOnly.GET("/maps/:id", mapHandler.Get)
			readOnly.GET("/repo-traders", traderHandler.List)
			readOnly.GET("/repo-traders/:id", traderHandler.Get)
			readOnly.GET("/projects", projectHandler.List)
			readOnly.GET("/projects/:id", projectHandler.Get)
		}

		// Progress routes (basic users can read and update their own progress)
		progress := api.Group("/progress")
		progress.Use(middleware.ProgressAuthMiddleware(authService, cfg, oidcService))
		{
			// Quest Progress
			progress.GET("/quests", progressHandler.GetMyQuestProgress)
			progress.PUT("/quests/:quest_id", progressHandler.UpdateQuestProgress)

			// Hideout Module Progress
			progress.GET("/hideout-modules", progressHandler.GetMyHideoutModuleProgress)
			progress.PUT("/hideout-modules/:module_id", progressHandler.UpdateHideoutModuleProgress)

			// Skill Node Progress
			progress.GET("/skill-nodes", progressHandler.GetMySkillNodeProgress)
			progress.PUT("/skill-nodes/:skill_node_id", progressHandler.UpdateSkillNodeProgress)

			// Blueprint Progress
			progress.GET("/blueprints", progressHandler.GetMyBlueprintProgress)
			progress.PUT("/blueprints/:item_id", progressHandler.UpdateBlueprintProgress)
		}

		// Write routes (require API key + JWT for regular users, or JWT only for admins)
		writeProtected := api.Group("")
		writeProtected.Use(middleware.WriteAuthMiddleware(authService, cfg, oidcService))
		{
			// Quests - Write
			writeProtected.POST("/quests", questHandler.Create)
			writeProtected.PUT("/quests/:id", questHandler.Update)
			writeProtected.DELETE("/quests/:id", questHandler.Delete)
			// Backward compatibility
			writeProtected.POST("/missions", missionHandler.Create)
			writeProtected.PUT("/missions/:id", missionHandler.Update)
			writeProtected.DELETE("/missions/:id", missionHandler.Delete)

			// Items - Write
			writeProtected.POST("/items", itemHandler.Create)
			writeProtected.PUT("/items/:id", itemHandler.Update)
			writeProtected.DELETE("/items/:id", itemHandler.Delete)

			// Skill Nodes - Write
			writeProtected.POST("/skill-nodes", skillNodeHandler.Create)
			writeProtected.PUT("/skill-nodes/:id", skillNodeHandler.Update)
			writeProtected.DELETE("/skill-nodes/:id", skillNodeHandler.Delete)

			// Hideout Modules - Write
			writeProtected.POST("/hideout-modules", hideoutModuleHandler.Create)
			writeProtected.PUT("/hideout-modules/:id", hideoutModuleHandler.Update)
			writeProtected.DELETE("/hideout-modules/:id", hideoutModuleHandler.Delete)

			// Enemy Types - Write
			writeProtected.POST("/enemy-types", enemyTypeHandler.Create)
			writeProtected.PUT("/enemy-types/:id", enemyTypeHandler.Update)
			writeProtected.DELETE("/enemy-types/:id", enemyTypeHandler.Delete)

			// Alerts - Write (admin only)
			writeProtected.POST("/alerts", alertHandler.Create)
			writeProtected.PUT("/alerts/:id", alertHandler.Update)
			writeProtected.DELETE("/alerts/:id", alertHandler.Delete)

			// Management API (admin only)
			admin := writeProtected.Group("/admin")
			admin.Use(middleware.AdminMiddleware())
			{
				admin.POST("/api-keys", managementHandler.CreateAPIKey)
				admin.GET("/api-keys", managementHandler.ListAPIKeys)
				admin.DELETE("/api-keys/:id", managementHandler.RevokeAPIKey)
				admin.POST("/jwts/revoke", managementHandler.RevokeJWT)
				admin.GET("/jwts", managementHandler.ListJWTs)
				admin.GET("/logs", managementHandler.QueryLogs)
				admin.POST("/sync/force", syncHandler.ForceSync)
				admin.GET("/sync/status", syncHandler.SyncStatus)
				admin.GET("/users", managementHandler.ListUsers)
				admin.GET("/users/:id", managementHandler.GetUser)
				admin.PUT("/users/:id/access", managementHandler.UpdateUserAccess)
				admin.PUT("/users/:id/role", managementHandler.UpdateUserRole)
				admin.DELETE("/users/:id", managementHandler.DeleteUser)
				admin.POST("/hideout-modules/cleanup-duplicates", managementHandler.CleanupDuplicateHideoutModules)

				// Data Export (CSV) - Admin only
				admin.GET("/export/quests", exportHandler.ExportQuests)
				admin.GET("/export/items", exportHandler.ExportItems)
				admin.GET("/export/skill-nodes", exportHandler.ExportSkillNodes)
				admin.GET("/export/hideout-modules", exportHandler.ExportHideoutModules)
				admin.GET("/export/enemy-types", exportHandler.ExportEnemyTypes)
				admin.GET("/export/alerts", exportHandler.ExportAlerts)
				admin.GET("/export/bots", exportHandler.ExportBots)
				admin.GET("/export/maps", exportHandler.ExportMaps)
				admin.GET("/export/traders", exportHandler.ExportTraders)
				admin.GET("/export/projects", exportHandler.ExportProjects)

				// Admin Progress Management - View/Edit any user's progress
				admin.GET("/users/:id/progress", progressHandler.GetAllUserProgress)
				admin.GET("/users/:id/progress/quests", progressHandler.GetUserQuestProgress)
				admin.PUT("/users/:id/progress/quests/:quest_id", progressHandler.UpdateUserQuestProgress)
				admin.GET("/users/:id/progress/hideout-modules", progressHandler.GetUserHideoutModuleProgress)
				admin.PUT("/users/:id/progress/hideout-modules/:module_id", progressHandler.UpdateUserHideoutModuleProgress)
				admin.GET("/users/:id/progress/skill-nodes", progressHandler.GetUserSkillNodeProgress)
				admin.PUT("/users/:id/progress/skill-nodes/:skill_node_id", progressHandler.UpdateUserSkillNodeProgress)
				admin.GET("/users/:id/progress/blueprints", progressHandler.GetUserBlueprintProgress)
				admin.PUT("/users/:id/progress/blueprints/:item_id", progressHandler.UpdateUserBlueprintProgress)
			}

			// User profile routes (users can update their own profile, admins can update any)
			userProfile := writeProtected.Group("/users")
			userProfile.Use(middleware.JWTAuthMiddleware(authService, cfg, oidcService)) // Require JWT (already checked in writeProtected, but explicit)
			{
				userProfile.PUT("/:id/profile", managementHandler.UpdateUserProfile)
			}
		}

		// Health check endpoints
		healthHandler := handlers.NewHealthHandler(db, cacheService)
		r.GET("/health", healthHandler.HealthCheck)
		r.GET("/health/ready", healthHandler.ReadinessCheck)
		r.GET("/health/live", healthHandler.LivenessCheck)

		// Frontend config endpoint (public - returns safe config for frontend)
		configHandler := handlers.NewConfigHandler()
		r.GET("/api/v1/config", configHandler.GetFrontendConfig)

		// GraphQL API endpoint (requires authentication)
		graphqlGroup := api.Group("")
		graph.SetupGraphQLRoutes(
			graphqlGroup,
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
			cfg,
			oidcService,
		)

		// Mobile callback page (public route - redirects to deep link)
		r.GET("/auth/mobile-callback", authHandler.MobileCallbackPage)

		// Serve frontend static files
		frontendDir := "./frontend/out"
		if _, err := os.Stat(frontendDir); err == nil {
			// Serve static assets (CSS, JS, images) from _next directory
			r.StaticFS("/_next", gin.Dir(frontendDir+"/_next", false))

			// Serve dashboard and other frontend routes
			r.GET("/dashboard", func(c *gin.Context) {
				c.File(frontendDir + "/dashboard/index.html")
			})
			r.GET("/dashboard/*path", func(c *gin.Context) {
				path := c.Param("path")

				// Handle OAuth callback routes specially
				if strings.HasPrefix(path, "/api/auth/github/callback") {
					c.File(frontendDir + "/api/auth/github/callback/index.html")
					return
				}
				if strings.HasPrefix(path, "/api/auth/discord/callback") {
					c.File(frontendDir + "/api/auth/discord/callback/index.html")
					return
				}

				filePath := frontendDir + "/dashboard" + path
				if strings.HasSuffix(path, "/") || path == "" {
					filePath += "index.html"
				}
				if _, err := os.Stat(filePath); err == nil {
					c.File(filePath)
				} else {
					c.File(frontendDir + "/dashboard/index.html")
				}
			})

			// Serve other frontend routes (login, missions, etc.)
			r.GET("/login", func(c *gin.Context) {
				c.File(frontendDir + "/login/index.html")
			})
			r.GET("/login/*path", func(c *gin.Context) {
				c.File(frontendDir + "/login/index.html")
			})

			// Serve specific frontend routes
			r.GET("/api-keys", func(c *gin.Context) {
				c.File(frontendDir + "/api-keys/index.html")
			})
			r.GET("/api-keys/*path", func(c *gin.Context) {
				c.File(frontendDir + "/api-keys/index.html")
			})
			r.GET("/quests", func(c *gin.Context) {
				c.File(frontendDir + "/quests/index.html")
			})
			r.GET("/quests/*path", func(c *gin.Context) {
				c.File(frontendDir + "/quests/index.html")
			})
			r.GET("/items", func(c *gin.Context) {
				c.File(frontendDir + "/items/index.html")
			})
			r.GET("/items/*path", func(c *gin.Context) {
				c.File(frontendDir + "/items/index.html")
			})
			r.GET("/required-items", func(c *gin.Context) {
				c.File(frontendDir + "/required-items/index.html")
			})
			r.GET("/required-items/*path", func(c *gin.Context) {
				c.File(frontendDir + "/required-items/index.html")
			})
			r.GET("/skill-nodes", func(c *gin.Context) {
				c.File(frontendDir + "/skill-nodes/index.html")
			})
			r.GET("/skill-nodes/*path", func(c *gin.Context) {
				c.File(frontendDir + "/skill-nodes/index.html")
			})
			r.GET("/hideout-modules", func(c *gin.Context) {
				c.File(frontendDir + "/hideout-modules/index.html")
			})
			r.GET("/hideout-modules/*path", func(c *gin.Context) {
				c.File(frontendDir + "/hideout-modules/index.html")
			})
			r.GET("/enemy-types", func(c *gin.Context) {
				c.File(frontendDir + "/enemy-types/index.html")
			})
			r.GET("/enemy-types/*path", func(c *gin.Context) {
				c.File(frontendDir + "/enemy-types/index.html")
			})
			r.GET("/alerts", func(c *gin.Context) {
				c.File(frontendDir + "/alerts/index.html")
			})
			r.GET("/alerts/*path", func(c *gin.Context) {
				c.File(frontendDir + "/alerts/index.html")
			})
			r.GET("/users", func(c *gin.Context) {
				c.File(frontendDir + "/users/index.html")
			})
			r.GET("/users/*path", func(c *gin.Context) {
				c.File(frontendDir + "/users/index.html")
			})
			r.GET("/appwrite", func(c *gin.Context) {
				c.File(frontendDir + "/appwrite/index.html")
			})
			r.GET("/appwrite/*path", func(c *gin.Context) {
				c.File(frontendDir + "/appwrite/index.html")
			})
			r.GET("/export", func(c *gin.Context) {
				c.File(frontendDir + "/export/index.html")
			})
			r.GET("/export/*path", func(c *gin.Context) {
				c.File(frontendDir + "/export/index.html")
			})

			// Catch-all for other frontend routes
			r.NoRoute(func(c *gin.Context) {
				// If route doesn't start with /api or /health, try to serve frontend
				if !strings.HasPrefix(c.Request.URL.Path, "/api") &&
					!strings.HasPrefix(c.Request.URL.Path, "/health") &&
					!strings.HasPrefix(c.Request.URL.Path, "/_next") {
					path := c.Request.URL.Path
					filePath := frontendDir + path

					// Add /index.html if it's a directory route
					if strings.HasSuffix(path, "/") || path == "" || path == "/" {
						filePath += "index.html"
					} else if !strings.Contains(path, ".") {
						// If no extension, it's probably a route that needs index.html
						filePath += "/index.html"
					}

					// Check if file exists
					if _, err := os.Stat(filePath); err == nil {
						c.File(filePath)
					} else {
						// Fallback to root index.html for client-side routing
						c.File(frontendDir + "/index.html")
					}
				} else {
					c.JSON(404, gin.H{"error": "Not found"})
				}
			})
			log.Println("Frontend dashboard enabled at /dashboard")
		} else {
			log.Printf("Warning: Frontend not found at %s. Build frontend with 'make build-frontend'", frontendDir)
		}
	}

	// Create HTTP server with timeouts
	srv := &http.Server{
		Addr:           ":" + cfg.APIPort,
		Handler:        r,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.APIPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
