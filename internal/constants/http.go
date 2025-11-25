package constants

// HTTP Header Names
const (
	HeaderContentType    = "Content-Type"
	HeaderAuthorization  = "Authorization"
	HeaderUserAgent      = "User-Agent"
	HeaderXRequestID     = "X-Request-ID"
	HeaderXTraceID       = "X-Trace-ID"
	HeaderXCorrelationID = "X-Correlation-ID"
	HeaderXForwardedFor  = "X-Forwarded-For"
	HeaderXRealIP        = "X-Real-IP"
	HeaderCFConnectingIP = "CF-Connecting-IP"
)

// HTTP Content Types
const (
	ContentTypeJSON      = "application/json"
	ContentTypeForm      = "application/x-www-form-urlencoded"
	ContentTypeXML       = "application/xml"
	ContentTypeText      = "text/plain"
	ContentTypeHTML      = "text/html"
	ContentTypeMultipart = "multipart/form-data"
)

// Common HTTP Error Messages
const (
	MsgUnauthorized       = "Unauthorized access"
	MsgForbidden          = "Access forbidden"
	MsgNotFound           = "Resource not found"
	MsgBadRequest         = "Invalid request"
	MsgInternalError      = "Internal server error"
	MsgServiceUnavailable = "Service temporarily unavailable"
	MsgConflict           = "Resource already exists"
	MsgMethodNotAllowed   = "Method not allowed"
	MsgTimeout            = "Request timeout"
)

// HTTP Success Messages
const (
	MsgCreated = "Resource created successfully"
	MsgUpdated = "Resource updated successfully"
	MsgDeleted = "Resource deleted successfully"
	MsgSuccess = "Operation completed successfully"
)
