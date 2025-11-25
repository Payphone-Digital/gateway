package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/surdiana/gateway/config"
	"github.com/surdiana/gateway/pkg/logger"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type Client struct {
	rdb *redis.Client
}

type CacheItem struct {
	Data      interface{} `json:"data"`
	ExpiresAt time.Time   `json:"expires_at"`
	Status    int         `json:"status"`
	Headers   map[string]string `json:"headers,omitempty"`
}

func NewClient(cfg *config.Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddress(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.Database,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
		PoolTimeout:  cfg.Redis.PoolTimeout,
	})

	client := &Client{rdb: rdb}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		logger.GetLogger().Error("Failed to connect to Redis",
			zap.String("address", cfg.RedisAddress()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.GetLogger().Info("Successfully connected to Redis",
		zap.String("address", cfg.RedisAddress()),
		zap.Int("database", cfg.Redis.Database),
	)

	return client, nil
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

// SetIntegrationResponse cache response from integration API
func (c *Client) SetIntegrationResponse(ctx context.Context, key string, data []byte, status int, headers map[string]string, ttl time.Duration) error {
	item := CacheItem{
		Data:      string(data),
		ExpiresAt: time.Now().Add(ttl),
		Status:    status,
		Headers:   headers,
	}

	jsonData, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal cache item: %w", err)
	}

	if err := c.rdb.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		logger.GetLogger().Error("Failed to set cache",
			zap.String("key", key),
			zap.Duration("ttl", ttl),
			zap.Error(err),
		)
		return fmt.Errorf("failed to set cache: %w", err)
	}

	logger.GetLogger().Debug("Cache set successfully",
		zap.String("key", key),
		zap.Duration("ttl", ttl),
		zap.Int("data_size", len(data)),
	)

	return nil
}

// GetIntegrationResponse retrieves cached integration response
func (c *Client) GetIntegrationResponse(ctx context.Context, key string) (*CacheItem, error) {
	data, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		logger.GetLogger().Error("Failed to get cache",
			zap.String("key", key),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	var item CacheItem
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		logger.GetLogger().Error("Failed to unmarshal cache item",
			zap.String("key", key),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to unmarshal cache item: %w", err)
	}

	// Check if item has expired (additional safety check)
	if time.Now().After(item.ExpiresAt) {
		// Remove expired item
		c.Delete(ctx, key)
		return nil, nil
	}

	logger.GetLogger().Debug("Cache hit successfully",
		zap.String("key", key),
		zap.Time("expires_at", item.ExpiresAt),
	)

	return &item, nil
}

// Delete removes cache entry
func (c *Client) Delete(ctx context.Context, key string) error {
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		logger.GetLogger().Error("Failed to delete cache",
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete cache: %w", err)
	}

	logger.GetLogger().Debug("Cache deleted successfully",
		zap.String("key", key),
	)

	return nil
}

// DeleteByPattern removes cache entries matching pattern
func (c *Client) DeleteByPattern(ctx context.Context, pattern string) error {
	keys, err := c.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys by pattern: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
		logger.GetLogger().Error("Failed to delete cache by pattern",
			zap.String("pattern", pattern),
			zap.Strings("keys", keys),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete cache by pattern: %w", err)
	}

	logger.GetLogger().Info("Cache deleted by pattern successfully",
		zap.String("pattern", pattern),
		zap.Int("deleted_count", len(keys)),
	)

	return nil
}

// Exists checks if key exists
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}
	return result > 0, nil
}

// GetStats returns Redis statistics
func (c *Client) GetStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := c.rdb.Info(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	// Parse basic info
	stats := make(map[string]interface{})
	stats["info"] = info

	// Get memory usage
	memoryInfo := c.rdb.Info(ctx, "memory").Val()
	stats["memory_info"] = memoryInfo

	// Get connection pool stats
	poolStats := c.rdb.PoolStats()
	stats["pool_stats"] = map[string]interface{}{
		"hits":        poolStats.Hits,
		"misses":      poolStats.Misses,
		"total_conns": poolStats.TotalConns,
		"idle_conns":  poolStats.IdleConns,
		"stale_conns": poolStats.StaleConns,
	}

	return stats, nil
}

// FlushAll clears all cache (use with caution)
func (c *Client) FlushAll(ctx context.Context) error {
	if err := c.rdb.FlushAll(ctx).Err(); err != nil {
		return fmt.Errorf("failed to flush all cache: %w", err)
	}

	logger.GetLogger().Warn("All cache flushed")
	return nil
}