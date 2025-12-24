package ctxutil

import (
	"context"
	"time"

	"github.com/Payphone-Digital/gateway/internal/constants"
)

// Re-export ContextKey type
type ContextKey = constants.ContextKey

// Re-export context keys
const (
	RequestIDKey     = constants.CtxKeyRequestID
	UserIDKey        = constants.CtxKeyUserID
	ClientIPKey      = constants.CtxKeyClientIP
	UserAgentKey     = constants.CtxKeyUserAgent
	TraceIDKey       = constants.CtxKeyTraceID
	CorrelationIDKey = constants.CtxKeyCorrelationID
	StartTimeKey     = constants.CtxKeyStartTime
	ModuleKey        = constants.CtxKeyModule
	FunctionKey      = constants.CtxKeyFunction
	UserLoginKey     = constants.CtxKeyUserLogin
)

// WithValue adds a value to context
func WithValue(ctx context.Context, key ContextKey, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID interface{}) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithTimeout creates context with timeout
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// WithDeadline creates context with deadline
func WithDeadline(ctx context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(ctx, deadline)
}

// Getter functions
func GetRequestID(ctx context.Context) string {
	if val, ok := ctx.Value(RequestIDKey).(string); ok {
		return val
	}
	return ""
}

func GetTraceID(ctx context.Context) string {
	if val, ok := ctx.Value(TraceIDKey).(string); ok {
		return val
	}
	return ""
}

func GetCorrelationID(ctx context.Context) string {
	if val, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return val
	}
	return ""
}

func GetClientIP(ctx context.Context) string {
	if val, ok := ctx.Value(ClientIPKey).(string); ok {
		return val
	}
	return ""
}

func GetUserAgent(ctx context.Context) string {
	if val, ok := ctx.Value(UserAgentKey).(string); ok {
		return val
	}
	return ""
}

func GetUserID(ctx context.Context) interface{} {
	return ctx.Value(UserIDKey)
}

func GetUserIDUint(ctx context.Context) (uint, bool) {
	if val, ok := ctx.Value(UserIDKey).(uint); ok {
		return val, true
	}
	return 0, false
}

func GetUserIDString(ctx context.Context) (string, bool) {
	if val, ok := ctx.Value(UserIDKey).(string); ok {
		return val, true
	}
	return "", false
}

func GetStartTime(ctx context.Context) time.Time {
	if val, ok := ctx.Value(StartTimeKey).(time.Time); ok {
		return val
	}
	return time.Time{}
}

func GetModule(ctx context.Context) string {
	if val, ok := ctx.Value(ModuleKey).(string); ok {
		return val
	}
	return ""
}

func GetFunction(ctx context.Context) string {
	if val, ok := ctx.Value(FunctionKey).(string); ok {
		return val
	}
	return ""
}

func GetUserLogin(ctx context.Context) string {
	if val, ok := ctx.Value(UserLoginKey).(string); ok {
		return val
	}
	return ""
}

// GetDuration calculates duration from start time
func GetDuration(ctx context.Context) time.Duration {
	startTime := GetStartTime(ctx)
	if !startTime.IsZero() {
		return time.Since(startTime)
	}
	return 0
}

// IsValidContext checks if context is still valid
func IsValidContext(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	default:
		return true
	}
}

// GetContextError returns error from context if any
func GetContextError(ctx context.Context) error {
	return ctx.Err()
}

// NewContext creates a new context with request tracking information
// Simplified version - only adds essential tracking IDs
func NewContext(ctx context.Context) context.Context {
	// This is a simplified stub - actual implementation should be in middleware
	// that has access to http.Request
	if ctx == nil {
		ctx = context.Background()
	}

	// Set start time if not already set
	if GetStartTime(ctx).IsZero() {
		ctx = context.WithValue(ctx, StartTimeKey, time.Now())
	}

	return ctx
}

// NewContextWithRequest creates context with HTTP request information
// This maintains backward compatibility with existing handler code
func NewContextWithRequest(ctx context.Context, req interface{}, module, function string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	// Add module and function
	ctx = context.WithValue(ctx, ModuleKey, module)
	ctx = context.WithValue(ctx, FunctionKey, function)

	// Set start time if not already set
	if GetStartTime(ctx).IsZero() {
		ctx = context.WithValue(ctx, StartTimeKey, time.Now())
	}

	return ctx
}

// ContextToMap converts context to map for logging
func ContextToMap(ctx context.Context) map[string]interface{} {
	result := make(map[string]interface{})

	if requestID := GetRequestID(ctx); requestID != "" {
		result["request_id"] = requestID
	}
	if traceID := GetTraceID(ctx); traceID != "" {
		result["trace_id"] = traceID
	}
	if correlationID := GetCorrelationID(ctx); correlationID != "" {
		result["correlation_id"] = correlationID
	}
	if clientIP := GetClientIP(ctx); clientIP != "" {
		result["client_ip"] = clientIP
	}
	if userAgent := GetUserAgent(ctx); userAgent != "" {
		result["user_agent"] = userAgent
	}
	if module := GetModule(ctx); module != "" {
		result["module"] = module
	}
	if function := GetFunction(ctx); function != "" {
		result["function"] = function
	}
	if userID := GetUserID(ctx); userID != nil {
		result["user_id"] = userID
	}
	if duration := GetDuration(ctx); duration > 0 {
		result["duration"] = duration
	}

	startTime := GetStartTime(ctx)
	if !startTime.IsZero() {
		result["start_time"] = startTime
	}

	return result
}
