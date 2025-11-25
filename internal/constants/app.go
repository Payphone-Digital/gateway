package constants

// Application Information
const (
	AppName    = "Management Service"
	AppVersion = "1.0.0"
)

// Environment Types
const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
)

// Default Application Settings
const (
	DefaultPort        = "8080"
	DefaultEnvironment = EnvDevelopment
)

// Cache Key Prefixes
const (
	CacheKeyPrefix      = "mgmt:"
	CacheKeyUser        = CacheKeyPrefix + "user:"
	CacheKeyAPI         = CacheKeyPrefix + "api:"
	CacheKeyIntegration = CacheKeyPrefix + "integration:"
	CacheKeyConfig      = CacheKeyPrefix + "config:"
)

// Log Levels
const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
	LogLevelFatal = "fatal"
)
