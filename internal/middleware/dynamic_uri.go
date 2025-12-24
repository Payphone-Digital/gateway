package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Payphone-Digital/gateway/internal/constants"
	"github.com/Payphone-Digital/gateway/internal/dto"
	"github.com/Payphone-Digital/gateway/internal/service"
	"github.com/Payphone-Digital/gateway/pkg/integrasi"
	"github.com/Payphone-Digital/gateway/pkg/logger"
	"github.com/Payphone-Digital/gateway/pkg/routing"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DynamicURIMiddleware handles custom URI routing for API configurations
type DynamicURIMiddleware struct {
	registry      *routing.RouteRegistry
	cacheService  *service.CacheService
	jwtMiddleware *JWTMiddleware
}

// NewDynamicURIMiddleware creates a new dynamic URI middleware
func NewDynamicURIMiddleware(registry *routing.RouteRegistry, cacheService *service.CacheService, jwtMiddleware *JWTMiddleware) *DynamicURIMiddleware {
	return &DynamicURIMiddleware{
		registry:      registry,
		cacheService:  cacheService,
		jwtMiddleware: jwtMiddleware,
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

		// Skip if this is an API management route (system routes)
		// Only skip /api/v1/ (management APIs) and /api/health (health checks)
		// Access to custom /api/ routes (e.g., /api/cek/object) should be allowed
		if strings.HasPrefix(requestPath, "/api/v1/") || strings.HasPrefix(requestPath, "/api/health") || strings.HasPrefix(requestPath, "/health") {
			c.Next()
			return
		}

		// Try to find API config in route registry (O(k) lookup)
		config, uriParams, err := m.registry.Match(requestPath, requestMethod)
		
		// Fallback: If not found and path starts with /api/, try stripping /api
		// This allows users to access /api/cek/object even if config is /cek/object
		if err != nil && strings.HasPrefix(requestPath, "/api/") {
			strippedPath := strings.TrimPrefix(requestPath, "/api")
			// Ensure it has leading slash if it became empty or didn't have one (though /api/x implies /x)
			if !strings.HasPrefix(strippedPath, "/") {
				strippedPath = "/" + strippedPath
			}
			
			configRetry, uriParamsRetry, errRetry := m.registry.Match(strippedPath, requestMethod)
			if errRetry == nil {
				config = configRetry
				uriParams = uriParamsRetry
				err = nil
				logger.GetLogger().Info("Dynamic URI matched using fallback (stripped /api prefix)",
					zap.String("original_path", requestPath),
					zap.String("matched_path", strippedPath),
				)
			}
		}

		if err != nil {
			// No matching API config found, continue with normal routing
			if err == routing.ErrRouteNotFound {
				logger.GetLogger().Debug("No API config found for URI, continuing normal routing",
					zap.String("path", requestPath),
					zap.String("method", requestMethod),
					zap.String("client_ip", clientIP),
				)
			} else if err == routing.ErrMethodNotAllowed {
				logger.GetLogger().Warn("Method not allowed for dynamic URI",
					zap.String("path", requestPath),
					zap.String("method", requestMethod),
					zap.String("client_ip", clientIP),
				)
				c.JSON(http.StatusMethodNotAllowed, gin.H{"message": "Method Not Allowed"})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// API config found, set it in context and process
		logger.GetLogger().Info("API config found for custom URI",
			zap.String("slug", config.Path),
			zap.String("uri", config.URI),
			zap.String("method", config.Method),
			zap.String("protocol", config.Protocol),
			zap.String("request_path", requestPath),
			zap.String("request_method", requestMethod),
			zap.String("client_ip", clientIP),
			zap.Bool("is_admin", config.IsAdmin),
            zap.Bool("is_active", config.URLConfig.IsActive),
		)

		// Check if URL Config is active
        if !config.URLConfig.IsActive {
            logger.GetLogger().Warn("URL Config is inactive",
                zap.String("slug", config.Path),
                zap.String("url_config", config.URLConfig.Nama),
            )
            c.JSON(http.StatusServiceUnavailable, constants.BuildErrorResponse("Service Unavailable", "Service is currently inactive"))
            c.Abort()
            return
        }

		// Store URI parameters in context EARLY so ExtractUserVars can find them
		// This is critical for validating path parameters
		c.Set("uri_params", uriParams)

        // VALIDATION
		// Convert dto.Variable map to map[string]interface{} for validator
		validationVars := make(map[string]interface{})
		for k, v := range config.Variables {
			validationVars[k] = v
		}

		validator := NewDynamicValidator(validationVars)
		
		// Perform comprehensive validation on ALL input sources (Body, Path, Query, Headers)
		// 1. Extract all user variables
		userVars := integrasi.ExtractUserVars(c)
		
		// 2. Flatten into a single map for validation
		// Precedence (Last wins): Header < Query < Path < Body
		// This ensures that explicit path params (usually most critical) or body payload take precedence
		validationData := make(map[string]interface{})
		
		for k, v := range userVars.HeaderParams {
			validationData[k] = v
		}
		for k, v := range userVars.QueryParams {
			validationData[k] = v
		}
		for k, v := range userVars.PathParams {
			validationData[k] = v
		}
		for k, v := range userVars.BodyParams {
			validationData[k] = v
		}

		// 3. Validate
		errors := validator.ValidateData(validationData)
		if len(errors) > 0 {
			logger.GetLogger().Warn("Request validation failed",
				zap.String("path", c.Request.URL.Path),
				zap.Any("validation_errors", errors),
			)

			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": "Unprocessable Entity",
				"errors":  errors,
			})
			c.Abort()
			return
		}

		// AUTHENTICATION & AUTHORIZATION
		// 1. Gateway Admin (Priority)
		if config.IsAdmin {
			logger.GetLogger().Info("Enforcing Gateway Admin Authentication",
				zap.String("slug", config.Path),
				zap.String("client_ip", clientIP),
			)
			m.jwtMiddleware.RequireAuth()(c)
			if c.IsAborted() {
				logger.GetLogger().Warn("Gateway Admin Authentication failed",
					zap.String("slug", config.Path),
					zap.String("client_ip", clientIP),
				)
				return
			}
		}

		// 2. Dynamic Authentication (If Configured)
		if !config.IsAdmin && config.AuthRequired {
			if !m.authenticate(c, config) {
				logger.GetLogger().Warn("Dynamic Authentication failed",
					zap.String("slug", config.Path),
					zap.String("auth_type", config.AuthType),
					zap.String("client_ip", clientIP),
				)
				c.Abort() // ensure aborted if not already
				return
			}
		}

		// Capture request body for cache key generation
		if c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Set("request_body", string(bodyBytes))
		}

		// Store the config and URI parameters in context for the handler to use
		c.Set("api_config", config)
		c.Set("is_dynamic_uri", true)
		// c.Set("uri_params", uriParams) // Already set above

		// Log URI parameters if any
		if len(uriParams) > 0 {
			logger.GetLogger().Info("URI parameters extracted",
				zap.String("slug", config.Path),
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
		zap.String("slug", config.Path),
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
				zap.String("slug", config.Path),
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
			zap.String("slug", config.Path),
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
		zap.String("slug", config.Path),
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
			zap.String("slug", config.Path),
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
		zap.String("slug", config.Path),
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
				zap.String("slug", config.Path),
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
				zap.String("slug", config.Path),
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
					zap.String("slug", config.Path),
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
					zap.String("slug", config.Path),
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
			zap.String("slug", config.Path),
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
				zap.String("slug", config.Path),
				zap.Any("uri_params", uriParams),
			)
		}

		logger.GetLogger().Info("Executing external HTTP request",
			zap.String("slug", config.Path),
			zap.String("method", apiRequestConfig.Method),
			zap.String("url", apiRequestConfig.URL),
			zap.Int("timeout", apiRequestConfig.Timeout),
			zap.Int("max_retries", apiRequestConfig.MaxRetries),
			zap.Int("uri_params_count", len(uriParams)),
		)

		return integrasi.DoRequestSafeWithRetry(ctx, apiRequestConfig)
	}
}

