package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Client defines Redis client interface
type Client interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeleteByPattern(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) (bool, error)
	GetStats(ctx context.Context) (map[string]interface{}, error)
	FlushAll(ctx context.Context) error
	Ping(ctx context.Context) error
	IsEnabled() bool
	Close() error

	// Integration-specific methods for backward compatibility
	SetIntegrationResponse(ctx context.Context, key string, data []byte, status int, headers map[string]string, ttl time.Duration) error
	GetIntegrationResponse(ctx context.Context, key string) (*CacheItem, error)
}

// Config holds Redis configuration
type Config struct {
	Host         string
	Port         int
	Password     string
	DB           int
	Enabled      bool
	PoolSize     int
	MinIdleConns int
}

// CacheItem represents cached integration response
type CacheItem struct {
	Data      interface{}       `json:"data"`
	ExpiresAt time.Time         `json:"expires_at"`
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// RedisClient implements Client using Redis
type RedisClient struct {
	client  *redis.Client
	enabled bool
	logger  *zap.Logger
}

// NewClient creates a new Redis client with graceful fallback
func NewClient(config Config, logger *zap.Logger) Client {
	if !config.Enabled {
		logger.Info("Redis cache disabled")
		return &RedisClient{enabled: false, logger: logger}
	}

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn("Redis connection failed, running in disabled mode",
			zap.String("host", config.Host),
			zap.Int("port", config.Port),
			zap.Error(err),
		)
		return &RedisClient{enabled: false, logger: logger}
	}

	logger.Info("Redis client connected successfully",
		zap.String("host", config.Host),
		zap.Int("port", config.Port),
		zap.Int("db", config.DB),
		zap.Int("pool_size", config.PoolSize),
	)

	return &RedisClient{
		client:  client,
		enabled: true,
		logger:  logger,
	}
}

func (c *RedisClient) IsEnabled() bool {
	return c.enabled
}

func (c *RedisClient) Ping(ctx context.Context) error {
	if !c.enabled {
		return fmt.Errorf("cache disabled")
	}
	return c.client.Ping(ctx).Err()
}

func (c *RedisClient) Get(ctx context.Context, key string, dest interface{}) error {
	if !c.enabled {
		return fmt.Errorf("cache disabled")
	}

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			c.logger.Debug("Cache miss", zap.String("key", key))
		} else {
			c.logger.Error("Cache get error", zap.String("key", key), zap.Error(err))
		}
		return err
	}

	c.logger.Debug("Cache hit", zap.String("key", key))
	return json.Unmarshal(data, dest)
}

func (c *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !c.enabled {
		c.logger.Debug("Cache set skipped (disabled)", zap.String("key", key))
		return nil // Silent fail when disabled
	}

	data, err := json.Marshal(value)
	if err != nil {
		c.logger.Error("Failed to marshal cache value", zap.String("key", key), zap.Error(err))
		return err
	}

	err = c.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		c.logger.Error("Cache set error", zap.String("key", key), zap.Error(err))
		return err
	}

	c.logger.Debug("Cache set successfully",
		zap.String("key", key),
		zap.Duration("ttl", ttl),
		zap.Int("data_size", len(data)),
	)
	return nil
}

func (c *RedisClient) Delete(ctx context.Context, key string) error {
	if !c.enabled {
		return nil // Silent fail when disabled
	}

	err := c.client.Del(ctx, key).Err()
	if err != nil {
		c.logger.Error("Cache delete error", zap.String("key", key), zap.Error(err))
		return err
	}

	c.logger.Debug("Cache deleted successfully", zap.String("key", key))
	return nil
}

func (c *RedisClient) DeleteByPattern(ctx context.Context, pattern string) error {
	if !c.enabled {
		return nil // Silent fail when disabled
	}

	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		c.logger.Error("Failed to get keys by pattern",
			zap.String("pattern", pattern),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get keys by pattern: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		c.logger.Error("Failed to delete cache by pattern",
			zap.String("pattern", pattern),
			zap.Strings("keys", keys),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete cache by pattern: %w", err)
	}

	c.logger.Info("Cache deleted by pattern successfully",
		zap.String("pattern", pattern),
		zap.Int("deleted_count", len(keys)),
	)
	return nil
}

func (c *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	if !c.enabled {
		return false, fmt.Errorf("cache disabled")
	}

	result, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}
	return result > 0, nil
}

func (c *RedisClient) GetStats(ctx context.Context) (map[string]interface{}, error) {
	if !c.enabled {
		return map[string]interface{}{
			"enabled": false,
			"message": "Redis cache is disabled",
		}, nil
	}

	info, err := c.client.Info(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	stats := make(map[string]interface{})
	stats["enabled"] = true
	stats["info"] = info

	// Get memory usage
	memoryInfo := c.client.Info(ctx, "memory").Val()
	stats["memory_info"] = memoryInfo

	// Get connection pool stats
	poolStats := c.client.PoolStats()
	stats["pool_stats"] = map[string]interface{}{
		"hits":        poolStats.Hits,
		"misses":      poolStats.Misses,
		"total_conns": poolStats.TotalConns,
		"idle_conns":  poolStats.IdleConns,
		"stale_conns": poolStats.StaleConns,
	}

	return stats, nil
}

func (c *RedisClient) FlushAll(ctx context.Context) error {
	if !c.enabled {
		return nil // Silent fail when disabled
	}

	if err := c.client.FlushAll(ctx).Err(); err != nil {
		return fmt.Errorf("failed to flush all cache: %w", err)
	}

	c.logger.Warn("All cache flushed")
	return nil
}

// SetIntegrationResponse caches response from integration API (backward compatibility)
func (c *RedisClient) SetIntegrationResponse(ctx context.Context, key string, data []byte, status int, headers map[string]string, ttl time.Duration) error {
	item := CacheItem{
		Data:      string(data),
		ExpiresAt: time.Now().Add(ttl),
		Status:    status,
		Headers:   headers,
	}

	return c.Set(ctx, key, item, ttl)
}

// GetIntegrationResponse retrieves cached integration response (backward compatibility)
func (c *RedisClient) GetIntegrationResponse(ctx context.Context, key string) (*CacheItem, error) {
	if !c.enabled {
		return nil, nil // Cache miss when disabled
	}

	var item CacheItem
	err := c.Get(ctx, key, &item)
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	// Check if item has expired (additional safety check)
	if time.Now().After(item.ExpiresAt) {
		// Remove expired item
		c.Delete(ctx, key)
		return nil, nil
	}

	c.logger.Debug("Integration cache hit",
		zap.String("key", key),
		zap.Time("expires_at", item.ExpiresAt),
		zap.Int("status", item.Status),
	)

	return &item, nil
}

// Close closes the Redis connection
func (c *RedisClient) Close() error {
	if c.enabled && c.client != nil {
		c.logger.Info("Closing Redis connection")
		return c.client.Close()
	}
	return nil
}
