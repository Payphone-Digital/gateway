package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/Payphone-Digital/gateway/internal/constants"
	"github.com/Payphone-Digital/gateway/internal/dto"
	"github.com/Payphone-Digital/gateway/internal/service"
	"github.com/Payphone-Digital/gateway/pkg/logger"
	"github.com/Payphone-Digital/gateway/pkg/routing"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type APIConfigHandler struct {
	integrasiService *service.APIConfigService
	routeRefresher   *routing.Refresher
	cacheService     *service.CacheService
}

// NewAPIConfigHandler creates a new API config handler
// routeRefresher is optional - if nil, routes will not be auto-refreshed on CRUD
func NewAPIConfigHandler(service *service.APIConfigService, routeRefresher *routing.Refresher, cacheService *service.CacheService) *APIConfigHandler {
	return &APIConfigHandler{
		integrasiService: service,
		routeRefresher:   routeRefresher,
		cacheService:     cacheService,
	}
}

// Start Create

func (h *APIConfigHandler) CreateConfig(c *gin.Context) {
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	logger.GetLogger().Info("Create API config request",
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("client_ip", clientIP),
		zap.String("user_agent", userAgent),
	)

	var req dto.APIConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.GetLogger().Warn("Invalid JSON in create config request",
			zap.String("client_ip", clientIP),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid request", err.Error()))
		return
	}

	// Log input data (sanitize sensitive data)
	logger.GetLogger().Info("Creating API config",
		zap.String("path", req.Path),
		zap.String("method", req.Method),
		zap.Uint("url_config_id", req.URLConfigID),
		zap.String("uri", req.URI),
		zap.Int("max_retries", req.MaxRetries),
		zap.Int("retry_delay", req.RetryDelay),
		zap.Int("timeout", req.Timeout),
		zap.String("client_ip", clientIP),
		zap.Bool("has_headers", len(req.Headers) > 0),
		zap.Bool("has_query_params", len(req.QueryParams) > 0),
		zap.Bool("has_body", req.Body != nil),
	)

	// Create context with timeout for the operation
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	status, err := h.integrasiService.CreateConfig(ctx, req)
	if err != nil {
		logger.GetLogger().Error("Failed to create API config",
			zap.String("path", req.Path),
			zap.String("client_ip", clientIP),
			zap.Int("http_status", status),
			zap.Error(err),
		)

		// Check if it's a timeout error
		if ctx.Err() == context.DeadlineExceeded {
			c.JSON(http.StatusRequestTimeout, constants.BuildErrorResponse("Request timeout", "Operation took too long"))
			return
		}

		c.JSON(status, constants.BuildErrorResponse("Create failed", err.Error()))
		return
	}

	logger.GetLogger().Info("API config created successfully",
		zap.String("path", req.Path),
		zap.String("client_ip", clientIP),
	)

	// Refresh route registry if refresher is available
	if h.routeRefresher != nil {
		go func() {
			refreshCtx, refreshCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer refreshCancel()
			if err := h.routeRefresher.RefreshSingle(refreshCtx, req.Path, req.Method); err != nil {
				logger.GetLogger().Warn("Failed to refresh route registry after create",
					zap.String("path", req.Path),
					zap.Error(err),
				)
			} else {
				logger.GetLogger().Info("Route registry refreshed after create",
					zap.String("path", req.Path),
				)
			}
		}()
	}

	c.JSON(status, constants.BuildSuccessResponse("Create successful"))
}

// DISABLED: Create API group handler
// func (h *APIConfigHandler) CreateGroup(c *gin.Context) {
// 	clientIP := c.ClientIP()
// 	userAgent := c.GetHeader("User-Agent")

// 	logger.GetLogger().Info("Create API group request",
// 		zap.String("method", c.Request.Method),
// 		zap.String("path", c.Request.URL.Path),
// 		zap.String("client_ip", clientIP),
// 		zap.String("user_agent", userAgent),
// 	)

// 	var req dto.APIGroupRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		logger.GetLogger().Warn("Invalid JSON in create group request",
// 			zap.String("client_ip", clientIP),
// 			zap.Error(err),
// 		)
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request", "details": err.Error()})
// 		return
// 	}

// 	logger.GetLogger().Info("Creating API group",
// 		zap.String("name", req.Name),
// 		zap.String("path", req.Path),
// 		zap.Bool("is_admin", req.IsAdmin),
// 		zap.String("client_ip", clientIP),
// 	)

// 	// Create context with timeout for the operation
// 	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
// 	defer cancel()

// 	status, err := h.integrasiService.CreateGroup(ctx, req)
// 	if err != nil {
// 		logger.GetLogger().Error("Failed to create API group",
// 			zap.String("name", req.Name),
// 			zap.String("client_ip", clientIP),
// 			zap.Int("http_status", status),
// 			zap.Error(err),
// 		)

// 		// Check if it's a timeout error
// 		if ctx.Err() == context.DeadlineExceeded {
// 			c.JSON(http.StatusRequestTimeout, gin.H{"message": "Request timeout", "details": "Operation took too long"})
// 			return
// 		}

// 		c.JSON(status, gin.H{"message": "Create failed", "details": err.Error()})
// 		return
// 	}

// 	logger.GetLogger().Info("API group created successfully",
// 		zap.String("name", req.Name),
// 		zap.String("client_ip", clientIP),
// 	)

// 	c.JSON(status, gin.H{"message": "Create successful"})
// }

// DISABLED: Create API group step handler
// func (h *APIConfigHandler) CreateGroupStep(c *gin.Context) {
// 	clientIP := c.ClientIP()
// 	userAgent := c.GetHeader("User-Agent")

// 	logger.GetLogger().Info("Create API group step request",
// 		zap.String("method", c.Request.Method),
// 		zap.String("path", c.Request.URL.Path),
// 		zap.String("client_ip", clientIP),
// 		zap.String("user_agent", userAgent),
// 	)

// 	var req dto.APIGroupStepRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		logger.GetLogger().Warn("Invalid JSON in create group step request",
// 			zap.String("client_ip", clientIP),
// 			zap.Error(err),
// 		)
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request", "details": err.Error()})
// 		return
// 	}

