package main

import (
	"context"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
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
	// Register WASM MIME type for proper browser loading
	mime.AddExtensionType(".wasm", "application/wasm")
	
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
	
	// Supabase Authentication Service (Replaces Authentik OIDC)
	supabaseAuthService, err := services.NewSupabaseAuthService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Supabase auth service: %v", err)
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
	authHandler := handlers.NewAuthHandler(authService, userService, cfg, apiKeyRepo)

	// Use cache-enabled handlers if cache is available
	var questHandler *handlers.QuestHandler
	if dataCacheService != nil {
		questHandler = handlers.NewQuestHandlerWithCache(questRepo, dataCacheService)
	} else {
		questHandler = handlers.NewQuestHandler(questRepo)
	}
	missionHandler := questHandler // Backward compatibility

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

	// Security middleware
	r.Use(middleware.SecurityMiddleware(cfg.GetAllowedOrigins()))

	// Logger middleware
	r.Use(middleware.LoggerMiddleware(auditLogRepo))

	// Public routes
	api := r.Group("/api/v1")
	api.Use(middleware.RateLimitMiddleware(cacheService, cfg.RateLimitRequests, cfg.RateLimitWindowSeconds))
	{
		// Sync Snapshot (Public - game data only, no sensitive info)
		api.GET("/sync/snapshot", syncHandler.GetSnapshot)

		// JWTAuthMiddleware handles Supabase JWT validation
		readOnly := api.Group("")
		readOnly.Use(middleware.JWTAuthMiddleware(authService, cfg, supabaseAuthService))
		{
			readOnly.GET("/me", authHandler.GetCurrentUser)
			// Quests - Read
			readOnly.GET("/quests", questHandler.List)
			readOnly.GET("/quests/:id", questHandler.Get)
			// Backward compatibility
			readOnly.GET("/missions", missionHandler.List)
			readOnly.GET("/missions/:id", missionHandler.Get)

			// Items - Read
			readOnly.GET("/items", itemHandler.List)
			readOnly.GET("/items/:id", itemHandler.Get)
			readOnly.GET("/items/required", itemHandler.RequiredItems)
			readOnly.GET("/items/blueprints", itemHandler.GetBlueprints)

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
			readOnly.GET("/alerts/active", alertHandler.GetActive)
			readOnly.GET("/alerts/:id", alertHandler.Get)

			// Traders - Read
			if tradersHandler != nil {
				readOnly.GET("/traders", tradersHandler.GetTraders)
			}
			readOnly.GET("/bots", botHandler.List)
			readOnly.GET("/bots/:id", botHandler.Get)
			readOnly.GET("/maps", mapHandler.List)
			readOnly.GET("/maps/:id", mapHandler.Get)
			readOnly.GET("/repo-traders", traderHandler.List)
			readOnly.GET("/repo-traders/:id", traderHandler.Get)
			readOnly.GET("/projects", projectHandler.List)
			readOnly.GET("/projects/:id", projectHandler.Get)
		}

		// Progress routes
		progress := api.Group("/progress")
		progress.Use(middleware.ProgressAuthMiddleware(authService, cfg, supabaseAuthService))
		{
			progress.GET("/quests", progressHandler.GetMyQuestProgress)
			progress.PUT("/quests/:quest_id", progressHandler.UpdateQuestProgress)
			progress.GET("/hideout-modules", progressHandler.GetMyHideoutModuleProgress)
			progress.PUT("/hideout-modules/:module_id", progressHandler.UpdateHideoutModuleProgress)
			progress.GET("/skill-nodes", progressHandler.GetMySkillNodeProgress)
			progress.PUT("/skill-nodes/:skill_node_id", progressHandler.UpdateSkillNodeProgress)
			progress.GET("/blueprints", progressHandler.GetMyBlueprintProgress)
			progress.PUT("/blueprints/:item_id", progressHandler.UpdateBlueprintProgress)
		}

		// Write routes
		writeProtected := api.Group("")
		writeProtected.Use(middleware.WriteAuthMiddleware(authService, cfg, supabaseAuthService))
		{
			writeProtected.POST("/quests", questHandler.Create)
			writeProtected.PUT("/quests/:id", questHandler.Update)
			writeProtected.DELETE("/quests/:id", questHandler.Delete)
			writeProtected.POST("/missions", missionHandler.Create)
			writeProtected.PUT("/missions/:id", missionHandler.Update)
			writeProtected.DELETE("/missions/:id", missionHandler.Delete)

			writeProtected.POST("/items", itemHandler.Create)
			writeProtected.PUT("/items/:id", itemHandler.Update)
			writeProtected.DELETE("/items/:id", itemHandler.Delete)

			writeProtected.POST("/skill-nodes", skillNodeHandler.Create)
			writeProtected.PUT("/skill-nodes/:id", skillNodeHandler.Update)
			writeProtected.DELETE("/skill-nodes/:id", skillNodeHandler.Delete)

			writeProtected.POST("/hideout-modules", hideoutModuleHandler.Create)
			writeProtected.PUT("/hideout-modules/:id", hideoutModuleHandler.Update)
			writeProtected.DELETE("/hideout-modules/:id", hideoutModuleHandler.Delete)

			writeProtected.POST("/enemy-types", enemyTypeHandler.Create)
			writeProtected.PUT("/enemy-types/:id", enemyTypeHandler.Update)
			writeProtected.DELETE("/enemy-types/:id", enemyTypeHandler.Delete)

			writeProtected.POST("/alerts", alertHandler.Create)
			writeProtected.PUT("/alerts/:id", alertHandler.Update)
			writeProtected.DELETE("/alerts/:id", alertHandler.Delete)

			admin := writeProtected.Group("/admin")
			admin.Use(middleware.AdminMiddleware())
			{
				admin.POST("/api-keys", managementHandler.CreateAPIKey)
				admin.GET("/api-keys", managementHandler.ListAPIKeys)
				admin.DELETE("/api-keys/:id", managementHandler.RevokeAPIKey)
				admin.GET("/logs", managementHandler.QueryLogs)
				admin.POST("/sync/force", syncHandler.ForceSync)
				admin.GET("/sync/status", syncHandler.SyncStatus)
				admin.GET("/users", managementHandler.ListUsers)
				admin.GET("/users/:id", managementHandler.GetUser)
				admin.PUT("/users/:id/access", managementHandler.UpdateUserAccess)
				admin.PUT("/users/:id/role", managementHandler.UpdateUserRole)
				admin.DELETE("/users/:id", managementHandler.DeleteUser)
				admin.POST("/hideout-modules/cleanup-duplicates", managementHandler.CleanupDuplicateHideoutModules)

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

			userProfile := writeProtected.Group("/users")
			userProfile.Use(middleware.JWTAuthMiddleware(authService, cfg, supabaseAuthService))
			{
				userProfile.PUT("/:id/profile", managementHandler.UpdateUserProfile)
			}
		}

		// Health endpoints
		healthHandler := handlers.NewHealthHandler(db, cacheService)
		r.GET("/health", healthHandler.HealthCheck)
		r.GET("/health/ready", healthHandler.ReadinessCheck)
		r.GET("/health/live", healthHandler.LivenessCheck)

		// Config endpoint
		configHandler := handlers.NewConfigHandler()
		r.GET("/api/v1/config", configHandler.GetFrontendConfig)

		// GraphQL
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
			supabaseAuthService,
		)

	}

	// Serve frontend static files
	frontendDir := "./frontend/out"
	if _, err := os.Stat(frontendDir); err == nil {
		absFrontendDir, _ := filepath.Abs(frontendDir)
		dashboardBase := filepath.Join(absFrontendDir, "dashboard")
		nextDir := filepath.Join(absFrontendDir, "_next")
		
		r.StaticFS("/_next", gin.Dir(nextDir, false))
		r.GET("/dashboard", func(c *gin.Context) {
			c.File(filepath.Join(dashboardBase, "index.html"))
		})
		r.GET("/dashboard/*path", func(c *gin.Context) {
			pathParam := c.Param("path")
			normalizedPath := path.Clean(pathParam)
			if !strings.HasPrefix(normalizedPath, "/") {
				normalizedPath = "/" + normalizedPath
			}
			
			candidate := filepath.Join(dashboardBase, normalizedPath)
			if strings.HasSuffix(normalizedPath, "/") || normalizedPath == "" {
				candidate = filepath.Join(candidate, "index.html")
			}
			if _, err := os.Stat(candidate); err == nil {
				c.File(candidate)
			} else {
				c.File(filepath.Join(dashboardBase, "index.html"))
			}
		})
		
		// Catch-all
		r.NoRoute(func(c *gin.Context) {
			if !strings.HasPrefix(c.Request.URL.Path, "/api") &&
				!strings.HasPrefix(c.Request.URL.Path, "/health") &&
				!strings.HasPrefix(c.Request.URL.Path, "/_next") {
				c.File(filepath.Join(absFrontendDir, "index.html"))
			} else {
				c.JSON(404, gin.H{"error": "Not found"})
			}
		})
	}

	// Server start
	srv := &http.Server{
		Addr:           ":" + cfg.APIPort,
		Handler:        r,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.APIPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("Server exited")
}
