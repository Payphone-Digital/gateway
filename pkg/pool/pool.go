package pool

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// PoolConfig defines connection pool configuration
type PoolConfig struct {
	MaxConnections      int           `json:"max_connections"`
	MinIdleConnections  int           `json:"min_idle_connections"`
	ConnectionTimeout   time.Duration `json:"connection_timeout"`
	ReadTimeout         time.Duration `json:"read_timeout"`
	WriteTimeout        time.Duration `json:"write_timeout"`
	IdleTimeout         time.Duration `json:"idle_timeout"`
	MaxIdleConns        int           `json:"max_idle_conns"`
	MaxIdleConnsPerHost int           `json:"max_idle_conns_per_host"`
}

// DefaultPoolConfig returns sensible defaults
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxConnections:      100,
		MinIdleConnections:  10,
		ConnectionTimeout:   5 * time.Second,
		ReadTimeout:         30 * time.Second,
		WriteTimeout:        30 * time.Second,
		IdleTimeout:         90 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
	}
}

// BackendHealth tracks backend health status
type BackendHealth struct {
	Address      string
	IsHealthy    bool
	LastCheck    time.Time
	LastError    error
	FailureCount int
	SuccessCount int
}

// ConnectionPool manages HTTP and gRPC connections
type ConnectionPool struct {
	mu          sync.RWMutex
	httpClients map[string]*http.Client
	grpcConns   map[string]*grpc.ClientConn
	healthStats map[string]*BackendHealth
	config      PoolConfig
	logger      *zap.Logger
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config PoolConfig, logger *zap.Logger) *ConnectionPool {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &ConnectionPool{
		httpClients: make(map[string]*http.Client),
		grpcConns:   make(map[string]*grpc.ClientConn),
		healthStats: make(map[string]*BackendHealth),
		config:      config,
		logger:      logger,
	}
}

// GetHTTPClient returns an HTTP client for the given address
func (p *ConnectionPool) GetHTTPClient(address string, tlsEnabled bool) *http.Client {
	p.mu.RLock()
	client, exists := p.httpClients[address]
	p.mu.RUnlock()

	if exists {
		return client
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double check after acquiring write lock
	if client, exists = p.httpClients[address]; exists {
		return client
	}

	// Create new HTTP client with connection pooling
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   p.config.ConnectionTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          p.config.MaxIdleConns,
		MaxIdleConnsPerHost:   p.config.MaxIdleConnsPerHost,
		IdleConnTimeout:       p.config.IdleTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}

	if tlsEnabled {
		transport.TLSClientConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	client = &http.Client{
		Transport: transport,
		Timeout:   p.config.ReadTimeout + p.config.WriteTimeout,
	}

	p.httpClients[address] = client
	p.healthStats[address] = &BackendHealth{
		Address:   address,
		IsHealthy: true,
		LastCheck: time.Now(),
	}

	p.logger.Info("Created new HTTP client",
		zap.String("address", address),
		zap.Bool("tls_enabled", tlsEnabled),
	)

	return client
}

// GetGRPCConnection returns a gRPC connection for the given address
func (p *ConnectionPool) GetGRPCConnection(ctx context.Context, address string, tlsEnabled bool) (*grpc.ClientConn, error) {
	p.mu.RLock()
	conn, exists := p.grpcConns[address]
	p.mu.RUnlock()

	if exists && conn.GetState() != connectivity.Shutdown {
		return conn, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double check after acquiring write lock
	if conn, exists = p.grpcConns[address]; exists && conn.GetState() != connectivity.Shutdown {
		return conn, nil
	}

	// Create new gRPC connection
	var opts []grpc.DialOption

	if tlsEnabled {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
		})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Add connection timeout
	dialCtx, cancel := context.WithTimeout(ctx, p.config.ConnectionTimeout)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, address, opts...)
	if err != nil {
		p.logger.Error("Failed to create gRPC connection",
			zap.String("address", address),
			zap.Error(err),
		)
		return nil, err
	}

	p.grpcConns[address] = conn
	p.healthStats[address] = &BackendHealth{
		Address:   address,
		IsHealthy: true,
		LastCheck: time.Now(),
	}

	p.logger.Info("Created new gRPC connection",
		zap.String("address", address),
		zap.Bool("tls_enabled", tlsEnabled),
	)

	return conn, nil
}

// RecordSuccess records a successful request to a backend
func (p *ConnectionPool) RecordSuccess(address string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if health, exists := p.healthStats[address]; exists {
		health.IsHealthy = true
		health.SuccessCount++
		health.LastCheck = time.Now()
		health.LastError = nil
	}
}

// RecordFailure records a failed request to a backend
func (p *ConnectionPool) RecordFailure(address string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if health, exists := p.healthStats[address]; exists {
		health.FailureCount++
		health.LastCheck = time.Now()
		health.LastError = err
	}
}

// IsHealthy checks if a backend is healthy
func (p *ConnectionPool) IsHealthy(address string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if health, exists := p.healthStats[address]; exists {
		return health.IsHealthy
	}
	return true // Assume healthy if not tracked
}

// GetHealthStats returns health stats for all backends
func (p *ConnectionPool) GetHealthStats() map[string]*BackendHealth {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]*BackendHealth)
	for addr, health := range p.healthStats {
		statsCopy := *health
		stats[addr] = &statsCopy
	}
	return stats
}

// CloseGRPCConnection closes a specific gRPC connection
func (p *ConnectionPool) CloseGRPCConnection(address string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, exists := p.grpcConns[address]; exists {
		delete(p.grpcConns, address)
		return conn.Close()
	}
	return nil
}

// CloseAllConnections closes all connections
func (p *ConnectionPool) CloseAllConnections() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error

	// Close all gRPC connections
	for addr, conn := range p.grpcConns {
		if err := conn.Close(); err != nil {
			lastErr = err
			p.logger.Error("Failed to close gRPC connection",
				zap.String("address", addr),
				zap.Error(err),
			)
		}
		delete(p.grpcConns, addr)
	}

	// Close HTTP transports
	for addr, client := range p.httpClients {
		if transport, ok := client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
		delete(p.httpClients, addr)
	}

	p.logger.Info("Closed all connections")
	return lastErr
}

// Stats returns pool statistics
func (p *ConnectionPool) Stats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"http_clients": len(p.httpClients),
		"grpc_conns":   len(p.grpcConns),
		"health_stats": len(p.healthStats),
	}
}
