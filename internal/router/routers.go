package router

import (
	"time"

	"github.com/surdiana/gateway/config"
	"github.com/surdiana/gateway/internal/handler"
	"github.com/surdiana/gateway/internal/middleware"
	"github.com/gin-gonic/gin"
)

type Router struct {
	IntegrasiHandler *handler.APIConfigHandler
	userHandler      *handler.UserHandler
	authHandler      *handler.AuthHandler
	cacheHandler     *handler.CacheHandler

	validMw               *middleware.ValidationMiddleware
	jwtMw                 *middleware.JWTMiddleware
	dynamicURIMiddleware  *middleware.DynamicURIMiddleware
	Config                *config.Config
}

func NewRouter(
	integrasi *handler.APIConfigHandler,
	user *handler.UserHandler,
	auth *handler.AuthHandler,
	cache *handler.CacheHandler,

	validMw *middleware.ValidationMiddleware,
	jwtMw *middleware.JWTMiddleware,
	dynamicURIMw *middleware.DynamicURIMiddleware,
	config *config.Config,
) *Router {
	return &Router{
		IntegrasiHandler: integrasi,
		userHandler:      user,
		authHandler:      auth,
		cacheHandler:     cache,

		validMw:               validMw,
		jwtMw:                 jwtMw,
		dynamicURIMiddleware:  dynamicURIMw,
		Config:                config,
	}
}

func (r *Router) SetupRoutes() *gin.Engine {
	// Create Gin router
	router := gin.Default()

	// Use custom logging and recovery middleware
	router.Use(middleware.LoggingMiddleware())
	router.Use(middleware.RecoveryMiddleware())
	router.Use(middleware.RequestResponseMiddleware())
	router.Use(middleware.SecurityLoggingMiddleware())
	router.Use(middleware.CORS())

	api := router.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":    "healthy",
				"version":   "1.0.0",
				"timestamp": time.Now(),
			})
		})
		v1 := api.Group("/v1")
		{
			v1.Use(middleware.RateLimit(r.Config.RateLimit.Request, time.Duration(r.Config.RateLimit.Duration)*time.Second))

			r.authRoutes(v1)
			r.userRoutes(v1)
			r.integrasiRoutes(v1)
			r.cacheRoutes(v1)

			// r.smtpRoutes(v1)
			// r.templateRoutes(v1)

		}
	}

	return router
}

// cacheRoutes defines cache management routes
func (r *Router) cacheRoutes(rg *gin.RouterGroup) {
	cache := rg.Group("/cache")
	{
		// Public routes (no authentication required)
		cache.GET("/health", r.cacheHandler.HealthCache)

		// Protected routes (JWT authentication required)
		protected := cache.Group("")
		protected.Use(r.jwtMw.RequireAuth())
		{
			// Cache statistics
			protected.GET("/stats", r.cacheHandler.GetCacheStats)

			// Cache invalidation
			protected.POST("/invalidate", r.cacheHandler.InvalidateCache)

			// Clear all cache (admin only)
			protected.DELETE("/clear", r.cacheHandler.ClearAllCache)
		}
	}
}

// func SetupRoutes(r *gin.Engine) {

// 	r.Use(middleware.ValidationMiddleware())

// 	maxRequest, _ := strconv.Atoi(os.Getenv("RATE_LIMIT_MAX_REQUEST"))
// 	duration, _ := strconv.Atoi(os.Getenv("RATE_LIMIT_DURATION"))
// 	emailMaxRequest, _ := strconv.Atoi(os.Getenv("EMAIL_RATE_LIMIT_MAX_REQUEST"))
// 	emailDuration, _ := strconv.Atoi(os.Getenv("EMAIL_RATE_LIMIT_DURATION"))

// 	// API group
// 	api := r.Group("/api")
// 	{
// 		api.GET("/health", func(c *gin.Context) {
// 			c.JSON(200, gin.H{
// 				"status":    "healthy",
// 				"version":   "1.0.0",
// 				"timestamp": time.Now(),
// 			})
// 		})
// 		v1 := api.Group("/v1")
// 		{

// 			v1.Use(middleware.RateLimit(maxRequest, time.Duration(duration)*time.Second))

// 			// Email routes
// 			email := v1.Group("/email")
// 			email.Use(middleware.RateLimit(emailMaxRequest, time.Duration(emailDuration)*time.Second))
// 			{
// 				email.POST("/send", emailHandler.SendEmail)
// 				email.POST("/send/:configId", emailHandler.SendEmailWithConfig)
// 				email.POST("/retry/:historyId", emailHandler.RetryFailedEmail)
// 			}

// 			// SMTP Config routes
// 			smtp := v1.Group("/smtp")
// 			{
// 				smtp.POST("", smtpHandler.CreateConfig)
// 				smtp.PUT(":id", smtpHandler.UpdateConfig)
// 				smtp.DELETE("/:id", smtpHandler.DeleteConfig)
// 				smtp.GET("/:id", smtpHandler.GetConfig)
// 				smtp.GET("", smtpHandler.GetAllConfigs)
// 			}

// 			template := v1.Group("/template")
// 			{
// 				template.POST("", templateHandler.CreateTemplate)
// 				template.PUT("/:id", templateHandler.UpdateTemplate)
// 				template.DELETE("/:id", templateHandler.DeleteTemplate)
// 				template.GET("/:id", templateHandler.GetTemplate)
// 				template.GET("", templateHandler.GetAllTemplates)
// 			}
// 		}
// 	}
// }