// 	logger.GetLogger().Info("Creating API group step",
// 		zap.Uint("group_id", req.GroupID),
// 		zap.Uint("api_config_id", req.APIConfigID),
// 		zap.String("alias", req.Alias),
// 		zap.Int("order_index", req.OrderIndex),
// 		zap.String("client_ip", clientIP),
// 	)

// 	// Create context with timeout for the operation
// 	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
// 	defer cancel()

// 	status, err := h.integrasiService.CreateGroupStep(ctx, req)
// 	if err != nil {
// 		logger.GetLogger().Error("Failed to create API group step",
// 			zap.Uint("group_id", req.GroupID),
// 			zap.Uint("api_config_id", req.APIConfigID),
// 			zap.String("alias", req.Alias),
// 			zap.String("client_ip", clientIP),
// 			zap.Int("http_status", status),
// 			zap.Error(err),
// 		)

// 		// Check if it's a timeout error
// 		if ctx.Err() == context.DeadlineExceeded {
// 			c.JSON(http.StatusRequestTimeout, gin.H{"message": "Request timeout", "details": "Operation took too long"})
// 			return
// 		}

// 		c.JSON(status, gin.H{"message": "Create failed", "details": err.Error()})
// 		return
// 	}

// 	logger.GetLogger().Info("API group step created successfully",
// 		zap.Uint("group_id", req.GroupID),
// 		zap.Uint("api_config_id", req.APIConfigID),
// 		zap.String("alias", req.Alias),
// 		zap.String("client_ip", clientIP),
// 	)

// 	c.JSON(status, gin.H{"message": "Create successful"})
// }

// DISABLED: Create API group cron handler
// func (h *APIConfigHandler) CreateGroupCron(c *gin.Context) {
// 	clientIP := c.ClientIP()
// 	userAgent := c.GetHeader("User-Agent")

// 	logger.GetLogger().Info("Create API group cron request",
// 		zap.String("method", c.Request.Method),
// 		zap.String("path", c.Request.URL.Path),
// 		zap.String("client_ip", clientIP),
// 		zap.String("user_agent", userAgent),
// 	)

// 	var req dto.APIGroupCronRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		logger.GetLogger().Warn("Invalid JSON in create group cron request",
// 			zap.String("client_ip", clientIP),
// 			zap.Error(err),
// 		)
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request", "details": err.Error()})
// 		return
// 	}

// 	logger.GetLogger().Info("Creating API group cron",
// 		zap.String("path", req.Path),
// 		zap.String("schedule", req.Schedule),
// 		zap.Bool("enabled", req.Enabled),
// 		zap.String("client_ip", clientIP),
// 	)

// 	// Create context with timeout for the operation
// 	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
// 	defer cancel()

// 	status, err := h.integrasiService.CreateGroupCron(ctx, req)
// 	if err != nil {
// 		logger.GetLogger().Error("Failed to create API group cron",
// 			zap.String("path", req.Path),
// 			zap.String("client_ip", clientIP),
// 			zap.Int("http_status", status),
// 			zap.Error(err),
// 		)

// 		// Check if it's a timeout error
// 		if ctx.Err() == context.DeadlineExceeded {
// 			c.JSON(http.StatusRequestTimeout, gin.H{"message": "Request timeout", "details": "Operation took too long"})
// 			return
// 		}

// 		c.JSON(status, gin.H{"message": "Create failed", "details": err.Error()})
// 		return
// 	}

// 	logger.GetLogger().Info("API group cron created successfully",
// 		zap.String("path", req.Path),
// 		zap.String("client_ip", clientIP),
// 	)

// 	c.JSON(status, gin.H{"message": "Create successful"})
// }

// End Create

// Start Update
func (h *APIConfigHandler) UpdateConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid ID", ""))
		return
	}

	var req dto.APIConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid request", err.Error()))
		return
	}

	// Create context with timeout for the operation
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Fetch OLD config BEFORE update to know old path/method for route invalidation
	oldConfig, _, _ := h.integrasiService.GetByIDConfig(ctx, uint(id))
	var oldPath, oldMethod string
	if oldConfig != nil {
		oldPath = oldConfig.Path
		oldMethod = oldConfig.Method
	}

	status, err := h.integrasiService.UpdateConfig(ctx, uint(id), req)
	if err != nil {
		// Check if it's a timeout error
		if ctx.Err() == context.DeadlineExceeded {
			c.JSON(http.StatusRequestTimeout, constants.BuildErrorResponse("Request timeout", "Operation took too long"))
			return
		}
		c.JSON(status, constants.BuildErrorResponse("Update failed", err.Error()))
		return
	}

	// Fetch updated config to ensure we have the correct path/slug for refresh
	// Use ID to get the source of truth from DB, avoiding prefix mismatches (e.g. /api/ vs /)
	updatedConfig, _, err := h.integrasiService.GetByIDConfig(ctx, uint(id))
	if err != nil {
		logger.GetLogger().Warn("Failed to fetch updated config for refresh",
			zap.Uint("config_id", uint(id)),
			zap.Error(err),
		)
		// Fallback to req.Path if fetch fails, though risk of staleness exists
	}

	refreshPath := req.Path
	refreshMethod := req.Method
	if updatedConfig != nil {
		refreshPath = updatedConfig.Path
		refreshMethod = updatedConfig.Method
	}

	// Refresher and Cache Invalidation
	go func() {
		refreshCtx, refreshCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer refreshCancel()

		// 1. Invalidate OLD route first (if path or method changed)
		if h.routeRefresher != nil && oldPath != "" {
			if oldPath != refreshPath || oldMethod != refreshMethod {
				_ = h.routeRefresher.InvalidateRoute(oldPath, oldMethod) // Remove old route
				logger.GetLogger().Info("Old route invalidated during update",
					zap.String("old_path", oldPath),
					zap.String("old_method", oldMethod),
				)
			}
		}

		// 2. Add/Update new route
		if h.routeRefresher != nil {
			if err := h.routeRefresher.RefreshSingle(refreshCtx, refreshPath, refreshMethod); err != nil {
				logger.GetLogger().Warn("Failed to refresh route registry after update",
					zap.String("path", refreshPath),
					zap.Error(err),
				)
			} else {
				logger.GetLogger().Info("Route registry refreshed after update",
					zap.String("path", refreshPath),
				)
			}
		}

		// 2. Invalidate Cache
		if h.cacheService != nil {
			if err := h.cacheService.InvalidateCache(refreshCtx, refreshPath); err != nil {
				logger.GetLogger().Warn("Failed to invalidate cache after update",
					zap.String("path", refreshPath),
					zap.Error(err),
				)
			} else {
				logger.GetLogger().Info("Cache invalidated after update",
					zap.String("path", refreshPath),
				)
			}
		}
	}()

	c.JSON(status, constants.BuildSuccessResponse("Update successful"))
}