// authenticate handles dynamic authentication based on config
func (m *DynamicURIMiddleware) authenticate(c *gin.Context, config *dto.APIConfigResponse) bool {
	authTypes := strings.Split(config.AuthType, ",")
	
	// Track checks performed to prevent "Default Allow" if config is empty/invalid but AuthRequired was true
	checksPerformed := 0

	for _, authType := range authTypes {
		authType = strings.TrimSpace(strings.ToLower(authType))
		if authType == "" || authType == "none" {
			continue
		}

		checksPerformed++
		success := false
		switch authType {
		case "jwt":
			if m.checkCustomJWT(c, config) {
				success = true
			}
		case "basic":
			if m.checkBasicAuth(c, config) {
				success = true
			}
		case "apikey":
			if m.checkAPIKey(c, config) {
				success = true
			}
		case "grpc":
			// Placeholder for gRPC Auth
			logger.GetLogger().Warn("gRPC Auth requested but not implemented",
				zap.String("slug", config.Path),
			)
			success = false
		case "gateway": 
			// Check Gateway Admin
			// Implementation depends on global middleware, but here we can flag it failed if not implemented/checked
			success = false
		default:
			// Fail securely on unknown auth types in strict mode
			success = false
		}

		if !success {
			logger.GetLogger().Warn("Authentication failed",
				zap.String("path", config.Path),
				zap.String("failed_auth_type", authType),
			)
			c.JSON(http.StatusUnauthorized, constants.BuildErrorResponse("Unauthorized", "Authentication failed for "+authType))
			return false
		}
	}

	// If no checks were performed (e.g. AuthType was empty or "none"), but we were called (AuthRequired=true),
	// this is a security failure (misconfiguration). Default Deny.
	if checksPerformed == 0 {
		logger.GetLogger().Warn("Authentication required but no valid auth types configured",
			zap.String("path", config.Path),
			zap.String("auth_type", config.AuthType),
		)
		c.JSON(http.StatusUnauthorized, constants.BuildErrorResponse("Unauthorized", "Configuration Error: No valid authentication method enabled"))
		return false
	}

	return true
}

