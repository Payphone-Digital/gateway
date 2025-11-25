// middleware/cors.go
package middleware

import (
	"github.com/surdiana/gateway/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")
		origin := c.GetHeader("Origin")

		logger.GetLogger().Debug("Middleware: CORS request processing",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
			zap.String("origin", origin),
		)

		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			logger.GetLogger().Debug("Middleware: CORS preflight request handled",
				zap.String("client_ip", clientIP),
				zap.String("origin", origin),
			)
			c.AbortWithStatus(204)
			return
		}

		logger.GetLogger().Debug("Middleware: CORS headers applied successfully",
			zap.String("client_ip", clientIP),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
		)

		c.Next()
	}
}
