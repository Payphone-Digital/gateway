package service

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Payphone-Digital/gateway/internal/dto"
	"github.com/Payphone-Digital/gateway/pkg/logger"
	"github.com/Payphone-Digital/gateway/pkg/redis"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CacheService struct {
	redisClient redis.Client
}

type CacheConfig struct {
	DefaultTTL   time.Duration
	EnableCache  bool
	KeyPrefix    string
	MaxKeyLength int
}

// NewCacheService creates a new cache service
func NewCacheService(redisClient redis.Client) *CacheService {
	return &CacheService{
		redisClient: redisClient,
	}
}

// GenerateCacheKey creates a unique cache key for integration request
func (s *CacheService) GenerateCacheKey(config *dto.APIConfigResponse, c *gin.Context, uriParams map[string]string) string {
	// Create a hash of the request details for consistent key generation
	h := md5.New()

	// Add config details
	h.Write([]byte(fmt.Sprintf("slug:%s:method:%s:uri:%s",
		config.Path, config.Method, config.URI)))

	// Add request method and path
	h.Write([]byte(fmt.Sprintf(":%s:%s", c.Request.Method, c.Request.URL.Path)))

	// Add query parameters (sorted for consistency)
	queryParams := c.Request.URL.Query()
	if len(queryParams) > 0 {
		// Sort query parameters for consistent hashing
		sortedKeys := make([]string, 0, len(queryParams))
		for k := range queryParams {
			sortedKeys = append(sortedKeys, k)
		}

		for _, k := range sortedKeys {
			values := queryParams[k]
			for _, v := range values {
				h.Write([]byte(fmt.Sprintf(":%s=%s", k, v)))
			}
		}
	}

	// Add URI parameters if any
	if len(uriParams) > 0 {
		// Sort URI parameters for consistent hashing
		sortedKeys := make([]string, 0, len(uriParams))
		for k := range uriParams {
			sortedKeys = append(sortedKeys, k)
		}

		for _, k := range sortedKeys {
			h.Write([]byte(fmt.Sprintf(":%s=%s", k, uriParams[k])))
		}
	}

	// Add selected headers that might affect response (like Authorization, Content-Type, etc.)
	headersToCache := []string{"authorization", "content-type", "accept", "user-agent"}
	for _, header := range headersToCache {
		if value := c.GetHeader(header); value != "" {
			h.Write([]byte(fmt.Sprintf(":%s:%s", header, value)))
		}
	}

	// Add request body for POST/PUT requests (limit size to prevent huge keys)
	if c.Request.Body != nil && (c.Request.Method == "POST" || c.Request.Method == "PUT") {
		body := c.GetString("request_body")
		if len(body) > 1000 { // Limit body size for cache key
			body = body[:1000]
		}
		h.Write([]byte(fmt.Sprintf(":body:%s", body)))
	}

	// Generate final key
	keyHash := fmt.Sprintf("%x", h.Sum(nil))
	return fmt.Sprintf("integration:%s:%s", config.Path, keyHash)
}

// GetCachedResponse retrieves cached response if available and valid
func (s *CacheService) GetCachedResponse(ctx context.Context, cacheKey string) ([]byte, int, map[string]string, bool) {
	if s.redisClient == nil {
		return nil, 0, nil, false
	}

	item, err := s.redisClient.GetIntegrationResponse(ctx, cacheKey)
	if err != nil {
		logger.GetLogger().Error("Failed to get cached response",
			zap.String("cache_key", cacheKey),
			zap.Error(err),
		)
		return nil, 0, nil, false
	}

	if item == nil {
		// Cache miss
		return nil, 0, nil, false
	}

	// Convert data back to bytes
	var data []byte
	if strData, ok := item.Data.(string); ok {
		data = []byte(strData)
	} else {
		// If it's not a string, convert it to JSON then bytes
		jsonData, err := json.Marshal(item.Data)
		if err != nil {
			logger.GetLogger().Error("Failed to marshal cached data",
				zap.String("cache_key", cacheKey),
				zap.Error(err),
			)
			return nil, 0, nil, false
		}
		data = jsonData
	}

	logger.GetLogger().Info("Cache hit",
		zap.String("cache_key", cacheKey),
		zap.Int("status", item.Status),
		zap.Int("data_size", len(data)),
	)

	return data, item.Status, item.Headers, true
}