// DISABLED: Update API group handler
// func (h *APIConfigHandler) UpdateGroup(c *gin.Context) {
// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid ID"})
// 		return
// 	}

// 	var req dto.APIGroupRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request", "details": err.Error()})
// 		return
// 	}

// 	status, err := h.integrasiService.UpdateGroup(uint(id), req)
// 	if err != nil {
// 		c.JSON(status, gin.H{"message": "Update failed", "details": err.Error()})
// 		return
// 	}
// 	c.JSON(status, gin.H{"message": "Update successful"})
// }

// DISABLED: Update API group step handler
// func (h *APIConfigHandler) UpdateGroupStep(c *gin.Context) {
// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid ID"})
// 		return
// 	}

// 	var req dto.APIGroupStepRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request", "details": err.Error()})
// 		return
// 	}

// 	status, err := h.integrasiService.UpdateGroupStep(uint(id), req)
// 	if err != nil {
// 		c.JSON(status, gin.H{"message": "Update failed", "details": err.Error()})
// 		return
// 	}
// 	c.JSON(status, gin.H{"message": "Update successful"})
// }

// DISABLED: Update API group cron handler
// func (h *APIConfigHandler) UpdateGroupCron(c *gin.Context) {
// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid ID"})
// 		return
// 	}

// 	var req dto.APIGroupCronRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request", "details": err.Error()})
// 		return
// 	}

// 	status, err := h.integrasiService.UpdateGroupCron(uint(id), req)
// 	if err != nil {
// 		c.JSON(status, gin.H{"message": "Update failed", "details": err.Error()})
// 		return
// 	}
// 	c.JSON(status, gin.H{"message": "Update successful"})
// }

// END Update

// Start Delete
func (h *APIConfigHandler) DeleteConfig(c *gin.Context) {
	clientIP := c.ClientIP()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.GetLogger().Warn("Invalid ID in delete config request",
			zap.String("client_ip", clientIP),
			zap.String("id_param", c.Param("id")),
		)
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid ID", ""))
		return
	}

	logger.GetLogger().Info("Delete API config request",
		zap.Uint("config_id", uint(id)),
		zap.String("client_ip", clientIP),
	)

	// Create context with timeout for the operation
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	status, err := h.integrasiService.DeleteConfig(ctx, uint(id))
	if err != nil {
		logger.GetLogger().Error("Failed to delete API config",
			zap.Uint("config_id", uint(id)),
			zap.String("client_ip", clientIP),
			zap.Int("http_status", status),
			zap.Error(err),
		)

		// Check if it's a timeout error
		if ctx.Err() == context.DeadlineExceeded {
			c.JSON(http.StatusRequestTimeout, constants.BuildErrorResponse("Request timeout", "Operation took too long"))
			return
		}

		c.JSON(status, constants.BuildErrorResponse("Delete failed", err.Error()))
		return
	}

	logger.GetLogger().Info("API config deleted successfully",
		zap.Uint("config_id", uint(id)),
		zap.String("client_ip", clientIP),
	)

	// Refresh route registry if refresher is available (full refresh to remove deleted route)
	if h.routeRefresher != nil {
		go func() {
			refreshCtx, refreshCancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer refreshCancel()
			if err := h.routeRefresher.Refresh(refreshCtx); err != nil {
				logger.GetLogger().Warn("Failed to refresh route registry after delete",
					zap.Uint("config_id", uint(id)),
					zap.Error(err),
				)
			} else {
				logger.GetLogger().Info("Route registry refreshed after delete",
					zap.Uint("config_id", uint(id)),
				)
			}
		}()
	}

	c.JSON(status, constants.BuildSuccessResponse("Delete successful"))
}

// DISABLED: Delete API group handler
// func (h *APIConfigHandler) DeleteGroup(c *gin.Context) {
// 	clientIP := c.ClientIP()

// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		logger.GetLogger().Warn("Invalid ID in delete group request",
// 			zap.String("client_ip", clientIP),
// 			zap.String("id_param", c.Param("id")),
// 		)
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid ID"})
// 		return
// 	}

// 	logger.GetLogger().Info("Delete API group request",
// 		zap.Uint("group_id", uint(id)),
// 		zap.String("client_ip", clientIP),
// 	)

// 	// Create context with timeout for the operation
// 	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
// 	defer cancel()

// 	status, err := h.integrasiService.DeleteGroup(ctx, uint(id))
// 	if err != nil {
// 		logger.GetLogger().Error("Failed to delete API group",
// 			zap.Uint("group_id", uint(id)),
// 			zap.String("client_ip", clientIP),
// 			zap.Int("http_status", status),
// 			zap.Error(err),
// 		)

// 		// Check if it's a timeout error
// 		if ctx.Err() == context.DeadlineExceeded {
// 			c.JSON(http.StatusRequestTimeout, gin.H{"message": "Request timeout", "details": "Operation took too long"})
// 			return
// 		}

// 		c.JSON(status, gin.H{"message": "Delete failed", "details": err.Error()})
// 		return
// 	}

// 	logger.GetLogger().Info("API group deleted successfully",
// 		zap.Uint("group_id", uint(id)),
// 		zap.String("client_ip", clientIP),
// 	)

