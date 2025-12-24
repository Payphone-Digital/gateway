package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/Payphone-Digital/gateway/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggingMiddleware logs HTTP requests and responses
func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			// Use Zap for structured logging
			logger.LogRequest(
				param.Method,
				param.Path,
				param.StatusCode,
				param.Latency.Milliseconds(),
				param.ClientIP,
				param.Request.UserAgent(),
			)

			// Additional context logging
			if param.ErrorMessage != "" {
				logger.GetLogger().Error("Request error",
					zap.String("error", param.ErrorMessage),
					zap.String("method", param.Method),
					zap.String("path", param.Path),
					zap.String("client_ip", param.ClientIP),
					zap.Int("status_code", param.StatusCode),
					zap.Duration("latency", param.Latency),
				)
			}

			// Log slow requests
			if param.Latency > time.Second*2 {
				logger.GetLogger().Warn("Slow request detected",
					zap.String("method", param.Method),
					zap.String("path", param.Path),
					zap.Duration("latency", param.Latency),
					zap.String("client_ip", param.ClientIP),
				)
			}

			return "" // Return empty string to prevent default logging
		},
		Output: io.Discard, // Discard default output since we're using Zap
	})
}

// RequestResponseMiddleware logs detailed request and response information
func RequestResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Read request body for logging (only for small requests)
		var requestBody []byte
		if c.Request.Body != nil && c.Request.ContentLength < 1024*1024 { // 1MB limit
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)

		// Log request completion
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("status_code", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.Int("response_size", c.Writer.Size()),
		}

		// Add request body for debugging (in development only)
		if gin.Mode() == gin.DebugMode && len(requestBody) > 0 {
			fields = append(fields, zap.ByteString("request_body", requestBody))
		}

		// Determine log level based on status code
		switch {
		case c.Writer.Status() >= 500:
			logger.GetLogger().Error("Server error", fields...)
		case c.Writer.Status() >= 400:
			logger.GetLogger().Warn("Client error", fields...)
		case latency > time.Second*2:
			logger.GetLogger().Warn("Slow request", fields...)
		default:
			logger.GetLogger().Info("Request completed", fields...)
		}
	}
}

// RecoveryMiddleware recovers from panics and logs them
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.LogPanic(recovered)

		c.JSON(500, gin.H{
			"error": "Internal server error",
			"code":  "INTERNAL_ERROR",
		})
	})
}

// SecurityLoggingMiddleware logs security-related events
func SecurityLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Detect suspicious activity
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// Log suspicious user agents
		if isSuspiciousUserAgent(userAgent) {
			logger.GetLogger().Warn("Suspicious user agent detected",
				zap.String("client_ip", clientIP),
				zap.String("user_agent", userAgent),
				zap.String("path", c.Request.URL.Path),
			)
		}

		// Log authentication attempts (if applicable)
		if c.Request.URL.Path == "/api/v1/auth/login" && c.Request.Method == "POST" {
			logger.GetLogger().Info("Login attempt",
				zap.String("client_ip", clientIP),
				zap.String("user_agent", userAgent),
			)
		}

		c.Next()
	}
}

// isSuspiciousUserAgent checks for common suspicious user agent patterns
func isSuspiciousUserAgent(userAgent string) bool {
	suspiciousPatterns := []string{
		"sqlmap", "nikto", "nmap", "masscan", "zap", "burp",
		"scanner", "bot", "crawler", "spider",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(strings.ToLower(userAgent), pattern) {
			return true
		}
	}

	return false
}


