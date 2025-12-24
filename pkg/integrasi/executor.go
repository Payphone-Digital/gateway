package integrasi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Payphone-Digital/gateway/internal/dto"
	"github.com/Payphone-Digital/gateway/pkg/circuit"
	"github.com/Payphone-Digital/gateway/pkg/health"
	"github.com/Payphone-Digital/gateway/pkg/pool"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Executor handles request execution with connection pooling, circuit breaker, and health monitoring
type Executor struct {
	pool           *pool.ConnectionPool
	circuitBreaker *circuit.BreakerRegistry
	healthMonitor  *health.Monitor
	logger         *zap.Logger
	mu             sync.RWMutex
}

// ExecutorConfig holds executor configuration
type ExecutorConfig struct {
	PoolConfig     pool.PoolConfig
	CircuitConfig  circuit.Config
	HealthInterval time.Duration
}

// DefaultExecutorConfig returns sensible defaults
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		PoolConfig:     pool.DefaultPoolConfig(),
		CircuitConfig:  circuit.DefaultConfig(),
		HealthInterval: 30 * time.Second,
	}
}

// NewExecutor creates a new request executor
func NewExecutor(config ExecutorConfig, logger *zap.Logger) *Executor {
	if logger == nil {
		logger = zap.NewNop()
	}

	executor := &Executor{
		pool:           pool.NewConnectionPool(config.PoolConfig, logger),
		circuitBreaker: circuit.NewBreakerRegistry(config.CircuitConfig, logger),
		healthMonitor:  health.NewMonitor(config.HealthInterval, logger),
		logger:         logger,
	}

	// Start health monitor
	executor.healthMonitor.Start()

	return executor
}

// ExecuteRequest executes a request based on protocol with full resilience
func (e *Executor) ExecuteRequest(ctx context.Context, config *dto.APIConfigResponse, c *gin.Context) ([]byte, int, error) {
	address := config.URLConfig.URL
	protocol := config.Protocol

	e.logger.Info("Executing request",
		zap.String("slug", config.Path),
		zap.String("protocol", protocol),
		zap.String("address", address),
		zap.String("method", config.Method),
	)

	// Get circuit breaker for this backend
	breaker := e.circuitBreaker.GetOrCreate(address)

	// Check circuit breaker
	if err := breaker.Allow(); err != nil {
		e.logger.Warn("Circuit breaker blocked request",
			zap.String("address", address),
			zap.String("state", breaker.State().String()),
			zap.Error(err),
		)
		return nil, http.StatusServiceUnavailable, fmt.Errorf("service unavailable: %w", err)
	}

	// Execute based on protocol
	var body []byte
	var statusCode int
	var err error

	switch protocol {
	case "grpc":
		body, statusCode, err = e.executeGRPC(ctx, config, c)
	case "http", "":
		body, statusCode, err = e.executeHTTP(ctx, config, c)
	default:
		err = fmt.Errorf("unsupported protocol: %s", protocol)
		statusCode = http.StatusBadRequest
	}

	// Record result in circuit breaker
	if err != nil || statusCode >= 500 {
		breaker.Record(fmt.Errorf("request failed: %v, status: %d", err, statusCode))
		e.pool.RecordFailure(address, err)
	} else {
		breaker.Record(nil)
		e.pool.RecordSuccess(address)
	}

	return body, statusCode, err
}

// executeHTTP executes an HTTP request
func (e *Executor) executeHTTP(ctx context.Context, config *dto.APIConfigResponse, c *gin.Context) ([]byte, int, error) {
	address := config.URLConfig.URL
	tlsEnabled := config.URLConfig.TLSEnabled

	// Get HTTP client from pool
	client := e.pool.GetHTTPClient(address, tlsEnabled)

	// Build request config
	apiConfig := ConvertToAPIResponseConfig(config)
	requestConfig := apiConfig.BuildAPIRequestConfig(c)

	// Apply Upstream Authentication
	if config.URLConfig.AuthType != "" && config.URLConfig.AuthType != "none" {
		applyUpstreamAuth(&requestConfig, config.URLConfig)
	}

	e.logger.Debug("Executing HTTP request",
		zap.String("method", requestConfig.Method),
		zap.String("url", requestConfig.URL),
		zap.Int("timeout", requestConfig.Timeout),
	)

	// Execute with custom client
	return e.doHTTPRequest(ctx, client, requestConfig)
}

// doHTTPRequest performs the actual HTTP request
func (e *Executor) doHTTPRequest(ctx context.Context, client *http.Client, config APIRequestConfig) ([]byte, int, error) {
	return DoRequestSafeWithRetry(ctx, config)
}

// executeGRPC executes a gRPC request
func (e *Executor) executeGRPC(ctx context.Context, config *dto.APIConfigResponse, c *gin.Context) ([]byte, int, error) {
	address := config.URLConfig.URL
	tlsEnabled := config.URLConfig.TLSEnabled

	// Get gRPC connection from pool
	conn, err := e.pool.GetGRPCConnection(ctx, address, tlsEnabled)
	if err != nil {
		return nil, http.StatusServiceUnavailable, fmt.Errorf("failed to get gRPC connection: %w", err)
	}

	// Build gRPC request config
	vars := make(map[string]Variable)
	for k, v := range config.Variables {
		vars[k] = Variable{
			Value:    getValueString(v.Value),
			Encoding: v.Encoding,
			DataType: DataType(v.DataType),
		}
	}
	grpcConfig := BuildGRPCRequestConfig(*config, vars, c)

	e.logger.Debug("Executing gRPC request",
		zap.String("service", grpcConfig.Service),
		zap.String("method", grpcConfig.Method),
		zap.String("address", grpcConfig.Address),
	)

	// Execute gRPC request
	return e.doGRPCRequest(ctx, conn, grpcConfig)
}

