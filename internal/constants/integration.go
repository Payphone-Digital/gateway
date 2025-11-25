package constants

import "time"

// Integration Defaults
const (
	DefaultIntegrationTimeout = 30 * time.Second
	DefaultMaxRetries         = 3
	DefaultRetryDelay         = 1 * time.Second
	DefaultRequestTimeout     = 30 * time.Second
)

// Legacy integer constants for backward compatibility
const (
	DefaultTimeoutSeconds = 30
	DefaultRetriesCount   = 0
	DefaultDelaySeconds   = 1
)

// Time Formats
const (
	TimeFormatDefault  = "2006-01-02 15:04:05"
	TimeFormatISO8601  = time.RFC3339
	TimeFormatDateOnly = "2006-01-02"
	TimeFormatTimeOnly = "15:04:05"
)

// Data Types for variable processing
const (
	DataTypeString  = "string"
	DataTypeNumber  = "number"
	DataTypeBoolean = "boolean"
	DataTypeObject  = "object"
	DataTypeArray   = "array"
)

// Encoding Types for data transformation
const (
	EncodingNone      = "none"
	EncodingBase64    = "base64"
	EncodingBasicAuth = "basic_auth"
	EncodingURLEncode = "urlencode"
)

// Protocol Types
const (
	ProtocolHTTP = "http"
	ProtocolGRPC = "grpc"
)

// Special Variable Keys
const (
	VarKeyCurrentDate = "current_date"
	VarKeyCurrentUser = "current_user"
)