// 	c.JSON(status, gin.H{"message": "Delete successful"})
// }

// DISABLED: Delete API group step handler
// func (h *APIConfigHandler) DeleteGroupStep(c *gin.Context) {
// 	clientIP := c.ClientIP()

// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		logger.GetLogger().Warn("Invalid ID in delete group step request",
// 			zap.String("client_ip", clientIP),
// 			zap.String("id_param", c.Param("id")),
// 		)
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid ID"})
// 		return
// 	}

// 	logger.GetLogger().Info("Delete API group step request",
// 		zap.Uint("step_id", uint(id)),
// 		zap.String("client_ip", clientIP),
// 	)

// 	// Create context with timeout for the operation
// 	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
// 	defer cancel()

// 	status, err := h.integrasiService.DeleteGroupStep(ctx, uint(id))
// 	if err != nil {
// 		logger.GetLogger().Error("Failed to delete API group step",
// 			zap.Uint("step_id", uint(id)),
// 			zap.String("client_ip", clientIP),
// 			zap.Int("http_status", status),
// 			zap.Error(err),
// 		)

// 		// Check if it's a timeout error
// 		if ctx.Err() == context.DeadlineExceeded {
// 			c.JSON(http.StatusRequestTimeout, gin.H{"message": "Request timeout", "details": "Operation took too long"})
// 			return
// 		}

// 		c.JSON(status, gin.H{"message": "Delete failed", "details": err.Error()})
// 		return
// 	}

// 	logger.GetLogger().Info("API group step deleted successfully",
// 		zap.Uint("step_id", uint(id)),
// 		zap.String("client_ip", clientIP),
// 	)

// 	c.JSON(status, gin.H{"message": "Delete successful"})
// }

// DISABLED: Delete API group cron handler
// func (h *APIConfigHandler) DeleteGroupCron(c *gin.Context) {
// 	clientIP := c.ClientIP()

// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		logger.GetLogger().Warn("Invalid ID in delete group cron request",
// 			zap.String("client_ip", clientIP),
// 			zap.String("id_param", c.Param("id")),
// 		)
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid ID"})
// 		return
// 	}

// 	logger.GetLogger().Info("Delete API group cron request",
// 		zap.Uint("cron_id", uint(id)),
// 		zap.String("client_ip", clientIP),
// 	)

// 	// Create context with timeout for the operation
// 	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
// 	defer cancel()

// 	status, err := h.integrasiService.DeleteGroupCron(ctx, uint(id))
// 	if err != nil {
// 		logger.GetLogger().Error("Failed to delete API group cron",
// 			zap.Uint("cron_id", uint(id)),
// 			zap.String("client_ip", clientIP),
// 			zap.Int("http_status", status),
// 			zap.Error(err),
// 		)

// 		// Check if it's a timeout error
// 		if ctx.Err() == context.DeadlineExceeded {
// 			c.JSON(http.StatusRequestTimeout, gin.H{"message": "Request timeout", "details": "Operation took too long"})
// 			return
// 		}

// 		c.JSON(status, gin.H{"message": "Delete failed", "details": err.Error()})
// 		return
// 	}

// 	logger.GetLogger().Info("API group cron deleted successfully",
// 		zap.Uint("cron_id", uint(id)),
// 		zap.String("client_ip", clientIP),
// 	)

// 	c.JSON(status, gin.H{"message": "Delete successful"})
// }

// End Delete

// Start Get By Id
func (h *APIConfigHandler) GetByIDConfig(c *gin.Context) {
	clientIP := c.ClientIP()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.GetLogger().Warn("Invalid ID in get config request",
			zap.String("client_ip", clientIP),
			zap.String("id_param", c.Param("id")),
		)
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid ID", ""))
		return
	}

	logger.GetLogger().Info("Get API config request",
		zap.Uint("config_id", uint(id)),
		zap.String("client_ip", clientIP),
	)

	// Create context with timeout for the operation
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	resp, status, err := h.integrasiService.GetByIDConfig(ctx, uint(id))
	if err != nil {
		logger.GetLogger().Error("Failed to get API config",
			zap.Uint("config_id", uint(id)),
			zap.String("client_ip", clientIP),
			zap.Int("http_status", status),
			zap.Error(err),
		)

		// Check if it's a timeout error
		if ctx.Err() == context.DeadlineExceeded {
			c.JSON(http.StatusRequestTimeout, constants.BuildErrorResponse("Request timeout", "Operation took too long"))
			return
		}

		c.JSON(status, constants.BuildErrorResponse("Get failed", err.Error()))
		return
	}

	logger.GetLogger().Info("API config retrieved successfully",
		zap.Uint("config_id", uint(id)),
		zap.String("client_ip", clientIP),
	)

	c.JSON(status, resp)
}

// DISABLED: Get API group by ID handler
// func (h *APIConfigHandler) GetByIDGroup(c *gin.Context) {
// 	clientIP := c.ClientIP()

// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		logger.GetLogger().Warn("Invalid ID in get group request",
// 			zap.String("client_ip", clientIP),
// 			zap.String("id_param", c.Param("id")),
// 		)
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid ID"})
// 		return
// 	}

// 	logger.GetLogger().Info("Get API group request",
// 		zap.Uint("group_id", uint(id)),
// 		zap.String("client_ip", clientIP),
// 	)

// 	// Create context with timeout for the operation
// 	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
// 	defer cancel()

// 	resp, status, err := h.integrasiService.GetByIDGroup(ctx, uint(id))
// 	if err != nil {
// 		logger.GetLogger().Error("Failed to get API group",
// 			zap.Uint("group_id", uint(id)),
// 			zap.String("client_ip", clientIP),
// 			zap.Int("http_status", status),
// 			zap.Error(err),
// 		)

// 		// Check if it's a timeout error
// 		if ctx.Err() == context.DeadlineExceeded {
// 			c.JSON(http.StatusRequestTimeout, gin.H{"message": "Request timeout", "details": "Operation took too long"})
// 			return
// 		}

// 		c.JSON(status, gin.H{"message": "Get failed", "details": err.Error()})
// 		return
// 	}

