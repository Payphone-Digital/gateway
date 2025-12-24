package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	configs "github.com/Payphone-Digital/gateway/config"
	"github.com/Payphone-Digital/gateway/internal/handler"
	"github.com/Payphone-Digital/gateway/internal/middleware"
	"github.com/Payphone-Digital/gateway/internal/repository"
	"github.com/Payphone-Digital/gateway/internal/router"
	"github.com/Payphone-Digital/gateway/internal/service"
	"github.com/Payphone-Digital/gateway/pkg/database"
	"github.com/Payphone-Digital/gateway/pkg/logger"
	"github.com/Payphone-Digital/gateway/pkg/redis"
	"github.com/Payphone-Digital/gateway/pkg/routing"
	"go.uber.org/zap"
)

func main() {
	config, err := configs.LoadConfig()
	if err != nil {
		panic("Failed to load config: " + err.Error())
	}

	// Initialize Zap logger
	if err := logger.InitLogger(config); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	logger.GetLogger().Info("Application starting",
		zap.String("app_name", config.App.Name),
		zap.String("environment", config.App.Environment),
		zap.String("version", "2.0.0"),
	)

	// Initialize database with standardized pattern
	db, err := database.NewPostgresDB(database.Config{
		Host:            config.Database.Host,
		Port:            config.Database.Port,
		User:            config.Database.User,
		Password:        config.Database.Password,
		Database:        config.Database.Name,
		SSLMode:         config.Database.SSLMode,
		MaxIdleConns:    config.Database.MaxIdleConns,
		MaxOpenConns:    config.Database.MaxOpenConns,
		ConnMaxLifetime: int(config.Database.ConnMaxLifetime.Minutes()),
		ConnMaxIdleTime: int(config.Database.ConnMaxIdleTime.Minutes()),
	})
	if err != nil {
		logger.GetLogger().Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.CloseDB(db)

	// Run auto migrations
	if err := database.AutoMigrate(db); err != nil {
		logger.GetLogger().Fatal("Failed to run database migrations", zap.Error(err))
	}
	logger.GetLogger().Info("Database migrated successfully")

	// Seed initial data
	if err := database.Seed(db); err != nil {
		logger.GetLogger().Error("Failed to seed database", zap.Error(err))
		// Don't fail - seed data may already exist
	} else {
		logger.GetLogger().Info("Database seeded successfully")
	}

	// Repositories
	integrasiRepo := repository.NewAPIConfigRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Initialize Redis client with new interface-based design
	redisClient := redis.NewClient(redis.Config{
		Host:         config.Redis.Host,
		Port:         config.Redis.Port,
		Password:     config.Redis.Password,
		DB:           config.Redis.Database,
		Enabled:      config.Redis.Enabled,
		PoolSize:     config.Redis.PoolSize,
		MinIdleConns: config.Redis.MinIdleConns,
	}, logger.GetLogger())
	defer redisClient.Close()

	logger.GetLogger().Info("Redis client initialized",
		zap.Bool("enabled", redisClient.IsEnabled()),
	)

	// Services
	integrasiService := service.NewAPIConfigService(integrasiRepo)
	jwtService := service.NewJWTService(config.JWT.Secret)
	userService := service.NewUserServiceWithJWT(userRepo, jwtService)

	// Initialize CacheService
	cacheService := service.NewCacheService(redisClient)

	// Initialize Route Registry
	registry := routing.NewRouteRegistry(logger.GetLogger())

	// Load all routes from database at startup
	logger.GetLogger().Info("Initializing route registry")
	refresher := routing.NewRefresher(registry, integrasiService, logger.GetLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := refresher.Refresh(ctx); err != nil {
		logger.GetLogger().Warn("Failed to load routes into registry at startup",
			zap.Error(err),
			zap.Int("route_count", registry.Count()),
		)
		// Don't fail - registry may be empty but service should still start
	} else {
		logger.GetLogger().Info("Route registry initialized successfully",
			zap.Int("route_count", registry.Count()),
		)
	}

	// Handlers
	integrasiHandler := handler.NewAPIConfigHandler(integrasiService, refresher, cacheService)
	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(userService)
	healthHandler := handler.NewHealthHandler(db, redisClient)

	// Initialize middleware
	validationMiddleware := middleware.NewValidationMiddleware()
	jwtMiddleware := middleware.NewJWTMiddleware(jwtService, userRepo)
	dynamicURIMiddleware := middleware.NewDynamicURIMiddleware(registry, cacheService, jwtMiddleware)

	r := router.NewRouter(
		db,
		integrasiHandler,
		userHandler,
		authHandler,
		handler.NewCacheHandler(cacheService),
		healthHandler,

		validationMiddleware,
		jwtMiddleware,
		dynamicURIMiddleware,
		config,
	).SetupRoutes()

	go func() {
		logger.GetLogger().Info("Server starting",
			zap.String("port", config.App.Port),
			zap.String("host", "0.0.0.0"),
		)
		if err := r.Run(":" + config.App.Port); err != nil {
			logger.GetLogger().Fatal("Failed to start server",
				zap.Error(err),
				zap.String("port", config.App.Port),
			)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.GetLogger().Info("Shutting down server...")
}
