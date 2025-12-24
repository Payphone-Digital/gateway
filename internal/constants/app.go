package constants

// Application metadata
const (
	AppName    = "Payment Gateway"
	AppVersion = "2.0.0"
)

// Environment types
const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
)

// Default application settings
const (
	DefaultPort        = "8080"
	DefaultEnvironment = EnvDevelopment
	DefaultTimeout     = 30 // seconds
)

// Cache key prefixes
const (
	CacheKeyPrefix      = "gateway:"
	CacheKeyRoute       = CacheKeyPrefix + "route:"
	CacheKeyUser        = CacheKeyPrefix + "user:"
	CacheKeyAPI         = CacheKeyPrefix + "api:"
	CacheKeyIntegration = CacheKeyPrefix + "integration:"
	CacheKeyConfig      = CacheKeyPrefix + "config:"
	CacheKeySession     = CacheKeyPrefix + "session:"
)

// Cache TTL (in seconds)
const (
	CacheTTLDefault = 300  // 5 minutes
	CacheTTLShort   = 60   // 1 minute
	CacheTTLMedium  = 600  // 10 minutes
	CacheTTLLong    = 3600 // 1 hour
)

// Log levels
const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
	LogLevelFatal = "fatal"
)