// 	logger.GetLogger().Info("API group retrieved successfully",
// 		zap.Uint("group_id", uint(id)),
// 		zap.String("client_ip", clientIP),
// 	)

// 	c.JSON(status, resp)
// }

// DISABLED: Get API group step by ID handler
// func (h *APIConfigHandler) GetByIDGroupStep(c *gin.Context) {
// 	clientIP := c.ClientIP()

// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		logger.GetLogger().Warn("Invalid ID in get group step request",
// 			zap.String("client_ip", clientIP),
// 			zap.String("id_param", c.Param("id")),
// 		)
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid ID"})
// 		return
// 	}

// 	logger.GetLogger().Info("Get API group step request",
// 		zap.Uint("step_id", uint(id)),
// 		zap.String("client_ip", clientIP),
// 	)

// 	// Create context with timeout for the operation
// 	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
// 	defer cancel()

// 	resp, status, err := h.integrasiService.GetByIDGroupStep(ctx, uint(id))
// 	if err != nil {
// 		logger.GetLogger().Error("Failed to get API group step",
// 			zap.Uint("step_id", uint(id)),
// 			zap.String("client_ip", clientIP),
// 			zap.Int("http_status", status),
// 			zap.Error(err),
// 		)

// 		// Check if it's a timeout error
// 		if ctx.Err() == context.DeadlineExceeded {
// 			c.JSON(http.StatusRequestTimeout, gin.H{"message": "Request timeout", "details": "Operation took too long"})
// 			return
// 		}

// 		c.JSON(status, gin.H{"message": "Get failed", "details": err.Error()})
// 		return
// 	}

// 	logger.GetLogger().Info("API group step retrieved successfully",
// 		zap.Uint("step_id", uint(id)),
// 		zap.String("client_ip", clientIP),
// 	)

// 	c.JSON(status, resp)
// }

// DISABLED: Get API group cron by ID handler
// func (h *APIConfigHandler) GetByIDGroupCron(c *gin.Context) {
// 	clientIP := c.ClientIP()

// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		logger.GetLogger().Warn("Invalid ID in get group cron request",
// 			zap.String("client_ip", clientIP),
// 			zap.String("id_param", c.Param("id")),
// 		)
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid ID"})
// 		return
// 	}

// 	logger.GetLogger().Info("Get API group cron request",
// 		zap.Uint("cron_id", uint(id)),
// 		zap.String("client_ip", clientIP),
// 	)

// 	// Create context with timeout for the operation
// 	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
// 	defer cancel()

// 	resp, status, err := h.integrasiService.GetByIDGroupCron(ctx, uint(id))
// 	if err != nil {
// 		logger.GetLogger().Error("Failed to get API group cron",
// 			zap.Uint("cron_id", uint(id)),
// 			zap.String("client_ip", clientIP),
// 			zap.Int("http_status", status),
// 			zap.Error(err),
// 		)

// 		// Check if it's a timeout error
// 		if ctx.Err() == context.DeadlineExceeded {
// 			c.JSON(http.StatusRequestTimeout, gin.H{"message": "Request timeout", "details": "Operation took too long"})
// 			return
// 		}

// 		c.JSON(status, gin.H{"message": "Get failed", "details": err.Error()})
// 		return
// 	}

// 	logger.GetLogger().Info("API group cron retrieved successfully",
// 		zap.Uint("cron_id", uint(id)),
// 		zap.String("client_ip", clientIP),
// 	)

// 	c.JSON(status, resp)
// }

//End Get By Id

// Start Get All
func (h *APIConfigHandler) GetAllConfig(c *gin.Context) {
	// Parse pagination parameters with DTO-based filters
	filter := &dto.APIConfigFilter{}
	pagination := constants.ParsePaginationParamsWithFilter(c, filter)

	// Extract search from All field for logging
	paginatedData, _ := pagination.All.(map[string]any)
	search, _ := paginatedData["search"].(string)

	logger.GetLogger().Info("Handler: Getting all API configs",
		zap.Int("page", pagination.Page),
		zap.Int("limit", pagination.Limit),
		zap.Int("offset", pagination.Offset),
		zap.String("search", search),
		zap.Any("filter", filter),
	)

	res, total, pageTotal, status, err := h.integrasiService.GetAllConfig(pagination.All)
	if err != nil {
		c.JSON(status, constants.BuildErrorResponse("Failed to fetch pages", err.Error()))
	} else {
		c.JSON(http.StatusOK, constants.BuildListResponse(total, pagination.Page, pageTotal, res))
	}
}

// DISABLED: Get all API groups handler
// func (h *APIConfigHandler) GetAllGroup(c *gin.Context) {
// 	pageStr := c.DefaultQuery("page", "1")
// 	limitStr := c.DefaultQuery("limit", "10")
// 	search := c.DefaultQuery("search", "")

// 	page, _ := strconv.Atoi(pageStr)
// 	limit, _ := strconv.Atoi(limitStr)
// 	if page < 1 {
// 		page = 1
// 	}
// 	if limit < 1 {
// 		limit = 10
// 	}
// 	offset := (page - 1) * limit

// 	res, total, pageTotal, status, err := h.integrasiService.GetAllGroup(limit, offset, search)
// 	if err != nil {
// 		c.JSON(status, gin.H{"message": "Failed to fetch pages", "details": err.Error()})
// 	} else {
// 		c.JSON(http.StatusOK, gin.H{
// 			"total":      total,
// 			"page":       page,
// 			"page_total": pageTotal,
// 			"data":       res,
// 		})
// 	}
// }

// DISABLED: Get all API group steps handler
// func (h *APIConfigHandler) GetAllGroupStep(c *gin.Context) {
// 	pageStr := c.DefaultQuery("page", "1")
// 	limitStr := c.DefaultQuery("limit", "10")
// 	search := c.DefaultQuery("search", "")
// 	groupID := c.DefaultQuery("group_id", "0")

// 	page, _ := strconv.Atoi(pageStr)
// 	limit, _ := strconv.Atoi(limitStr)
// 	gID, _ := strconv.ParseUint(groupID, 10, 32)
// 	if page < 1 {
// 		page = 1
// 	}
// 	if limit < 1 {
// 		limit = 10
// 	}
// 	offset := (page - 1) * limit

