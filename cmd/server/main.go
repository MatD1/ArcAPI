package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
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
	questRepo := repository.NewQuestRepository(db)
	itemRepo := repository.NewItemRepository(db)
	skillNodeRepo := repository.NewSkillNodeRepository(db)
	hideoutModuleRepo := repository.NewHideoutModuleRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)

	// Initialize services
	authService := services.NewAuthService(userRepo, apiKeyRepo, jwtTokenRepo, cacheService, cfg)
	userService := services.NewUserService(userRepo)

	// Initialize sync service
	syncService := services.NewSyncService(
		questRepo,
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
	authHandler := handlers.NewAuthHandler(authService, userService, cfg, apiKeyRepo)
	questHandler := handlers.NewQuestHandler(questRepo)
	missionHandler := questHandler // Backward compatibility - uses questHandler internally
	itemHandler := handlers.NewItemHandler(itemRepo)
	skillNodeHandler := handlers.NewSkillNodeHandler(skillNodeRepo)
	hideoutModuleHandler := handlers.NewHideoutModuleHandler(hideoutModuleRepo)
	managementHandler := handlers.NewManagementHandler(
		authService,
		apiKeyRepo,
		jwtTokenRepo,
		auditLogRepo,
		userRepo,
	)
	syncHandler := handlers.NewSyncHandler(syncService)

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

			// Skill Nodes - Read
			readOnly.GET("/skill-nodes", skillNodeHandler.List)
			readOnly.GET("/skill-nodes/:id", skillNodeHandler.Get)

			// Hideout Modules - Read
			readOnly.GET("/hideout-modules", hideoutModuleHandler.List)
			readOnly.GET("/hideout-modules/:id", hideoutModuleHandler.Get)
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
			}
		}

		// Health check endpoint
		r.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

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

				// Handle OAuth callback route specially
				if strings.HasPrefix(path, "/api/auth/github/callback") {
					c.File(frontendDir + "/api/auth/github/callback/index.html")
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
}
