package constants

import "time"

// HTTP/gRPC timeouts
const (
	DefaultIntegrationTimeout = 30 * time.Second
	DefaultRequestTimeout     = 30 * time.Second
	DefaultMaxRetries         = 3
	DefaultRetryDelay         = 1 * time.Second
	DefaultConnTimeout        = 5 * time.Second
)

// Legacy integer constants for backward compatibility
const (
	DefaultTimeoutSeconds = 30
	DefaultRetriesCount   = 0
	DefaultDelaySeconds   = 1
)

// Time formats
const (
	TimeFormatDefault  = "2006-01-02 15:04:05"
	TimeFormatISO8601  = time.RFC3339
	TimeFormatDateOnly = "2006-01-02"
	TimeFormatTimeOnly = "15:04:05"
	TimeFormatUnix     = "unix"
)

// Data types for variable processing
const (
	DataTypeString  = "string"
	DataTypeNumber  = "number"
	DataTypeInteger = "integer"
	DataTypeBoolean = "boolean"
	DataTypeObject  = "object"
	DataTypeArray   = "array"
	DataTypeDate    = "date"
	DataTypeNull    = "null"
)

// Encoding types for data transformation
const (
	EncodingNone      = "none"
	EncodingBase64    = "base64"
	EncodingBasicAuth = "basic_auth"
	EncodingURLEncode = "urlencode"
	EncodingJWT       = "jwt"
)

// Protocol types
const (
	ProtocolHTTP  = "http"
	ProtocolHTTPS = "https"
	ProtocolGRPC  = "grpc"
	ProtocolWS    = "ws"
	ProtocolWSS   = "wss"
)

// HTTP methods
const (
	MethodGET     = "GET"
	MethodPOST    = "POST"
	MethodPUT     = "PUT"
	MethodPATCH   = "PATCH"
	MethodDELETE  = "DELETE"
	MethodHEAD    = "HEAD"
	MethodOPTIONS = "OPTIONS"
)

// Special variable keys
const (
	VarKeyCurrentDate   = "current_date"
	VarKeyCurrentUser   = "current_user"
	VarKeyRequestID     = "request_id"
	VarKeyTraceID       = "trace_id"
	VarKeyCorrelationID = "correlation_id"
)