// 	res, total, pageTotal, status, err := h.integrasiService.GetAllGroupStep(limit, offset, search, uint(gID))
// 	if err != nil {
// 		c.JSON(status, gin.H{"message": "Failed to fetch pages", "details": err.Error()})
// 	} else {
// 		c.JSON(http.StatusOK, gin.H{
// 			"total":      total,
// 			"page":       page,
// 			"page_total": pageTotal,
// 			"data":       res,
// 		})
// 	}
// }
// DISABLED: Get all API group crons handler
// func (h *APIConfigHandler) GetAllGroupCron(c *gin.Context) {
// 	pageStr := c.DefaultQuery("page", "1")
// 	limitStr := c.DefaultQuery("limit", "10")
// 	search := c.DefaultQuery("search", "")
// 	slug := c.DefaultQuery("slug", "")

// 	page, _ := strconv.Atoi(pageStr)
// 	limit, _ := strconv.Atoi(limitStr)
// 	if page < 1 {
// 		page = 1
// 	}
// 	if limit < 1 {
// 		limit = 10
// 	}
// 	offset := (page - 1) * limit

// 	res, total, pageTotal, status, err := h.integrasiService.GetAllGroupCron(limit, offset, search, slug)
// 	if err != nil {
// 		c.JSON(status, gin.H{"message": "Failed to fetch pages", "details": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{
// 		"total":      total,
// 		"page":       page,
// 		"page_total": pageTotal,
// 		"data":       res,
// 	})
// }

//End Get All

// DISABLED: External integration handler
// func (h *APIConfigHandler) ExternalIntegrasi(c *gin.Context) {
// 	slug := c.Param("slug")
// 	clientIP := c.ClientIP()
// 	userAgent := c.GetHeader("User-Agent")
// 	requestMethod := c.Request.Method

// 	logger.GetLogger().Info("External integration request",
// 		zap.String("method", requestMethod),
// 		zap.String("path", c.Request.URL.Path),
// 		zap.String("slug", slug),
// 		zap.String("client_ip", clientIP),
// 		zap.String("user_agent", userAgent),
// 		zap.String("query", c.Request.URL.RawQuery),
// 	)

// 	if slug == "" {
// 		logger.GetLogger().Warn("Empty slug in external integration request",
// 			zap.String("client_ip", clientIP),
// 		)
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid Slug"})
// 		return
// 	}

// 	// Create context with timeout for the operation
// 	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
// 	defer cancel()

// 	resp, status, err := h.integrasiService.GetBySlugConfig(ctx, slug)
// 	if err != nil {
// 		// Check if it's a timeout error
// 		if ctx.Err() == context.DeadlineExceeded {
// 			logger.GetLogger().Error("External integration config lookup timeout",
// 				zap.String("slug", slug),
// 				zap.String("client_ip", clientIP),
// 				zap.Error(ctx.Err()),
// 			)
// 			c.JSON(http.StatusRequestTimeout, gin.H{"message": "Request timeout", "details": "Config lookup took too long"})
// 			return
// 		}

// 		logger.GetLogger().Error("Failed to get config for external integration",
// 			zap.String("slug", slug),
// 			zap.String("client_ip", clientIP),
// 			zap.Int("http_status", status),
// 			zap.Error(err),
// 		)
// 		c.JSON(status, gin.H{"message": "Get failed", "details": err.Error()})
// 		return
// 	}

// 	// For HTTP protocol, validate HTTP method match
// 	// For gRPC protocol, allow any HTTP method (gRPC uses POST internally)
// 	if resp.Protocol == "http" {
// 		allowedMethod := strings.ToUpper(resp.Method)

// 		logger.GetLogger().Info("External integration config loaded",
// 			zap.String("slug", slug),
// 			zap.String("allowed_method", allowedMethod),
// 			zap.String("request_method", requestMethod),
// 			zap.String("target_url", resp.URL),
// 			zap.String("url_config_nama", resp.URLConfig.Nama),
// 			zap.String("uri", resp.URI),
// 			zap.String("client_ip", clientIP),
// 		)

// 		if allowedMethod != requestMethod {
// 			logger.GetLogger().Warn("Method not allowed for external integration",
// 				zap.String("slug", slug),
// 				zap.String("allowed_method", allowedMethod),
// 				zap.String("request_method", requestMethod),
// 				zap.String("client_ip", clientIP),
// 			)
// 			c.JSON(http.StatusMethodNotAllowed, gin.H{"message": "Method Not Allowed"})
// 			return
// 		}
// 	} else {
// 		logger.GetLogger().Info("External integration config loaded",
// 			zap.String("slug", slug),
// 			zap.String("protocol", resp.Protocol),
// 			zap.String("grpc_method", resp.Method),
// 			zap.String("request_method", requestMethod),
// 			zap.String("target_address", resp.URL),
// 			zap.String("url_config_nama", resp.URLConfig.Nama),
// 			zap.String("client_ip", clientIP),
// 		)
// 	}

// 	var body []byte

// 	if resp.Protocol == "grpc" {
// 		// Handle gRPC request
// 		logger.GetLogger().Info("Executing external gRPC request",
// 			zap.String("slug", slug),
// 			zap.String("service", resp.URLConfig.GRPCService),
// 			zap.String("method", resp.Method),
// 			zap.String("address", resp.URL),
// 			zap.Bool("tls_enabled", resp.URLConfig.TLSEnabled),
// 			zap.String("client_ip", clientIP),
// 			)

// 		// Use timeout from config or default
// 		timeout := time.Duration(resp.Timeout) * time.Second
// 		if timeout <= 0 {
// 			timeout = 30 * time.Second
// 		}
// 		requestCtx, cancel := context.WithTimeout(ctx, timeout)
// 		defer cancel()

// 		body, status, err = integrasi.DoRequestWithProtocol(requestCtx, resp, c)
// 	} else {
// 		// Handle HTTP request
// 		config := integrasi.ConvertToAPIResponseConfig(resp).BuildAPIRequestConfig(c)

