package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/Payphone-Digital/gateway/pkg/logger"
	"github.com/Payphone-Digital/gateway/pkg/redis"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db          *gorm.DB
	redisClient redis.Client
}

type HealthCheckResponse struct {
	Status    string                 `json:"status"`
	Version   string                 `json:"version"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]HealthCheck `json:"checks"`
}

type HealthCheck struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func NewHealthHandler(db *gorm.DB, redisClient redis.Client) *HealthHandler {
	return &HealthHandler{
		db:          db,
		redisClient: redisClient,
	}
}

// HealthCheck performs comprehensive health check
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response := HealthCheckResponse{
		Status:    "healthy",
		Version:   "1.0.0",
		Timestamp: time.Now(),
		Checks:    make(map[string]HealthCheck),
	}

	// Check Database
	dbStatus := h.checkDatabase(ctx)
	response.Checks["database"] = dbStatus
	if dbStatus.Status != "healthy" {
		response.Status = "unhealthy"
	}

	// Check Redis (if enabled)
	redisStatus := h.checkRedis(ctx)
	response.Checks["redis"] = redisStatus
	// Redis is optional, so don't mark overall status as unhealthy if Redis is down

	// Check API Responsiveness
	apiStatus := HealthCheck{
		Status:  "healthy",
		Message: "API is responsive",
	}
	response.Checks["api"] = apiStatus

	// Determine HTTP status code
	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	logger.GetLogger().Debug("Health check performed",
		zap.String("overall_status", response.Status),
		zap.Int("status_code", statusCode),
	)

	c.JSON(statusCode, response)
}

// BasicHealth returns a simple health check (for load balancers)
func (h *HealthHandler) BasicHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"version":   "1.0.0",
		"timestamp": time.Now(),
	})
}

func (h *HealthHandler) checkDatabase(ctx context.Context) HealthCheck {
	if h.db == nil {
		return HealthCheck{
			Status:  "unhealthy",
			Message: "Database connection not initialized",
		}
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		logger.GetLogger().Error("Failed to get DB instance for health check", zap.Error(err))
		return HealthCheck{
			Status:  "unhealthy",
			Message: "Failed to get database instance",
		}
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		logger.GetLogger().Error("Database ping failed", zap.Error(err))
		return HealthCheck{
			Status:  "unhealthy",
			Message: "Database ping failed: " + err.Error(),
		}
	}

	stats := sqlDB.Stats()
	return HealthCheck{
		Status: "healthy",
		Message: "Database connection is healthy (open: " +
			string(rune(stats.OpenConnections)) + ", idle: " +
			string(rune(stats.Idle)) + ")",
	}
}

func (h *HealthHandler) checkRedis(ctx context.Context) HealthCheck {
	if h.redisClient == nil || !h.redisClient.IsEnabled() {
		return HealthCheck{
			Status:  "disabled",
			Message: "Redis cache is disabled",
		}
	}

	if err := h.redisClient.Ping(ctx); err != nil {
		logger.GetLogger().Warn("Redis ping failed", zap.Error(err))
		return HealthCheck{
			Status:  "unhealthy",
			Message: "Redis ping failed: " + err.Error(),
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: "Redis connection is healthy",
	}
}
