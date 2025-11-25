package handler

import (
	"net/http"

	"github.com/surdiana/gateway/internal/service"
	"github.com/surdiana/gateway/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CacheHandler struct {
	cacheService *service.CacheService
}

func NewCacheHandler(cacheService *service.CacheService) *CacheHandler {
	return &CacheHandler{
		cacheService: cacheService,
	}
}

// InvalidateCacheRequest represents request to invalidate cache
type InvalidateCacheRequest struct {
	Slug string `json:"slug" binding:"required"`
}

// InvalidateCacheResponse represents response for cache invalidation
type InvalidateCacheResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// InvalidateCache invalidates cache for a specific integration slug
func (h *CacheHandler) InvalidateCache(c *gin.Context) {
	var req InvalidateCacheRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.GetLogger().Error("Invalid request for cache invalidation",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, InvalidateCacheResponse{
			Success: false,
			Message: "Invalid request format",
		})
		return
	}

	if err := h.cacheService.InvalidateCache(c.Request.Context(), req.Slug); err != nil {
		logger.GetLogger().Error("Failed to invalidate cache",
			zap.String("slug", req.Slug),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, InvalidateCacheResponse{
			Success: false,
			Message: "Failed to invalidate cache",
		})
		return
	}

	logger.GetLogger().Info("Cache invalidated successfully",
		zap.String("slug", req.Slug),
	)

	c.JSON(http.StatusOK, InvalidateCacheResponse{
		Success: true,
		Message: "Cache invalidated successfully",
	})
}

// GetCacheStats returns cache statistics
func (h *CacheHandler) GetCacheStats(c *gin.Context) {
	stats, err := h.cacheService.GetCacheStats(c.Request.Context())
	if err != nil {
		logger.GetLogger().Error("Failed to get cache stats",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get cache statistics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// ClearAllCache clears all cache (admin only)
func (h *CacheHandler) ClearAllCache(c *gin.Context) {
	// This is a dangerous operation, so we might want additional checks
	// For now, we'll require a confirmation parameter
	confirm := c.Query("confirm")
	if confirm != "true" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Please add ?confirm=true to clear all cache",
		})
		return
	}

	if err := h.cacheService.ClearAll(c.Request.Context()); err != nil {
		logger.GetLogger().Error("Failed to clear all cache",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to clear all cache",
		})
		return
	}

	logger.GetLogger().Warn("All cache cleared by admin")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "All cache cleared successfully",
	})
}

// HealthCache checks Redis health
func (h *CacheHandler) HealthCache(c *gin.Context) {
	stats, err := h.cacheService.GetCacheStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"message": "Redis is not available",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"data":   stats,
	})
}