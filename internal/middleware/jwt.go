package middleware

import (
	"net/http"
	"strings"

	"github.com/Payphone-Digital/gateway/internal/repository"
	"github.com/Payphone-Digital/gateway/internal/service"
	"github.com/Payphone-Digital/gateway/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type JWTMiddleware struct {
	jwtService *service.JWTService
	userRepo   *repository.UserRepository
}

func NewJWTMiddleware(jwtService *service.JWTService, userRepo *repository.UserRepository) *JWTMiddleware {
	return &JWTMiddleware{
		jwtService: jwtService,
		userRepo:   userRepo,
	}
}

// RequireAuth validates JWT token and sets user info in context
func (m *JWTMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.GetLogger().Warn("Missing Authorization header",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method))
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Unauthorized",
			})
			c.Abort()
			return
		}

		// Check Bearer token format
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			logger.GetLogger().Warn("Invalid Authorization header format",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.String("auth_header", authHeader))
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Unauthorized",
			})
			c.Abort()
			return
		}

		tokenString := tokenParts[1]

		// First, validate token structure and expiry
		claims, err := m.jwtService.ValidateToken(tokenString)
		if err != nil {
			logger.GetLogger().Warn("Invalid or expired token",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Unauthorized",
			})
			c.Abort()
			return
		}

		// Extract user ID from claims
		userIDFloat, ok := (*claims)["user_id"].(float64)
		if !ok {
			logger.GetLogger().Warn("Invalid user ID in token",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method))
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Unauthorized",
			})
			c.Abort()
			return
		}

		userID := uint(userIDFloat)

		// Get user from database to check token version
		ctx := c.Request.Context()
		user, err := m.userRepo.GetByID(ctx, int(userID))
		if err != nil {
			logger.GetLogger().Warn("User not found in database",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.Uint("user_id", userID),
				zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Unauthorized",
			})
			c.Abort()
			return
		}

		// Validate token version
		tokenVersionFloat, ok := (*claims)["token_version"].(float64)
		if !ok {
			logger.GetLogger().Warn("Invalid token version in token",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.Uint("user_id", userID))
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Unauthorized",
			})
			c.Abort()
			return
		}

		tokenVersion := int(tokenVersionFloat)
		if tokenVersion != user.TokenVersion {
			logger.GetLogger().Warn("Token version mismatch - token has been invalidated",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.Uint("user_id", userID),
				zap.Int("token_version", tokenVersion),
				zap.Int("db_version", user.TokenVersion))
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Unauthorized",
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", userID)
		c.Set("email", (*claims)["email"].(string))
		c.Set("first_name", (*claims)["first_name"].(string))
		c.Set("last_name", (*claims)["last_name"].(string))

		logger.GetLogger().Debug("User authenticated successfully",
			zap.Uint("user_id", userID),
			zap.String("email", (*claims)["email"].(string)),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method))

		c.Next()
	}
}

// OptionalAuth checks for token but doesn't require it
func (m *JWTMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.Next()
			return
		}

		tokenString := tokenParts[1]
		claims, err := m.jwtService.ValidateToken(tokenString)
		if err != nil {
			c.Next()
			return
		}

		// Set user information in context if token is valid
		c.Set("user_id", uint((*claims)["user_id"].(float64)))
		c.Set("email", (*claims)["email"].(string))
		c.Set("first_name", (*claims)["first_name"].(string))
		c.Set("last_name", (*claims)["last_name"].(string))

		c.Next()
	}
}
