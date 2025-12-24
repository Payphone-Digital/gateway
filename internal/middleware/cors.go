// middleware/cors.go
package middleware

import (
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Payphone-Digital/gateway/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	db              *gorm.DB
	allowedOrigins  map[string]bool
	mu              sync.RWMutex
	lastRefresh     time.Time
	refreshInterval time.Duration
}

// NewCORSConfig creates a new CORS configuration
func NewCORSConfig(db *gorm.DB) *CORSConfig {
	config := &CORSConfig{
		db:              db,
		allowedOrigins:  make(map[string]bool),
		refreshInterval: 5 * time.Minute, // Refresh allowed origins every 5 minutes
	}
	
	// Initial load
	config.refreshAllowedOrigins()
	
	return config
}

// refreshAllowedOrigins loads allowed origins from url_configs table
func (c *CORSConfig) refreshAllowedOrigins() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	var urlConfigs []struct {
		BaseURL string `gorm:"column:base_url"`
	}
	
	// Query all unique base_urls from url_configs
	if err := c.db.Table("url_configs").
		Select("DISTINCT base_url").
		Where("base_url IS NOT NULL AND base_url != ''").
		Find(&urlConfigs).Error; err != nil {
		logger.GetLogger().Error("Failed to load CORS allowed origins from database",
			zap.Error(err),
		)
		return
	}
	
	// Clear and rebuild allowed origins map
	c.allowedOrigins = make(map[string]bool)
	
	// Always allow localhost and management frontend
	c.allowedOrigins["http://localhost:5173"] = true
	c.allowedOrigins["http://127.0.0.1:5173"] = true
	c.allowedOrigins["http://192.168.100.128:5173"] = true // Frontend dev server
	
	// Add origins from database
	for _, config := range urlConfigs {
		if config.BaseURL != "" {
			// Parse URL to extract origin
			if parsedURL, err := url.Parse(config.BaseURL); err == nil {
				origin := parsedURL.Scheme + "://" + parsedURL.Host
				c.allowedOrigins[origin] = true
				logger.GetLogger().Debug("CORS: Added allowed origin from URL config",
					zap.String("origin", origin),
				)
			}
		}
	}
	
	c.lastRefresh = time.Now()
	
	logger.GetLogger().Info("CORS: Allowed origins refreshed",
		zap.Int("count", len(c.allowedOrigins)),
		zap.Time("refresh_time", c.lastRefresh),
	)
}

// isOriginAllowed checks if an origin is in the allowed list
func (c *CORSConfig) isOriginAllowed(origin string) bool {
	// Refresh if needed
	if time.Since(c.lastRefresh) > c.refreshInterval {
		go c.refreshAllowedOrigins()
	}
	
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Normalize origin (remove trailing slash)
	origin = strings.TrimSuffix(origin, "/")
	
	return c.allowedOrigins[origin]
}

// CORS returns a CORS middleware with dynamic origin validation
func CORS(db *gorm.DB) gin.HandlerFunc {
	corsConfig := NewCORSConfig(db)
	
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

		// Determine allowed origin
		allowedOrigin := "*"
		if origin != "" {
			if corsConfig.isOriginAllowed(origin) {
				allowedOrigin = origin
				logger.GetLogger().Debug("CORS: Origin allowed",
					zap.String("origin", origin),
				)
			} else {
				logger.GetLogger().Warn("CORS: Origin not in allowed list",
					zap.String("origin", origin),
					zap.String("client_ip", clientIP),
				)
				// Still allow but log warning
				allowedOrigin = origin
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-API-Key")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

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
			zap.String("allowed_origin", allowedOrigin),
		)

		c.Next()
	}
}