// doGRPCRequest performs the actual gRPC request
func (e *Executor) doGRPCRequest(ctx context.Context, conn *grpc.ClientConn, config GRPCRequestConfig) ([]byte, int, error) {
	return globalGRPCHandler.ExecuteGRPCRequest(ctx, config)
}

// RegisterHealthCheck registers a backend for health checking
func (e *Executor) RegisterHealthCheck(address, protocol, path string) {
	switch protocol {
	case "http":
		client := e.pool.GetHTTPClient(address, false)
		e.healthMonitor.RegisterHTTPChecker(address, path, client)
	case "grpc":
		conn, err := e.pool.GetGRPCConnection(context.Background(), address, false)
		if err == nil {
			e.healthMonitor.RegisterGRPCChecker(address, conn)
		}
	}
}

// IsBackendHealthy checks if a backend is healthy
func (e *Executor) IsBackendHealthy(address string) bool {
	// Check health monitor
	if !e.healthMonitor.IsHealthy(address) {
		return false
	}

	// Check circuit breaker
	if breaker, exists := e.circuitBreaker.Get(address); exists {
		if breaker.IsOpen() {
			return false
		}
	}

	return true
}

// GetStats returns executor statistics
func (e *Executor) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"pool":            e.pool.Stats(),
		"circuit_breaker": e.circuitBreaker.Stats(),
		"health":          e.healthMonitor.GetAllResults(),
	}
}

// Close closes all connections and stops monitoring
func (e *Executor) Close() error {
	e.healthMonitor.Stop()
	return e.pool.CloseAllConnections()
}

// GlobalExecutor is the default executor instance
var (
	globalExecutor     *Executor
	globalExecutorOnce sync.Once
)

// GetGlobalExecutor returns the global executor instance
func GetGlobalExecutor(logger *zap.Logger) *Executor {
	globalExecutorOnce.Do(func() {
		globalExecutor = NewExecutor(DefaultExecutorConfig(), logger)
	})
	return globalExecutor
}

// ExecuteWithResilience is a convenience function for executing requests with full resilience
func ExecuteWithResilience(ctx context.Context, config *dto.APIConfigResponse, c *gin.Context, logger *zap.Logger) ([]byte, int, error) {
	executor := GetGlobalExecutor(logger)
	return executor.ExecuteRequest(ctx, config, c)
}

// ResponseWrapper wraps response with metadata
type ResponseWrapper struct {
	Success  bool              `json:"success"`
	Data     interface{}       `json:"data,omitempty"`
	Error    string            `json:"error,omitempty"`
	Metadata *ResponseMetadata `json:"metadata,omitempty"`
}

// ResponseMetadata contains response metadata
type ResponseMetadata struct {
	Slug       string `json:"slug"`
	Protocol   string `json:"protocol"`
	Backend    string `json:"backend"`
	Latency    string `json:"latency"`
	FromCache  bool   `json:"from_cache"`
	StatusCode int    `json:"status_code"`
}

// WrapResponse wraps a raw response with metadata
func WrapResponse(body []byte, statusCode int, config *dto.APIConfigResponse, latency time.Duration, fromCache bool) *ResponseWrapper {
	wrapper := &ResponseWrapper{
		Success: statusCode >= 200 && statusCode < 300,
		Metadata: &ResponseMetadata{
			Slug:       config.Path,
			Protocol:   config.Protocol,
			Backend:    config.URLConfig.URL,
			Latency:    latency.String(),
			FromCache:  fromCache,
			StatusCode: statusCode,
		},
	}

	if wrapper.Success {
		var data interface{}
		if err := json.Unmarshal(body, &data); err == nil {
			wrapper.Data = data
		} else {
			wrapper.Data = string(body)
		}
	} else {
		wrapper.Error = string(body)
	}

	return wrapper
}

// applyUpstreamAuth injects authentication credentials into the request config
func applyUpstreamAuth(reqConfig *APIRequestConfig, urlConfig dto.URLConfigResponse) {
	if reqConfig.Headers == nil {
		reqConfig.Headers = make(map[string]string)
	}
	if reqConfig.Query == nil {
		reqConfig.Query = make(map[string]string)
	}

	switch urlConfig.AuthType {
	case "basic":
		if urlConfig.AuthUsername != "" && urlConfig.AuthPassword != "" {
			auth := urlConfig.AuthUsername + ":" + urlConfig.AuthPassword
			reqConfig.Headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
		}
	case "bearer":
		if urlConfig.AuthToken != "" {
			reqConfig.Headers["Authorization"] = "Bearer " + urlConfig.AuthToken
		}
	case "apikey":
		key := urlConfig.AuthKey
		value := urlConfig.AuthValue
		if key != "" && value != "" {
			if urlConfig.AuthAddTo == "query" {
				reqConfig.Query[key] = value
			} else {
				// Default to header
				reqConfig.Headers[key] = value
			}
		}
	}
}
