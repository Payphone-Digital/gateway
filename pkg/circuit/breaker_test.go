package circuit

import (
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewBreaker(t *testing.T) {
	config := DefaultConfig()
	breaker := NewBreaker("test", config, nil)

	if breaker.State() != StateClosed {
		t.Errorf("Expected initial state CLOSED, got %s", breaker.State().String())
	}

	if breaker.IsOpen() {
		t.Error("Expected breaker to not be open initially")
	}
}

func TestBreaker_TransitionToOpen(t *testing.T) {
	config := Config{
		Threshold:        3,
		Timeout:          1 * time.Second,
		SuccessThreshold: 2,
		MaxHalfOpen:      2,
	}
	breaker := NewBreaker("test", config, zap.NewNop())

	// Record failures until threshold
	for i := 0; i < 3; i++ {
		breaker.Record(errors.New("test error"))
	}

	if breaker.State() != StateOpen {
		t.Errorf("Expected state OPEN after %d failures, got %s", config.Threshold, breaker.State().String())
	}

	// Should reject requests when open
	if err := breaker.Allow(); err != ErrCircuitOpen {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}
}

func TestBreaker_TransitionToHalfOpen(t *testing.T) {
	config := Config{
		Threshold:        2,
		Timeout:          100 * time.Millisecond,
		SuccessThreshold: 2,
		MaxHalfOpen:      2,
	}
	breaker := NewBreaker("test", config, zap.NewNop())

	// Trigger open state
	breaker.Record(errors.New("error 1"))
	breaker.Record(errors.New("error 2"))

	if breaker.State() != StateOpen {
		t.Fatalf("Expected state OPEN, got %s", breaker.State().String())
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should transition to half-open on next Allow()
	if err := breaker.Allow(); err != nil {
		t.Errorf("Expected Allow() to succeed after timeout, got %v", err)
	}

	if breaker.State() != StateHalfOpen {
		t.Errorf("Expected state HALF_OPEN, got %s", breaker.State().String())
	}
}

func TestBreaker_TransitionToClosed(t *testing.T) {
	config := Config{
		Threshold:        2,
		Timeout:          50 * time.Millisecond,
		SuccessThreshold: 2,
		MaxHalfOpen:      5,
	}
	breaker := NewBreaker("test", config, zap.NewNop())

	// Trigger open state
	breaker.Record(errors.New("error"))
	breaker.Record(errors.New("error"))

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Allow to transition to half-open
	breaker.Allow()

	// Record successes to close
	breaker.Record(nil)
	breaker.Record(nil)

	if breaker.State() != StateClosed {
		t.Errorf("Expected state CLOSED after successes, got %s", breaker.State().String())
	}
}

func TestBreaker_Execute(t *testing.T) {
	config := DefaultConfig()
	breaker := NewBreaker("test", config, nil)

	// Execute successful function
	err := breaker.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Execute failing function
	testErr := errors.New("test failure")
	err = breaker.Execute(func() error {
		return testErr
	})
	if err != testErr {
		t.Errorf("Expected test error, got %v", err)
	}
}

func TestBreaker_Reset(t *testing.T) {
	config := Config{Threshold: 1, Timeout: time.Hour}
	breaker := NewBreaker("test", config, nil)

	// Trigger open
	breaker.Record(errors.New("error"))

	if breaker.State() != StateOpen {
		t.Fatal("Expected state OPEN")
	}

	// Reset
	breaker.Reset()

	if breaker.State() != StateClosed {
		t.Errorf("Expected state CLOSED after reset, got %s", breaker.State().String())
	}
}

func TestBreakerRegistry(t *testing.T) {
	config := DefaultConfig()
	registry := NewBreakerRegistry(config, nil)

	// Get or create
	breaker1 := registry.GetOrCreate("backend1")
	breaker2 := registry.GetOrCreate("backend2")
	breaker1Again := registry.GetOrCreate("backend1")

	if breaker1 != breaker1Again {
		t.Error("Expected same breaker instance for same name")
	}

	if breaker1 == breaker2 {
		t.Error("Expected different breakers for different names")
	}

	// Stats
	stats := registry.Stats()
	if len(stats) != 2 {
		t.Errorf("Expected 2 breakers in stats, got %d", len(stats))
	}
}

func TestState_String(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "CLOSED"},
		{StateOpen, "OPEN"},
		{StateHalfOpen, "HALF_OPEN"},
		{State(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("State(%d).String() = %s, want %s", tt.state, got, tt.expected)
		}
	}
}
