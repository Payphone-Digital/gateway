package constants

// HTTP header names
const (
	// Standard headers
	HeaderContentType   = "Content-Type"
	HeaderAuthorization = "Authorization"
	HeaderUserAgent     = "User-Agent"
	HeaderAccept        = "Accept"
	HeaderCacheControl  = "Cache-Control"

	// Request tracking
	HeaderXRequestID     = "X-Request-ID"
	HeaderXTraceID       = "X-Trace-ID"
	HeaderXCorrelationID = "X-Correlation-ID"

	// Client information
	HeaderXForwardedFor  = "X-Forwarded-For"
	HeaderXRealIP        = "X-Real-IP"
	HeaderCFConnectingIP = "CF-Connecting-IP"

	// Custom headers
	HeaderXUserID = "X-User-ID"
	HeaderXApiKey = "X-Api-Key"
)

// HTTP content types
const (
	ContentTypeJSON      = "application/json"
	ContentTypeJSONUTF8  = "application/json; charset=utf-8"
	ContentTypeForm      = "application/x-www-form-urlencoded"
	ContentTypeXML       = "application/xml"
	ContentTypeText      = "text/plain"
	ContentTypeHTML      = "text/html"
	ContentTypeMultipart = "multipart/form-data"
	ContentTypeBinary    = "application/octet-stream"
	ContentTypeProtobuf  = "application/protobuf"
)

// HTTP status code messages
const (
	// 2xx Success
	Msg200OK        = "Request successful"
	Msg201Created   = "Resource created successfully"
	Msg204NoContent = "No content"

	// 4xx Client errors
	Msg400BadRequest          = "Invalid request"
	Msg401Unauthorized        = "Unauthorized access"
	Msg403Forbidden           = "Access forbidden"
	Msg404NotFound            = "Resource not found"
	Msg405MethodNotAllowed    = "Method not allowed"
	Msg409Conflict            = "Resource already exists"
	Msg422UnprocessableEntity = "Validation failed"
	Msg429TooManyRequests     = "Too many requests"

	// 5xx Server errors
	Msg500InternalError      = "Internal server error"
	Msg502BadGateway         = "Bad gateway"
	Msg503ServiceUnavailable = "Service temporarily unavailable"
	Msg504GatewayTimeout     = "Gateway timeout"
)

// Common operation messages
const (
	MsgCreated        = "Resource created successfully"
	MsgUpdated        = "Resource updated successfully"
	MsgDeleted        = "Resource deleted successfully"
	MsgSuccess        = "Operation completed successfully"
	MsgProcessing     = "Request is being processed"
	MsgInvalidRequest = "Invalid request parameters"
)

// Rate limiting
const (
	RateLimitDefault   = 100 // requests per minute
	RateLimitBurst     = 150 // burst capacity
	RateLimitWindowSec = 60  // window in seconds
)
