package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/surdiana/gateway/internal/dto"
	"github.com/surdiana/gateway/internal/service"
	"github.com/surdiana/gateway/pkg/integrasi"
	"github.com/surdiana/gateway/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DynamicURIMiddleware handles custom URI routing for API configurations
type DynamicURIMiddleware struct {
	apiConfigService *service.APIConfigService
	cacheService     *service.CacheService
}

// NewDynamicURIMiddleware creates a new dynamic URI middleware
func NewDynamicURIMiddleware(apiConfigService *service.APIConfigService, cacheService *service.CacheService) *DynamicURIMiddleware {
	return &DynamicURIMiddleware{
		apiConfigService: apiConfigService,
		cacheService:     cacheService,
	}
}

// HandleDynamicURI processes requests to custom URIs and routes them to appropriate API configs
func (m *DynamicURIMiddleware) HandleDynamicURI() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		requestPath := c.Request.URL.Path
		requestMethod := c.Request.Method

		logger.GetLogger().Info("Dynamic URI middleware processing request",
			zap.String("method", requestMethod),
			zap.String("path", requestPath),
			zap.String("client_ip", clientIP),
		)

		// Skip if this is an API management route - let them go through normal middleware chain
		if strings.HasPrefix(requestPath, "/api/") || strings.HasPrefix(requestPath, "/health") {
			c.Next()
			return
		}

		// Try to find API config by URI (with pattern matching support)
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		config, uriParams, err := m.apiConfigService.GetByURIConfigWithPattern(ctx, requestPath, requestMethod)
		if err != nil {
			// No matching API config found, continue with normal routing
			logger.GetLogger().Debug("No API config found for URI, continuing normal routing",
				zap.String("path", requestPath),
				zap.String("method", requestMethod),
				zap.String("client_ip", clientIP),
			)
			c.Next()
			return
		}

		// API config found, set it in context and process
		logger.GetLogger().Info("API config found for custom URI",
			zap.String("slug", config.Slug),
			zap.String("uri", config.URI),
			zap.String("method", config.Method),
			zap.String("protocol", config.Protocol),
			zap.String("request_path", requestPath),
			zap.String("request_method", requestMethod),
			zap.String("client_ip", clientIP),
		)

		// Capture request body for cache key generation
		if c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Set("request_body", string(bodyBytes))
		}

		// Store the config and URI parameters in context for the handler to use
		c.Set("api_config", config)
		c.Set("is_dynamic_uri", true)
		c.Set("uri_params", uriParams)

		// Log URI parameters if any
		if len(uriParams) > 0 {
			logger.GetLogger().Info("URI parameters extracted",
				zap.String("slug", config.Slug),
				zap.Any("uri_params", uriParams),
			)
		}

		// Call the external integration handler
		m.processDynamicURI(c, config, uriParams)
		c.Abort() // Stop further middleware processing
	}
}

