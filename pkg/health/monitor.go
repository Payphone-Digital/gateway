package health

import (
	"context"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// Status represents health check status
type Status int

const (
	StatusUnknown Status = iota
	StatusHealthy
	StatusUnhealthy
	StatusDegraded
)

func (s Status) String() string {
	switch s {
	case StatusHealthy:
		return "HEALTHY"
	case StatusUnhealthy:
		return "UNHEALTHY"
	case StatusDegraded:
		return "DEGRADED"
	default:
		return "UNKNOWN"
	}
}

// CheckResult represents the result of a health check
type CheckResult struct {
	Address      string
	Protocol     string
	Status       Status
	Latency      time.Duration
	LastCheck    time.Time
	LastError    error
	CheckCount   int
	FailureCount int
}

// Checker interface for health checks
type Checker interface {
	Check(ctx context.Context) CheckResult
}

// HTTPChecker checks HTTP endpoint health
type HTTPChecker struct {
	Address string
	Path    string
	Client  *http.Client
}

// Check performs HTTP health check
func (c *HTTPChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		Address:   c.Address,
		Protocol:  "http",
		LastCheck: start,
	}

	url := c.Address + c.Path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		result.Status = StatusUnhealthy
		result.LastError = err
		result.Latency = time.Since(start)
		return result
	}

	resp, err := c.Client.Do(req)
	result.Latency = time.Since(start)

	if err != nil {
		result.Status = StatusUnhealthy
		result.LastError = err
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Status = StatusHealthy
	} else if resp.StatusCode >= 500 {
		result.Status = StatusUnhealthy
	} else {
		result.Status = StatusDegraded
	}

	return result
}

// GRPCChecker checks gRPC endpoint health using standard health protocol
type GRPCChecker struct {
	Address string
	Conn    *grpc.ClientConn
}

// Check performs gRPC health check
func (c *GRPCChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		Address:   c.Address,
		Protocol:  "grpc",
		LastCheck: start,
	}

	if c.Conn == nil {
		result.Status = StatusUnhealthy
		result.Latency = time.Since(start)
		return result
	}

	client := grpc_health_v1.NewHealthClient(c.Conn)
	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	result.Latency = time.Since(start)

	if err != nil {
		result.Status = StatusUnhealthy
		result.LastError = err
		return result
	}

	switch resp.Status {
	case grpc_health_v1.HealthCheckResponse_SERVING:
		result.Status = StatusHealthy
	case grpc_health_v1.HealthCheckResponse_NOT_SERVING:
		result.Status = StatusUnhealthy
	default:
		result.Status = StatusUnknown
	}

	return result
}

// Monitor manages health checks for multiple backends
type Monitor struct {
	mu       sync.RWMutex
	checkers map[string]Checker
	results  map[string]*CheckResult
	interval time.Duration
	logger   *zap.Logger
	ctx      context.Context
	cancel   context.CancelFunc
	running  bool
}

// NewMonitor creates a new health monitor
func NewMonitor(interval time.Duration, logger *zap.Logger) *Monitor {
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Monitor{
		checkers: make(map[string]Checker),
		results:  make(map[string]*CheckResult),
		interval: interval,
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// RegisterHTTPChecker registers an HTTP health checker
func (m *Monitor) RegisterHTTPChecker(address, path string, client *http.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	m.checkers[address] = &HTTPChecker{
		Address: address,
		Path:    path,
		Client:  client,
	}

	m.logger.Info("Registered HTTP health checker",
		zap.String("address", address),
		zap.String("path", path),
	)
}

// RegisterGRPCChecker registers a gRPC health checker
func (m *Monitor) RegisterGRPCChecker(address string, conn *grpc.ClientConn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.checkers[address] = &GRPCChecker{
		Address: address,
		Conn:    conn,
	}

	m.logger.Info("Registered gRPC health checker",
		zap.String("address", address),
	)
}

// Start starts the health monitor
func (m *Monitor) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()

	go m.runChecks()
}

// Stop stops the health monitor
func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.running = false
	m.cancel()
}

// runChecks runs health checks periodically
func (m *Monitor) runChecks() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Run initial checks
	m.checkAll()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkAll()
		}
	}
}

// checkAll checks all registered backends
func (m *Monitor) checkAll() {
	m.mu.RLock()
	checkers := make(map[string]Checker)
	for addr, checker := range m.checkers {
		checkers[addr] = checker
	}
	m.mu.RUnlock()

	for address, checker := range checkers {
		ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
		result := checker.Check(ctx)
		cancel()

		m.mu.Lock()
		if existing, ok := m.results[address]; ok {
			result.CheckCount = existing.CheckCount + 1
			if result.Status == StatusUnhealthy {
				result.FailureCount = existing.FailureCount + 1
			} else {
				result.FailureCount = existing.FailureCount
			}
		} else {
			result.CheckCount = 1
			if result.Status == StatusUnhealthy {
				result.FailureCount = 1
			}
		}
		m.results[address] = &result
		m.mu.Unlock()

		if result.Status != StatusHealthy {
			m.logger.Warn("Health check failed",
				zap.String("address", address),
				zap.String("status", result.Status.String()),
				zap.Duration("latency", result.Latency),
				zap.Error(result.LastError),
			)
		}
	}
}

// IsHealthy checks if a backend is healthy
func (m *Monitor) IsHealthy(address string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if result, ok := m.results[address]; ok {
		return result.Status == StatusHealthy
	}
	return true // Assume healthy if not tracked
}

// GetResult gets health check result for an address
func (m *Monitor) GetResult(address string) (*CheckResult, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result, exists := m.results[address]
	if !exists {
		return nil, false
	}
	resultCopy := *result
	return &resultCopy, true
}

// GetAllResults returns all health check results
func (m *Monitor) GetAllResults() map[string]*CheckResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]*CheckResult)
	for addr, result := range m.results {
		resultCopy := *result
		results[addr] = &resultCopy
	}
	return results
}
