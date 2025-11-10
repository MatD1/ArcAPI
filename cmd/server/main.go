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

	// Initialize services
	authService := services.NewAuthService(userRepo, apiKeyRepo, jwtTokenRepo, cacheService, cfg)
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
			dataCacheService,
			cfg,
		)
	} else {
		syncService = services.NewSyncService(
			questRepo,
			itemRepo,
			skillNodeRepo,
			hideoutModuleRepo,
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
	authHandler := handlers.NewAuthHandler(authService, userService, cfg, apiKeyRepo)

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
		}

		// Read-only routes (require JWT only)
		readOnly := api.Group("")
		readOnly.Use(middleware.JWTAuthMiddleware(authService))
		{
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
		}

		// Progress routes (basic users can read and update their own progress)
		progress := api.Group("/progress")
		progress.Use(middleware.ProgressAuthMiddleware(authService))
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
		writeProtected.Use(middleware.WriteAuthMiddleware(authService))
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
				admin.DELETE("/users/:id", managementHandler.DeleteUser)
				admin.POST("/hideout-modules/cleanup-duplicates", managementHandler.CleanupDuplicateHideoutModules)
			}

			// User profile routes (users can update their own profile, admins can update any)
			userProfile := writeProtected.Group("/users")
			userProfile.Use(middleware.JWTAuthMiddleware(authService)) // Require JWT (already checked in writeProtected, but explicit)
			{
				userProfile.PUT("/:id/profile", managementHandler.UpdateUserProfile)
			}
		}

		// Health check endpoints
		healthHandler := handlers.NewHealthHandler(db, cacheService)
		r.GET("/health", healthHandler.HealthCheck)
		r.GET("/health/ready", healthHandler.ReadinessCheck)
		r.GET("/health/live", healthHandler.LivenessCheck)

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
