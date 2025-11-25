package ctxutil

import (
	"context"
	"net/http"
	"time"

	"github.com/surdiana/gateway/internal/constants"
	"github.com/google/uuid"
)

// Re-export ContextKey type for backward compatibility
type ContextKey = constants.ContextKey

// Re-export context keys for backward compatibility
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
)

// RequestContext berisi informasi request yang disimpan di context
type RequestContext struct {
	RequestID     string
	TraceID       string
	CorrelationID string
	ClientIP      string
	UserAgent     string
	UserID        interface{}
	StartTime     time.Time
	Module        string
	Function      string
}

// NewContext membuat context baru dengan request information
func NewContext(ctx context.Context, req *http.Request, module, function string) context.Context {
	// Generate unique IDs
	requestID := uuid.New().String()
	traceID := getOrCreateTraceID(ctx)
	correlationID := getOrCreateCorrelationID(ctx)

	// Get client IP
	clientIP := getClientIP(req)
	userAgent := req.Header.Get("User-Agent")

	// Create request context
	requestCtx := RequestContext{
		RequestID:     requestID,
		TraceID:       traceID,
		CorrelationID: correlationID,
		ClientIP:      clientIP,
		UserAgent:     userAgent,
		StartTime:     time.Now(),
		Module:        module,
		Function:      function,
	}

	// Store all values in context
	ctx = context.WithValue(ctx, RequestIDKey, requestID)
	ctx = context.WithValue(ctx, TraceIDKey, traceID)
	ctx = context.WithValue(ctx, CorrelationIDKey, correlationID)
	ctx = context.WithValue(ctx, ClientIPKey, clientIP)
	ctx = context.WithValue(ctx, UserAgentKey, userAgent)
	ctx = context.WithValue(ctx, StartTimeKey, requestCtx.StartTime)
	ctx = context.WithValue(ctx, ModuleKey, module)
	ctx = context.WithValue(ctx, FunctionKey, function)

	return ctx
}

// WithUserID menambahkan user ID ke context
func WithUserID(ctx context.Context, userID interface{}) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithTimeout membuat context dengan timeout
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// WithDeadline membuat context dengan deadline
func WithDeadline(ctx context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(ctx, deadline)
}

// WithValue menambahkan custom value ke context
func WithValue(ctx context.Context, key ContextKey, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

// Helper functions untuk mengambil values dari context
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

// GetRequestContext mengembalikan semua request context information
func GetRequestContext(ctx context.Context) RequestContext {
	return RequestContext{
		RequestID:     GetRequestID(ctx),
		TraceID:       GetTraceID(ctx),
		CorrelationID: GetCorrelationID(ctx),
		ClientIP:      GetClientIP(ctx),
		UserAgent:     GetUserAgent(ctx),
		UserID:        GetUserID(ctx),
		StartTime:     GetStartTime(ctx),
		Module:        GetModule(ctx),
		Function:      GetFunction(ctx),
	}
}

// GetDuration menghitung duration dari start time
func GetDuration(ctx context.Context) time.Duration {
	startTime := GetStartTime(ctx)
	if !startTime.IsZero() {
		return time.Since(startTime)
	}
	return 0
}

// IsValidContext memeriksa apakah context masih valid
func IsValidContext(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	default:
		return true
	}
}

// GetContextError mengembalikan error dari context jika ada
func GetContextError(ctx context.Context) error {
	return ctx.Err()
}

// Helper functions private
func getOrCreateTraceID(ctx context.Context) string {
	if traceID := GetTraceID(ctx); traceID != "" {
		return traceID
	}
	return uuid.New().String()
}

func getOrCreateCorrelationID(ctx context.Context) string {
	if correlationID := GetCorrelationID(ctx); correlationID != "" {
		return correlationID
	}
	return uuid.New().String()
}

func getClientIP(req *http.Request) string {
	// Try to get real IP from headers first
	if ip := req.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := req.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := req.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	// Fallback to RemoteAddr
	return req.RemoteAddr
}

// ContextToMap mengubah context ke map untuk logging
func ContextToMap(ctx context.Context) map[string]interface{} {
	requestCtx := GetRequestContext(ctx)

	result := map[string]interface{}{
		"request_id":     requestCtx.RequestID,
		"trace_id":       requestCtx.TraceID,
		"correlation_id": requestCtx.CorrelationID,
		"client_ip":      requestCtx.ClientIP,
		"user_agent":     requestCtx.UserAgent,
		"module":         requestCtx.Module,
		"function":       requestCtx.Function,
		"duration":       GetDuration(ctx),
	}

	if requestCtx.UserID != nil {
		result["user_id"] = requestCtx.UserID
	}

	if !requestCtx.StartTime.IsZero() {
		result["start_time"] = requestCtx.StartTime
	}

	return result
}
