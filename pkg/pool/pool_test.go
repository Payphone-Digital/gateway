package pool

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewConnectionPool(t *testing.T) {
	config := DefaultPoolConfig()
	logger := zap.NewNop()

	pool := NewConnectionPool(config, logger)
	if pool == nil {
		t.Fatal("Expected non-nil pool")
	}

	stats := pool.Stats()
	if stats["http_clients"].(int) != 0 {
		t.Errorf("Expected 0 http clients, got %d", stats["http_clients"].(int))
	}
}

func TestConnectionPool_GetHTTPClient(t *testing.T) {
	config := DefaultPoolConfig()
	pool := NewConnectionPool(config, nil)

	// Get client - should create new
	client1 := pool.GetHTTPClient("http://example.com", false)
	if client1 == nil {
		t.Fatal("Expected non-nil client")
	}

	// Get same client again - should return cached
	client2 := pool.GetHTTPClient("http://example.com", false)
	if client1 != client2 {
		t.Error("Expected same client instance")
	}

	// Stats should show 1 client
	stats := pool.Stats()
	if stats["http_clients"].(int) != 1 {
		t.Errorf("Expected 1 http client, got %d", stats["http_clients"].(int))
	}
}

func TestConnectionPool_HealthTracking(t *testing.T) {
	config := DefaultPoolConfig()
	pool := NewConnectionPool(config, nil)

	address := "http://test-backend.com"
	pool.GetHTTPClient(address, false)

	// Record success
	pool.RecordSuccess(address)
	if !pool.IsHealthy(address) {
		t.Error("Expected backend to be healthy after success")
	}

	// Record failure
	pool.RecordFailure(address, nil)
	stats := pool.GetHealthStats()
	if stats[address].FailureCount != 1 {
		t.Errorf("Expected 1 failure, got %d", stats[address].FailureCount)
	}
}

func TestConnectionPool_CloseAll(t *testing.T) {
	config := DefaultPoolConfig()
	pool := NewConnectionPool(config, nil)

	pool.GetHTTPClient("http://example1.com", false)
	pool.GetHTTPClient("http://example2.com", false)

	err := pool.CloseAllConnections()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	stats := pool.Stats()
	if stats["http_clients"].(int) != 0 {
		t.Errorf("Expected 0 http clients after close, got %d", stats["http_clients"].(int))
	}
}

func TestConnectionPool_ConcurrentAccess(t *testing.T) {
	config := DefaultPoolConfig()
	pool := NewConnectionPool(config, nil)

	// Concurrent access test
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			address := "http://concurrent-test.com"
			pool.GetHTTPClient(address, false)
			pool.RecordSuccess(address)
			pool.IsHealthy(address)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for goroutines")
		}
	}

	// Should only have 1 client
	stats := pool.Stats()
	if stats["http_clients"].(int) != 1 {
		t.Errorf("Expected 1 http client, got %d", stats["http_clients"].(int))
	}
}

func TestGRPCConnection_InvalidAddress(t *testing.T) {
	config := DefaultPoolConfig()
	config.ConnectionTimeout = 1 * time.Second
	pool := NewConnectionPool(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try to connect to invalid address - should still create conn object
	// (gRPC connections are lazy)
	conn, err := pool.GetGRPCConnection(ctx, "invalid-address:99999", false)
	if err != nil {
		// Expected for invalid address with timeout
		t.Logf("Got expected error: %v", err)
	} else if conn == nil {
		t.Error("Expected non-nil connection or error")
	}
}
