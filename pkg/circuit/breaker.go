package circuit

import (
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// State represents circuit breaker state
type State int

const (
	StateClosed   State = iota // Normal operation - requests pass through
	StateOpen                  // Circuit is open - requests fail fast
	StateHalfOpen              // Testing if service recovered
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Errors
var (
	ErrCircuitOpen     = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)

// Config defines circuit breaker configuration
type Config struct {
	Threshold        int           // Failures before opening circuit
	Timeout          time.Duration // Time to wait before half-open
	SuccessThreshold int           // Successes needed to close from half-open
	MaxHalfOpen      int           // Max concurrent requests in half-open
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		Threshold:        5,
		Timeout:          30 * time.Second,
		SuccessThreshold: 3,
		MaxHalfOpen:      3,
	}
}

// Breaker implements the circuit breaker pattern
type Breaker struct {
	mu               sync.RWMutex
	state            State
	failures         int
	successes        int
	halfOpenRequests int
	lastFailure      time.Time
	config           Config
	logger           *zap.Logger
	name             string
}

// NewBreaker creates a new circuit breaker
func NewBreaker(name string, config Config, logger *zap.Logger) *Breaker {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Breaker{
		state:  StateClosed,
		config: config,
		logger: logger,
		name:   name,
	}
}

// Execute wraps a function with circuit breaker logic
func (b *Breaker) Execute(fn func() error) error {
	if err := b.Allow(); err != nil {
		return err
	}

	err := fn()
	b.Record(err)
	return err
}

// Allow checks if a request should be allowed
func (b *Breaker) Allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if timeout has passed
		if time.Since(b.lastFailure) >= b.config.Timeout {
			b.transitionTo(StateHalfOpen)
			b.halfOpenRequests = 1
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		if b.halfOpenRequests >= b.config.MaxHalfOpen {
			return ErrTooManyRequests
		}
		b.halfOpenRequests++
		return nil

	default:
		return nil
	}
}

// Record records the result of a request
func (b *Breaker) Record(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err != nil {
		b.recordFailure()
	} else {
		b.recordSuccess()
	}
}

// recordFailure handles a failure (must hold lock)
func (b *Breaker) recordFailure() {
	b.failures++
	b.successes = 0
	b.lastFailure = time.Now()

	switch b.state {
	case StateClosed:
		if b.failures >= b.config.Threshold {
			b.transitionTo(StateOpen)
		}

	case StateHalfOpen:
		// Single failure in half-open reopens circuit
		b.transitionTo(StateOpen)
	}
}

// recordSuccess handles a success (must hold lock)
func (b *Breaker) recordSuccess() {
	b.failures = 0

	switch b.state {
	case StateHalfOpen:
		b.successes++
		if b.successes >= b.config.SuccessThreshold {
			b.transitionTo(StateClosed)
		}

	case StateClosed:
		b.successes++
	}
}

// transitionTo changes state (must hold lock)
func (b *Breaker) transitionTo(newState State) {
	oldState := b.state
	b.state = newState
	b.halfOpenRequests = 0

	if newState == StateClosed {
		b.failures = 0
		b.successes = 0
	}

	b.logger.Info("Circuit breaker state changed",
		zap.String("name", b.name),
		zap.String("from", oldState.String()),
		zap.String("to", newState.String()),
		zap.Int("failures", b.failures),
	)
}

// State returns current state
func (b *Breaker) State() State {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// IsOpen returns true if circuit is open
func (b *Breaker) IsOpen() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state == StateOpen
}

// Stats returns circuit breaker statistics
func (b *Breaker) Stats() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return map[string]interface{}{
		"name":         b.name,
		"state":        b.state.String(),
		"failures":     b.failures,
		"successes":    b.successes,
		"last_failure": b.lastFailure,
		"threshold":    b.config.Threshold,
		"timeout":      b.config.Timeout.String(),
	}
}

// Reset resets the circuit breaker to closed state
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.state = StateClosed
	b.failures = 0
	b.successes = 0
	b.halfOpenRequests = 0

	b.logger.Info("Circuit breaker reset",
		zap.String("name", b.name),
	)
}

// BreakerRegistry manages multiple circuit breakers
type BreakerRegistry struct {
	mu       sync.RWMutex
	breakers map[string]*Breaker
	config   Config
	logger   *zap.Logger
}

// NewBreakerRegistry creates a new registry
func NewBreakerRegistry(config Config, logger *zap.Logger) *BreakerRegistry {
	return &BreakerRegistry{
		breakers: make(map[string]*Breaker),
		config:   config,
		logger:   logger,
	}
}

// GetOrCreate gets an existing breaker or creates a new one
func (r *BreakerRegistry) GetOrCreate(name string) *Breaker {
	r.mu.RLock()
	breaker, exists := r.breakers[name]
	r.mu.RUnlock()

	if exists {
		return breaker
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double check
	if breaker, exists = r.breakers[name]; exists {
		return breaker
	}

	breaker = NewBreaker(name, r.config, r.logger)
	r.breakers[name] = breaker
	return breaker
}

// Get gets a breaker by name
func (r *BreakerRegistry) Get(name string) (*Breaker, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	breaker, exists := r.breakers[name]
	return breaker, exists
}

// Stats returns stats for all breakers
func (r *BreakerRegistry) Stats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]interface{})
	for name, breaker := range r.breakers {
		stats[name] = breaker.Stats()
	}
	return stats
}

// ResetAll resets all breakers
func (r *BreakerRegistry) ResetAll() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, breaker := range r.breakers {
		breaker.Reset()
	}
}
