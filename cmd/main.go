package main

import (
	"os"
	"os/signal"
	"syscall"

	configs "github.com/surdiana/gateway/config"
	"github.com/surdiana/gateway/internal/handler"
	"github.com/surdiana/gateway/internal/middleware"
	"github.com/surdiana/gateway/internal/repository"
	"github.com/surdiana/gateway/internal/router"
	"github.com/surdiana/gateway/internal/service"
	"github.com/surdiana/gateway/pkg/database"
	"github.com/surdiana/gateway/pkg/logger"
	"github.com/surdiana/gateway/pkg/redis"
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
		zap.String("version", "1.0.0"),
	)

	db := database.InitDatabase(config)
	defer database.CloseDB()

	// Repositories
	integrasiRepo := repository.NewAPIConfigRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Initialize Redis client
	redisClient, err := redis.NewClient(config)
	if err != nil {
		logger.GetLogger().Fatal("Failed to initialize Redis",
			zap.Error(err),
		)
	}
	defer redisClient.Close()

	// Services
	integrasiService := service.NewAPIConfigService(integrasiRepo)
	jwtService := service.NewJWTService(config.JWT.Secret)
	userService := service.NewUserServiceWithJWT(userRepo, jwtService)
	cacheService := service.NewCacheService(redisClient)

	// Handlers
	integrasiHandler := handler.NewAPIConfigHandler(integrasiService)
	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(userService)

	validationMiddleware := middleware.NewValidationMiddleware()
	jwtMiddleware := middleware.NewJWTMiddleware(jwtService, userRepo)
	dynamicURIMiddleware := middleware.NewDynamicURIMiddleware(integrasiService, cacheService)

	r := router.NewRouter(
		integrasiHandler,
		userHandler,
		authHandler,
		handler.NewCacheHandler(cacheService),

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
