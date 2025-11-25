package middleware

import (
	"context"
	"net/http"
	"time"

	ctxutil "github.com/surdiana/gateway/pkg/context"
	"github.com/surdiana/gateway/pkg/logger"
	"github.com/gin-gonic/gin"
)

// ContextMiddleware middleware untuk context management
func ContextMiddleware(module string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create context with request information
		function := c.Request.URL.Path
		ctx := ctxutil.NewContext(c.Request.Context(), c.Request, module, function)

		// Add timeout to context (default 30 seconds)
		ctx, cancel := ctxutil.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Store context in Gin
		c.Request = c.Request.WithContext(ctx)

		// Log request start
		logger.InfoWithContext(ctx, "Request started").
			String("method", c.Request.Method).
			String("path", c.Request.URL.Path).
			String("query", c.Request.URL.RawQuery).
			Log()

		// Process request
		c.Next()

		// Log request completion
		logger.InfoWithContext(ctx, "Request completed").
			String("method", c.Request.Method).
			String("path", c.Request.URL.Path).
			Int("status_code", c.Writer.Status()).
			Int("response_size", c.Writer.Size()).
			Duration(ctxutil.GetDuration(ctx)).
			Log()
	}
}

// RequestTimeoutMiddleware middleware untuk timeout per request
func RequestTimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := ctxutil.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		// Check if context is done before processing
		select {
		case <-ctx.Done():
			logger.WarnWithContext(ctx, "Request timeout before processing").
				Duration(timeout).
				Log()
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error":   "Request timeout",
				"timeout": timeout.String(),
			})
			c.Abort()
			return
		default:
			c.Next()
		}
	}
}

// UserContextMiddleware middleware untuk menambahkan user ke context
func UserContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from JWT token or session
		userID := getUserFromRequest(c)
		if userID != nil {
			ctx := ctxutil.WithUserID(c.Request.Context(), userID)
			c.Request = c.Request.WithContext(ctx)
		}

		c.Next()
	}
}

// CorrelationMiddleware middleware untuk correlation ID
func CorrelationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get correlation ID from header or create new
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = c.GetHeader("X-Request-ID")
		}
		if correlationID == "" {
			correlationID = c.GetHeader("X-Trace-ID")
		}

		ctx := c.Request.Context()
		if correlationID != "" {
			ctx = context.WithValue(ctx, ctxutil.CorrelationIDKey, correlationID)
		} else {
			// Generate new correlation ID
			correlationID = ctxutil.GetCorrelationID(ctx)
		}

		// Set response header
		c.Header("X-Correlation-ID", correlationID)

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// TracingMiddleware middleware untuk distributed tracing
func TracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Add trace ID if not exists
		if ctxutil.GetTraceID(ctx) == "" {
			traceID := ctxutil.GetTraceID(ctx)
			ctx = context.WithValue(ctx, ctxutil.TraceIDKey, traceID)
			c.Request = c.Request.WithContext(ctx)
		}

		c.Next()
	}
}

// ContextValidationMiddleware middleware untuk validasi context
func ContextValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Validate context is not cancelled
		if err := ctx.Err(); err != nil {
			logger.WarnWithContext(ctx, "Context already cancelled").
				Err(err).
				Log()
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Request cancelled",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// PerformanceMiddleware middleware untuk monitoring performance
func PerformanceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		ctx := c.Request.Context()

		// Add performance monitoring
		defer func() {
			duration := time.Since(start)

			// Log slow requests
			if duration > 5*time.Second {
				logger.WarnWithContext(ctx, "Slow request detected").
					Duration(duration).
					String("method", c.Request.Method).
					String("path", c.Request.URL.Path).
					Log()
			}

			// Add duration to context
			ctx = context.WithValue(ctx, ctxutil.StartTimeKey, start)
		}()

		c.Next()
	}
}

// SecurityContextMiddleware middleware untuk security context
func SecurityContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Add security information to context
		clientIP := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")

		// Check for suspicious patterns
		if isSuspiciousRequest(c) {
			logger.WarnWithContext(ctx, "Suspicious request detected").
				String("client_ip", clientIP).
				String("user_agent", userAgent).
				String("method", c.Request.Method).
				String("path", c.Request.URL.Path).
				Log()
		}

		// Add security context
		ctx = context.WithValue(ctx, "security_checked", true)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// Helper functions
func getUserFromRequest(c *gin.Context) interface{} {
	// Try to get user from JWT token
	if userID, exists := c.Get("user_id"); exists {
		return userID
	}

	// Try to get from context (set by auth middleware)
	if userID := ctxutil.GetUserID(c.Request.Context()); userID != nil {
		return userID
	}

	return nil
}

func isSuspiciousRequest(c *gin.Context) bool {
	userAgent := c.GetHeader("User-Agent")
	clientIP := c.ClientIP()
	path := c.Request.URL.Path
	method := c.Request.Method

	// Check for suspicious patterns
	suspiciousPatterns := []struct {
		field string
		value string
	}{
		{"user_agent", "sqlmap"},
		{"user_agent", "nikto"},
		{"user_agent", "nmap"},
		{"user_agent", "scanner"},
		{"path", "/admin"},
		{"path", "/wp-admin"},
		{"method", "TRACE"},
		{"method", "TRACK"},
	}

	for _, pattern := range suspiciousPatterns {
		var value string
		switch pattern.field {
		case "user_agent":
			value = userAgent
		case "client_ip":
			value = clientIP
		case "path":
			value = path
		case "method":
			value = method
		}

		if value == pattern.value {
			return true
		}
	}

	return false
}

// ContextHelper helper functions untuk context management
type ContextHelper struct{}

func NewContextHelper() *ContextHelper {
	return &ContextHelper{}
}

// WithContext helper function untuk menambahkan context ke handler
func (ch *ContextHelper) WithContext(module, function string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := ctxutil.NewContext(c.Request.Context(), c.Request, module, function)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// WithTimeout helper function untuk timeout
func (ch *ContextHelper) WithTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := ctxutil.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// DefaultContextMiddleware kombinasi middleware default
func DefaultContextMiddleware(module string) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		CorrelationMiddleware(),
		TracingMiddleware(),
		ContextValidationMiddleware(),
		SecurityContextMiddleware(),
		PerformanceMiddleware(),
		RequestTimeoutMiddleware(30 * time.Second),
		UserContextMiddleware(),
		ContextMiddleware(module),
	}
}