// processDynamicURI handles the actual request processing for dynamic URIs
func (m *DynamicURIMiddleware) processDynamicURI(c *gin.Context, config *dto.APIConfigResponse, uriParams map[string]string) {
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	requestMethod := c.Request.Method

	logger.GetLogger().Info("Processing dynamic URI request",
		zap.String("slug", config.Slug),
		zap.String("method", requestMethod),
		zap.String("path", c.Request.URL.Path),
		zap.String("client_ip", clientIP),
		zap.String("user_agent", userAgent),
		zap.Int("uri_params_count", len(uriParams)),
	)

	// Validate HTTP method for HTTP protocol
	if config.Protocol == "http" {
		allowedMethod := strings.ToUpper(config.Method)
		if allowedMethod != requestMethod {
			logger.GetLogger().Warn("Method not allowed for dynamic URI",
				zap.String("slug", config.Slug),
				zap.String("allowed_method", allowedMethod),
				zap.String("request_method", requestMethod),
				zap.String("client_ip", clientIP),
			)
			c.JSON(http.StatusMethodNotAllowed, gin.H{"message": "Method Not Allowed"})
			return
		}
	}

	// Generate cache key
	cacheKey := m.cacheService.GenerateCacheKey(config, c, uriParams)

	// Check cache first
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	cachedData, cachedStatus, cachedHeaders, found := m.cacheService.GetCachedResponse(ctx, cacheKey)
	if found {
		logger.GetLogger().Info("Cache hit for dynamic URI request",
			zap.String("slug", config.Slug),
			zap.String("cache_key", cacheKey),
			zap.Int("cached_status", cachedStatus),
			zap.Int("cached_data_size", len(cachedData)),
			zap.String("client_ip", clientIP),
		)

		// Set cached headers if any
		for key, value := range cachedHeaders {
			c.Header(key, value)
		}

		// Return cached response
		m.returnResponse(c, cachedData, cachedStatus, config)
		return
	}

	// Cache miss - proceed with integration request
	logger.GetLogger().Info("Cache miss for dynamic URI request",
		zap.String("slug", config.Slug),
		zap.String("cache_key", cacheKey),
		zap.String("client_ip", clientIP),
	)

	// Create context with timeout for the operation
	ctx, cancel = context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Execute the integration request using existing handler logic
	body, status, err := m.executeExternalIntegration(ctx, config, c, uriParams)
	if err != nil {
		logger.GetLogger().Error("Dynamic URI integration request failed",
			zap.String("slug", config.Slug),
			zap.String("protocol", config.Protocol),
			zap.String("method", config.Method),
			zap.String("client_ip", clientIP),
			zap.Int("http_status", status),
			zap.Error(err),
		)

		// Cache error responses briefly (except 5xx errors)
		if status >= 400 && status < 500 && m.cacheService.ShouldCache(c, config, status, len(body)) {
			m.cacheService.SetCachedResponse(ctx, cacheKey, body, status, nil, config)
		}

		if status == http.StatusRequestTimeout {
			c.JSON(http.StatusRequestTimeout, gin.H{"message": "Request timeout", "details": "Operation took too long"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.GetLogger().Info("Dynamic URI integration request completed",
		zap.String("slug", config.Slug),
		zap.String("protocol", config.Protocol),
		zap.String("method", config.Method),
		zap.Int("response_status", status),
		zap.Int("response_size", len(body)),
		zap.String("client_ip", clientIP),
	)

	// Cache the successful response if caching is enabled and appropriate
	if m.cacheService.ShouldCache(c, config, status, len(body)) {
		// Extract response headers for caching
		headers := make(map[string]string)
		importantHeaders := []string{"content-type", "cache-control", "etag", "last-modified"}
		for _, header := range importantHeaders {
			if value := c.Writer.Header().Get(header); value != "" {
				headers[header] = value
			}
		}

		if err := m.cacheService.SetCachedResponse(ctx, cacheKey, body, status, headers, config); err != nil {
			logger.GetLogger().Error("Failed to cache response",
				zap.String("slug", config.Slug),
				zap.String("cache_key", cacheKey),
				zap.Error(err),
			)
		}
	}

	// Return the response
	m.returnResponse(c, body, status, config)
}

// returnResponse handles response formatting and template manipulation
func (m *DynamicURIMiddleware) returnResponse(c *gin.Context, body []byte, status int, config *dto.APIConfigResponse) {
	switch status {
	case http.StatusNoContent:
		c.Status(status)

	case http.StatusOK, http.StatusCreated:
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err != nil {
			logger.GetLogger().Error("Invalid JSON in dynamic URI response",
				zap.String("slug", config.Slug),
				zap.String("client_ip", c.ClientIP()),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid JSON"})
			return
		}

		// Apply template manipulation if configured
		if strings.TrimSpace(config.Manipulation) != "" {
			rendered, err := integrasi.RenderTemplateWithSprig(config.Manipulation, jsonData)
			if err != nil {
				logger.GetLogger().Error("Failed to render template for dynamic URI",
					zap.String("slug", config.Slug),
					zap.String("client_ip", c.ClientIP()),
					zap.Error(err),
				)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "failed to render template",
					"details": err.Error(),
				})
				return
			}

			var finalResult interface{}
			if err := json.Unmarshal([]byte(rendered), &finalResult); err != nil {
				logger.GetLogger().Error("Rendered output is not valid JSON",
					zap.String("slug", config.Slug),
					zap.String("client_ip", c.ClientIP()),
					zap.String("rendered_output", rendered),
					zap.Error(err),
				)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":    "rendered output is not valid JSON",
					"rendered": rendered,
				})
				return
			}

			c.JSON(status, finalResult)
			return
		}

		c.JSON(status, jsonData)

	default:
		c.Data(status, "application/json", body)
	}
}

// executeExternalIntegration handles the actual integration execution
func (m *DynamicURIMiddleware) executeExternalIntegration(ctx context.Context, config *dto.APIConfigResponse, c *gin.Context, uriParams map[string]string) ([]byte, int, error) {
	if config.Protocol == "grpc" {
		// Handle gRPC request
		logger.GetLogger().Info("Executing external gRPC request",
			zap.String("slug", config.Slug),
			zap.String("service", config.URLConfig.GRPCService),
			zap.String("method", config.Method),
			zap.String("address", config.URL),
			zap.Bool("tls_enabled", config.URLConfig.TLSEnabled),
			zap.Int("uri_params_count", len(uriParams)),
		)

		return integrasi.DoRequestWithProtocol(ctx, config, c)
	} else {
		// Handle HTTP request
		apiRequestConfig := integrasi.ConvertToAPIResponseConfig(config).BuildAPIRequestConfig(c)

		// Add URI parameters to the request context for template rendering
		if len(uriParams) > 0 {
			// Create a modified context that includes URI parameters
			// This can be used for template rendering in headers, body, etc.
			c.Set("uri_params", uriParams)

			logger.GetLogger().Info("URI parameters added to request context",
				zap.String("slug", config.Slug),
				zap.Any("uri_params", uriParams),
			)
		}

		logger.GetLogger().Info("Executing external HTTP request",
			zap.String("slug", config.Slug),
			zap.String("method", apiRequestConfig.Method),
			zap.String("url", apiRequestConfig.URL),
			zap.Int("timeout", apiRequestConfig.Timeout),
			zap.Int("max_retries", apiRequestConfig.MaxRetries),
			zap.Int("uri_params_count", len(uriParams)),
		)

		return integrasi.DoRequestSafeWithRetry(ctx, apiRequestConfig)
	}
}