// SetCachedResponse stores response in cache
func (s *CacheService) SetCachedResponse(ctx context.Context, cacheKey string, data []byte, status int, headers map[string]string, config *dto.APIConfigResponse) error {
	if s.redisClient == nil {
		return nil // Cache is disabled
	}

	// Don't cache error responses (except 404 which might be legitimate)
	if status >= 400 && status != 404 {
		logger.GetLogger().Debug("Skipping cache for error response",
			zap.String("cache_key", cacheKey),
			zap.Int("status", status),
		)
		return nil
	}

	// Don't cache large responses (adjust threshold as needed)
	if len(data) > 1024*1024 { // 1MB limit
		logger.GetLogger().Debug("Skipping cache for large response",
			zap.String("cache_key", cacheKey),
			zap.Int("data_size", len(data)),
		)
		return nil
	}

	// Determine TTL based on response and config
	ttl := s.determineTTL(status, config)

	if err := s.redisClient.SetIntegrationResponse(ctx, cacheKey, data, status, headers, ttl); err != nil {
		logger.GetLogger().Error("Failed to set cached response",
			zap.String("cache_key", cacheKey),
			zap.Duration("ttl", ttl),
			zap.Error(err),
		)
		return err
	}

	logger.GetLogger().Info("Response cached",
		zap.String("cache_key", cacheKey),
		zap.Int("status", status),
		zap.Int("data_size", len(data)),
		zap.Duration("ttl", ttl),
	)

	return nil
}

// determineTTL calculates appropriate TTL based on response status and config
func (s *CacheService) determineTTL(status int, config *dto.APIConfigResponse) time.Duration {
	// Default TTL
	defaultTTL := 5 * time.Minute

	// Adjust based on HTTP status
	switch status {
	case http.StatusOK:
		return 10 * time.Minute // Success responses can be cached longer
	case http.StatusCreated:
		return 5 * time.Minute // Created responses
	case http.StatusNoContent:
		return 1 * time.Minute // No content responses
	case http.StatusNotFound:
		return 2 * time.Minute // 404s can be cached briefly
	case http.StatusTooManyRequests:
		return 30 * time.Second // Rate limit responses
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return 10 * time.Second // Error responses cached very briefly
	default:
		return defaultTTL
	}
}

// InvalidateCache removes cached responses for specific config
func (s *CacheService) InvalidateCache(ctx context.Context, slug string) error {
	if s.redisClient == nil {
		return nil
	}

	pattern := fmt.Sprintf("integration:%s:*", slug)
	return s.redisClient.DeleteByPattern(ctx, pattern)
}

// InvalidateCacheByURI removes cached responses for specific URI pattern
func (s *CacheService) InvalidateCacheByURI(ctx context.Context, uri string) error {
	if s.redisClient == nil {
		return nil
	}

	// This would require storing URI in cache key or maintaining a separate index
	// For now, we'll use a pattern-based approach if URI contains identifiable parts
	if strings.Contains(uri, "{") {
		// This is a pattern URI, we can't easily invalidate specific entries
		// You might need to maintain a separate index for this
		return nil
	}

	// For exact URIs, we can try pattern matching
	pattern := fmt.Sprintf("integration:*:%x", md5.Sum([]byte(uri)))
	return s.redisClient.DeleteByPattern(ctx, pattern+"*")
}

// GetCacheStats returns cache statistics
func (s *CacheService) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	if s.redisClient == nil {
		return map[string]interface{}{
			"enabled": false,
		}, nil
	}

	stats, err := s.redisClient.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	stats["enabled"] = true
	return stats, nil
}

// ClearAll clears all cache (use with caution)
func (s *CacheService) ClearAll(ctx context.Context) error {
	if s.redisClient == nil {
		return nil
	}
	return s.redisClient.FlushAll(ctx)
}

// ShouldCache determines if a response should be cached based on various factors
func (s *CacheService) ShouldCache(c *gin.Context, config *dto.APIConfigResponse, status int, dataSize int) bool {
	// Don't cache if disabled
	if s.redisClient == nil {
		return false
	}

	// Don't cache error responses (except some specific cases)
	if status >= 400 && status != 404 && status != http.StatusTooManyRequests {
		return false
	}

	// Don't cache if no-cache header is present
	if strings.ToLower(c.GetHeader("Cache-Control")) == "no-cache" {
		return false
	}

	// Don't cache if authorization header might make this user-specific
	// (unless you want per-user caching)
	if c.GetHeader("Authorization") != "" {
		// This is user-specific data, you might want to cache it separately
		// For now, we'll skip caching authenticated requests
		return false
	}

	// Don't cache very large responses
	if dataSize > 1024*1024 { // 1MB
		return false
	}

	// Don't cache streaming responses
	if strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "stream") {
		return false
	}

	return true
}