// 		logger.GetLogger().Info("Executing external integration request",
// 			zap.String("slug", slug),
// 			zap.String("method", config.Method),
// 			zap.String("url", config.URL),
// 			zap.Int("timeout", config.Timeout),
// 			zap.Int("max_retries", config.MaxRetries),
// 			zap.String("client_ip", clientIP),
// 		)

// 		// Use timeout from config or default
// 		timeout := time.Duration(config.Timeout) * time.Second
// 		if timeout <= 0 {
// 			timeout = 30 * time.Second
// 		}
// 		requestCtx, cancel := context.WithTimeout(ctx, timeout)
// 		defer cancel()

// 		body, status, err = integrasi.DoRequestSafeWithRetry(requestCtx, config)
// 	}
// 	if err != nil {
// 		if resp.Protocol == "grpc" {
// 			logger.GetLogger().Error("External gRPC integration request failed",
// 				zap.String("slug", slug),
// 				zap.String("service", resp.URLConfig.GRPCService),
// 				zap.String("method", resp.Method),
// 				zap.String("address", resp.URL),
// 				zap.String("client_ip", clientIP),
// 				zap.Int("http_status", status),
// 				zap.Error(err),
// 			)
// 		} else {
// 			logger.GetLogger().Error("External HTTP integration request failed",
// 				zap.String("slug", slug),
// 				zap.String("method", resp.Method),
// 				zap.String("url", resp.URL),
// 				zap.String("client_ip", clientIP),
// 				zap.Int("http_status", status),
// 				zap.Error(err),
// 			)
// 		}
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	if resp.Protocol == "grpc" {
// 		logger.GetLogger().Info("External gRPC integration request completed",
// 			zap.String("slug", slug),
// 			zap.String("service", resp.URLConfig.GRPCService),
// 			zap.String("method", resp.Method),
// 			zap.String("address", resp.URL),
// 			zap.Int("response_status", status),
// 			zap.Int("response_size", len(body)),
// 			zap.String("client_ip", clientIP),
// 		)
// 	} else {
// 		logger.GetLogger().Info("External HTTP integration request completed",
// 			zap.String("slug", slug),
// 			zap.String("method", resp.Method),
// 			zap.String("url", resp.URL),
// 			zap.Int("response_status", status),
// 			zap.Int("response_size", len(body)),
// 			zap.String("client_ip", clientIP),
// 		)
// 	}

// 	switch status {
// 	case http.StatusNoContent:
// 		logger.GetLogger().Debug("External integration returned no content",
// 			zap.String("slug", slug),
// 			zap.String("client_ip", clientIP),
// 		)
// 		c.Status(status)

// 	case http.StatusOK, http.StatusCreated:
// 		var jsonData interface{}
// 		if err := json.Unmarshal(body, &jsonData); err != nil {
// 			logger.GetLogger().Error("Invalid JSON in external integration response",
// 				zap.String("slug", slug),
// 				zap.String("client_ip", clientIP),
// 				zap.Error(err),
// 			)
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid JSON"})
// 			return
// 		}

// 		if strings.TrimSpace(resp.Manipulation) != "" {
// 			logger.GetLogger().Debug("Applying template manipulation",
// 				zap.String("slug", slug),
// 				zap.String("manipulation", resp.Manipulation),
// 				zap.String("client_ip", clientIP),
// 			)

// 			rendered, err := integrasi.RenderTemplateWithSprig(resp.Manipulation, jsonData)
// 			if err != nil {
// 				logger.GetLogger().Error("Failed to render template for external integration",
// 					zap.String("slug", slug),
// 					zap.String("client_ip", clientIP),
// 					zap.Error(err),
// 				)
// 				c.JSON(http.StatusInternalServerError, gin.H{
// 					"error":   "failed to render template",
// 					"details": err.Error(),
// 				})
// 				return
// 			}

// 			var result interface{}
// 			if err := json.Unmarshal([]byte(rendered), &result); err != nil {
// 				logger.GetLogger().Error("Rendered output is not valid JSON",
// 					zap.String("slug", slug),
// 					zap.String("client_ip", clientIP),
// 					zap.String("rendered_output", rendered),
// 					zap.Error(err),
// 				)
// 				c.JSON(http.StatusInternalServerError, gin.H{
// 					"error":    "rendered output is not valid JSON",
// 					"rendered": rendered,
// 				})
// 				return
// 			}

// 			logger.GetLogger().Info("Template manipulation applied successfully",
// 				zap.String("slug", slug),
// 				zap.String("client_ip", clientIP),
// 			)

// 			c.JSON(status, result)
// 			return
// 		}

// 		logger.GetLogger().Debug("Returning external integration JSON response",
// 			zap.String("slug", slug),
// 			zap.String("client_ip", clientIP),
// 		)

// 		c.JSON(status, jsonData)

// 	default:
// 		logger.GetLogger().Debug("Returning external integration raw response",
// 			zap.String("slug", slug),
// 			zap.Int("status", status),
// 			zap.String("client_ip", clientIP),
// 		)
// 		c.Data(status, "application/json", body)
// 	}
// }

// DISABLED: Execute integration by slug handler
// func (h *APIConfigHandler) ExecuteBySlug(c *gin.Context) {
// 	slug := c.Param("slug")
// 	if slug == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid Slug"})
// 		return
// 	}

// 	var requestBody map[string]interface{}
// 	if err := c.ShouldBindJSON(&requestBody); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body", "details": err.Error()})
// 		return
// 	}

// 	result, status, err := h.integrasiService.ExecuteBySlug(slug, requestBody)
// 	if err != nil {
// 		c.JSON(status, gin.H{"message": "Execution failed", "details": err.Error()})
// 		return
// 	}

// 	c.JSON(status, result)
// }

// URLConfig Handler Functions
func (h *APIConfigHandler) CreateURLConfig(c *gin.Context) {
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	logger.GetLogger().Info("Create URL config request",
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("client_ip", clientIP),
		zap.String("user_agent", userAgent),
	)

	var req dto.URLConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.GetLogger().Warn("Invalid JSON in create URL config request",
			zap.String("client_ip", clientIP),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid request", err.Error()))
		return
	}

	logger.GetLogger().Info("Creating URL config",
		zap.String("nama", req.Nama),
		zap.String("protocol", req.Protocol),
		zap.String("url", req.URL),
		zap.Bool("is_active", req.IsActive),
		zap.String("client_ip", clientIP),
	)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	status, err := h.integrasiService.CreateURLConfig(ctx, req)
	if err != nil {
		logger.GetLogger().Error("Failed to create URL config",
			zap.String("nama", req.Nama),
			zap.String("client_ip", clientIP),
			zap.Int("http_status", status),
			zap.Error(err),
		)

		if ctx.Err() == context.DeadlineExceeded {
			c.JSON(http.StatusRequestTimeout, constants.BuildErrorResponse("Request timeout", "Operation took too long"))
			return
		}

		c.JSON(status, constants.BuildErrorResponse("Create failed", err.Error()))
		return
	}

	logger.GetLogger().Info("URL config created successfully",
		zap.String("nama", req.Nama),
		zap.String("client_ip", clientIP),
	)

	c.JSON(status, constants.BuildSuccessResponse("Create successful"))
}

func (h *APIConfigHandler) GetByIDURLConfig(c *gin.Context) {
	clientIP := c.ClientIP()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.GetLogger().Warn("Invalid ID in get URL config request",
			zap.String("client_ip", clientIP),
			zap.String("id_param", c.Param("id")),
		)
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid ID", ""))
		return
	}

	logger.GetLogger().Info("Get URL config request",
		zap.Uint("url_config_id", uint(id)),
		zap.String("client_ip", clientIP),
	)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	resp, status, err := h.integrasiService.GetByIDURLConfig(ctx, uint(id))
	if err != nil {
		logger.GetLogger().Error("Failed to get URL config",
			zap.Uint("url_config_id", uint(id)),
			zap.String("client_ip", clientIP),
			zap.Int("http_status", status),
			zap.Error(err),
		)

		if ctx.Err() == context.DeadlineExceeded {
			c.JSON(http.StatusRequestTimeout, constants.BuildErrorResponse("Request timeout", "Operation took too long"))
			return
		}

		c.JSON(status, constants.BuildErrorResponse("Get failed", err.Error()))
		return
	}

	logger.GetLogger().Info("URL config retrieved successfully",
		zap.Uint("url_config_id", uint(id)),
		zap.String("client_ip", clientIP),
	)

	c.JSON(status, resp)
}

