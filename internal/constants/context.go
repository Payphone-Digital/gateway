package constants

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

// Context Keys for request tracking and metadata
const (
	CtxKeyRequestID     ContextKey = "request_id"
	CtxKeyUserID        ContextKey = "user_id"
	CtxKeyClientIP      ContextKey = "client_ip"
	CtxKeyUserAgent     ContextKey = "user_agent"
	CtxKeyTraceID       ContextKey = "trace_id"
	CtxKeyCorrelationID ContextKey = "correlation_id"
	CtxKeyStartTime     ContextKey = "start_time"
	CtxKeyModule        ContextKey = "module"
	CtxKeyFunction      ContextKey = "function"
	CtxKeyUserLogin     ContextKey = "user_login"
)