func (m *DynamicURIMiddleware) checkCustomJWT(c *gin.Context, config *dto.APIConfigResponse) bool {
	// If no custom secret, use global middleware logic (RequireAuth)
	if config.JWTSecretKey == "" {
		m.jwtMiddleware.RequireAuth()(c)
		return !c.IsAborted()
	}

	// Custom JWT Logic
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return false
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return false
	}
	tokenString := parts[1]

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.JWTSecretKey), nil
	})

	if err != nil || !token.Valid {
		return false
	}

	// Validate Claims (Issuer/Audience)
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if config.JWTIssuer != "" {
			if iss, err := claims.GetIssuer(); err != nil || iss != config.JWTIssuer {
				return false
			}
		}
		if config.JWTAudience != "" {
			if aud, err := claims.GetAudience(); err != nil || !contains(aud, config.JWTAudience) {
				return false
			}
		}

		// Store user info in context
		c.Set("user_id", claims["sub"])
		c.Set("claims", claims)
		return true
	}

	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (m *DynamicURIMiddleware) checkBasicAuth(c *gin.Context, config *dto.APIConfigResponse) bool {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return false
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Basic" {
		return false
	}

	payload, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	pair := strings.SplitN(string(payload), ":", 2)
	if len(pair) != 2 {
		return false
	}
	username := pair[0]
	password := pair[1]

	for _, user := range config.BasicAuthUsers {
		if user.Username == username && user.Password == password { // Plaintext check for now
			c.Set("user_username", username)
			return true
		}
	}
	return false
}

func (m *DynamicURIMiddleware) checkAPIKey(c *gin.Context, config *dto.APIConfigResponse) bool {
	keyName := config.APIKeyHeader
	if keyName == "" {
		keyName = "X-API-Key"
	}

	var keyValue string
	if config.APIKeyLocation == "query" {
		keyValue = c.Query(keyName)
	} else {
		keyValue = c.GetHeader(keyName)
	}

	if keyValue == "" {
		return false
	}

	for _, key := range config.APIKeys {
		if key.Active && key.Key == keyValue {
			c.Set("api_key_name", key.Name)
			return true
		}
	}
	return false
}