func (h *APIConfigHandler) GetAllURLConfig(c *gin.Context) {
	// Parse pagination parameters with DTO-based filters
	filter := &dto.URLConfigFilter{}
	pagination := constants.ParsePaginationParamsWithFilter(c, filter)

	// Extract search from All field for logging
	paginatedData, _ := pagination.All.(map[string]any)
	search, _ := paginatedData["search"].(string)

	logger.GetLogger().Info("Handler: Getting all URL configs",
		zap.Int("page", pagination.Page),
		zap.Int("limit", pagination.Limit),
		zap.Int("offset", pagination.Offset),
		zap.String("search", search),
		zap.Any("filter", filter),
	)

	res, total, pageTotal, status, err := h.integrasiService.GetAllURLConfig(pagination.All)
	if err != nil {
		c.JSON(status, constants.BuildErrorResponse("Failed to fetch URL configs", err.Error()))
	} else {
		c.JSON(http.StatusOK, constants.BuildListResponse(total, pagination.Page, pageTotal, res))
	}
}

func (h *APIConfigHandler) UpdateURLConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid ID", ""))
		return
	}

	var req dto.URLConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid request", err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	status, err := h.integrasiService.UpdateURLConfig(ctx, uint(id), req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			c.JSON(http.StatusRequestTimeout, constants.BuildErrorResponse("Request timeout", "Operation took too long"))
			return
		}
		c.JSON(status, constants.BuildErrorResponse("Update failed", err.Error()))
		return
	}
	// Refresh entire registry because URLConfig change might affect multiple routes (e.g. IsActive status)
	if h.routeRefresher != nil {
		go func() {
			refreshCtx, refreshCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer refreshCancel()
			if err := h.routeRefresher.Refresh(refreshCtx); err != nil {
				logger.GetLogger().Warn("Failed to refresh registry after URL config update",
					zap.Uint("url_config_id", uint(id)),
					zap.Error(err),
				)
			} else {
				logger.GetLogger().Info("Registry refreshed after URL config update",
					zap.Uint("url_config_id", uint(id)),
				)
			}
		}()
	}

	c.JSON(status, constants.BuildSuccessResponse("Update successful"))
}

func (h *APIConfigHandler) DeleteURLConfig(c *gin.Context) {
	clientIP := c.ClientIP()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.GetLogger().Warn("Invalid ID in delete URL config request",
			zap.String("client_ip", clientIP),
			zap.String("id_param", c.Param("id")),
		)
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid ID", ""))
		return
	}

	logger.GetLogger().Info("Delete URL config request",
		zap.Uint("url_config_id", uint(id)),
		zap.String("client_ip", clientIP),
	)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	status, err := h.integrasiService.DeleteURLConfig(ctx, uint(id))
	if err != nil {
		logger.GetLogger().Error("Failed to delete URL config",
			zap.Uint("url_config_id", uint(id)),
			zap.String("client_ip", clientIP),
			zap.Int("http_status", status),
			zap.Error(err),
		)

		if ctx.Err() == context.DeadlineExceeded {
			c.JSON(http.StatusRequestTimeout, constants.BuildErrorResponse("Request timeout", "Operation took too long"))
			return
		}

		c.JSON(status, constants.BuildErrorResponse("Delete failed", err.Error()))
		return
	}

	logger.GetLogger().Info("URL config deleted successfully",
		zap.Uint("url_config_id", uint(id)),
		zap.String("client_ip", clientIP),
	)

	c.JSON(status, constants.BuildSuccessResponse("Delete successful"))
}
