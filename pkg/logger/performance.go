package logger

import (
	"os"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// PerformanceConfig konfigurasi untuk performa logger
type PerformanceConfig struct {
	EnableAsync     bool          `json:"enable_async"`
	BufferSize      int           `json:"buffer_size"`
	FlushInterval   time.Duration `json:"flush_interval"`
	SamplingRate    float64       `json:"sampling_rate"`
	MinLogLevel     zapcore.Level `json:"min_log_level"`
	EnableSampling  bool          `json:"enable_sampling"`
	MaxLogPerSecond int           `json:"max_log_per_second"`
	EnableRateLimit bool          `json:"enable_rate_limit"`
}

// DefaultPerformanceConfig konfigurasi default
func DefaultPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		EnableAsync:     true,
		BufferSize:      1000,
		FlushInterval:   time.Second,
		SamplingRate:    1.0, // 100% sampling
		MinLogLevel:     zapcore.InfoLevel,
		EnableSampling:  false,
		MaxLogPerSecond: 1000,
		EnableRateLimit: false,
	}
}

// ProductionConfig konfigurasi untuk production
func ProductionConfig() PerformanceConfig {
	return PerformanceConfig{
		EnableAsync:     true,
		BufferSize:      5000,
		FlushInterval:   5 * time.Second,
		SamplingRate:    0.1, // 10% sampling
		MinLogLevel:     zapcore.WarnLevel,
		EnableSampling:  true,
		MaxLogPerSecond: 500,
		EnableRateLimit: true,
	}
}

// DevelopmentConfig konfigurasi untuk development
func DevelopmentConfig() PerformanceConfig {
	return PerformanceConfig{
		EnableAsync:     false, // Sync logging untuk dev
		BufferSize:      100,
		FlushInterval:   100 * time.Millisecond,
		SamplingRate:    1.0, // 100% sampling
		MinLogLevel:     zapcore.DebugLevel,
		EnableSampling:  false,
		MaxLogPerSecond: 10000,
		EnableRateLimit: false,
	}
}

// OptimizedLogger logger dengan performa optimal
type OptimizedLogger struct {
	config      PerformanceConfig
	logger      *zap.Logger
	rateLimiter *RateLimiter
	mu          sync.RWMutex
}

// RateLimiter untuk membatasi jumlah log per detik
type RateLimiter struct {
	maxLogs   int
	current   int
	lastReset time.Time
	mu        sync.Mutex
}

func NewRateLimiter(maxLogs int) *RateLimiter {
	return &RateLimiter{
		maxLogs:   maxLogs,
		lastReset: time.Now(),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if now.Sub(rl.lastReset) >= time.Second {
		rl.current = 0
		rl.lastReset = now
	}

	if rl.current >= rl.maxLogs {
		return false
	}

	rl.current++
	return true
}

// NewOptimizedLogger membuat optimized logger
func NewOptimizedLogger(config PerformanceConfig) (*OptimizedLogger, error) {
	// Build zap config dengan optimasi
	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = zap.NewAtomicLevelAt(config.MinLogLevel)
	zapConfig.OutputPaths = []string{"stdout"}
	zapConfig.ErrorOutputPaths = []string{"stderr"}
	zapConfig.EncoderConfig.TimeKey = "timestamp"
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapConfig.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder

	// Disable stack trace untuk performance
	zapConfig.DisableStacktrace = true

	// Build logger
	zapLogger, err := zapConfig.Build(
		zap.WithCaller(false), // Disable caller untuk performance
	)
	if err != nil {
		return nil, err
	}

	// Apply sampling jika enabled
	if config.EnableSampling {
		zapLogger = zapLogger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewSamplerWithOptions(
				core,
				time.Second,
				int(config.SamplingRate*100), // Sample sekali per N logs
				0,                            // Always log the first N logs
			)
		}))
	}

	optimized := &OptimizedLogger{
		config:      config,
		logger:      zapLogger,
		rateLimiter: NewRateLimiter(config.MaxLogPerSecond),
	}

	return optimized, nil
}

// ShouldLog menentukan apakah log harus ditulis
func (ol *OptimizedLogger) ShouldLog(level zapcore.Level) bool {
	// Check log level
	if level < ol.config.MinLogLevel {
		return false
	}

	// Check rate limiting
	if ol.config.EnableRateLimit && !ol.rateLimiter.Allow() {
		return false
	}

	return true
}

// OptimizedLogBuilder builder dengan performa optimal
type OptimizedLogBuilder struct {
	logger    *OptimizedLogger
	level     zapcore.Level
	fields    []zap.Field
	message   string
	shouldLog bool
}

// Build membuat optimized log builder
func (ol *OptimizedLogger) Build() *OptimizedLogBuilder {
	return &OptimizedLogBuilder{
		logger:    ol,
		level:     zapcore.InfoLevel,
		fields:    make([]zap.Field, 0, 8), // Pre-allocate
		shouldLog: true,
	}
}

// Level methods dengan early return untuk performance
func (olb *OptimizedLogBuilder) Info(message string) *OptimizedLogBuilder {
	if !olb.logger.ShouldLog(zapcore.InfoLevel) {
		olb.shouldLog = false
		return olb
	}
	olb.level = zapcore.InfoLevel
	olb.message = message
	return olb
}

func (olb *OptimizedLogBuilder) Warn(message string) *OptimizedLogBuilder {
	if !olb.logger.ShouldLog(zapcore.WarnLevel) {
		olb.shouldLog = false
		return olb
	}
	olb.level = zapcore.WarnLevel
	olb.message = message
	return olb
}

func (olb *OptimizedLogBuilder) Error(message string) *OptimizedLogBuilder {
	if !olb.logger.ShouldLog(zapcore.ErrorLevel) {
		olb.shouldLog = false
		return olb
	}
	olb.level = zapcore.ErrorLevel
	olb.message = message
	return olb
}

func (olb *OptimizedLogBuilder) Debug(message string) *OptimizedLogBuilder {
	if !olb.logger.ShouldLog(zapcore.DebugLevel) {
		olb.shouldLog = false
		return olb
	}
	olb.level = zapcore.DebugLevel
	olb.message = message
	return olb
}

// Field methods dengan chaining
func (olb *OptimizedLogBuilder) String(key, value string) *OptimizedLogBuilder {
	if olb.shouldLog {
		olb.fields = append(olb.fields, zap.String(key, value))
	}
	return olb
}

func (olb *OptimizedLogBuilder) Int(key string, value int) *OptimizedLogBuilder {
	if olb.shouldLog {
		olb.fields = append(olb.fields, zap.Int(key, value))
	}
	return olb
}

func (olb *OptimizedLogBuilder) Int64(key string, value int64) *OptimizedLogBuilder {
	if olb.shouldLog {
		olb.fields = append(olb.fields, zap.Int64(key, value))
	}
	return olb
}

func (olb *OptimizedLogBuilder) Bool(key string, value bool) *OptimizedLogBuilder {
	if olb.shouldLog {
		olb.fields = append(olb.fields, zap.Bool(key, value))
	}
	return olb
}

func (olb *OptimizedLogBuilder) Float64(key string, value float64) *OptimizedLogBuilder {
	if olb.shouldLog {
		olb.fields = append(olb.fields, zap.Float64(key, value))
	}
	return olb
}

func (olb *OptimizedLogBuilder) Duration(value time.Duration) *OptimizedLogBuilder {
	if olb.shouldLog {
		olb.fields = append(olb.fields, zap.Duration("duration", value))
	}
	return olb
}

func (olb *OptimizedLogBuilder) Err(err error) *OptimizedLogBuilder {
	if olb.shouldLog && err != nil {
		olb.fields = append(olb.fields, zap.Error(err))
	}
	return olb
}

func (olb *OptimizedLogBuilder) Any(key string, value interface{}) *OptimizedLogBuilder {
	if olb.shouldLog {
		olb.fields = append(olb.fields, zap.Any(key, value))
	}
	return olb
}

func (olb *OptimizedLogBuilder) Module(module string) *OptimizedLogBuilder {
	if olb.shouldLog {
		olb.fields = append(olb.fields, zap.String("module", module))
	}
	return olb
}

func (olb *OptimizedLogBuilder) Function(function string) *OptimizedLogBuilder {
	if olb.shouldLog {
		olb.fields = append(olb.fields, zap.String("function", function))
	}
	return olb
}

func (olb *OptimizedLogBuilder) ClientIP(ip string) *OptimizedLogBuilder {
	if olb.shouldLog {
		olb.fields = append(olb.fields, zap.String("client_ip", ip))
	}
	return olb
}

func (olb *OptimizedLogBuilder) UserID(id interface{}) *OptimizedLogBuilder {
	if olb.shouldLog {
		switch v := id.(type) {
		case string:
			olb.fields = append(olb.fields, zap.String("user_id", v))
		case int:
			olb.fields = append(olb.fields, zap.Int("user_id", v))
		case int64:
			olb.fields = append(olb.fields, zap.Int64("user_id", v))
		case uint:
			olb.fields = append(olb.fields, zap.Uint("user_id", v))
		default:
			olb.fields = append(olb.fields, zap.Any("user_id", id))
		}
	}
	return olb
}

func (olb *OptimizedLogBuilder) Method(method string) *OptimizedLogBuilder {
	if olb.shouldLog {
		olb.fields = append(olb.fields, zap.String("method", method))
	}
	return olb
}

func (olb *OptimizedLogBuilder) Path(path string) *OptimizedLogBuilder {
	if olb.shouldLog {
		olb.fields = append(olb.fields, zap.String("path", path))
	}
	return olb
}

// Log menulis log (hanya jika shouldLog true)
func (olb *OptimizedLogBuilder) Log() {
	if !olb.shouldLog {
		return
	}

	switch olb.level {
	case zapcore.DebugLevel:
		olb.logger.logger.Debug(olb.message, olb.fields...)
	case zapcore.InfoLevel:
		olb.logger.logger.Info(olb.message, olb.fields...)
	case zapcore.WarnLevel:
		olb.logger.logger.Warn(olb.message, olb.fields...)
	case zapcore.ErrorLevel:
		olb.logger.logger.Error(olb.message, olb.fields...)
	}
}

// Global optimized logger instance
var optimizedLogger *OptimizedLogger

// InitOptimizedLogger menginisialisasi optimized logger
func InitOptimizedLogger(config PerformanceConfig) error {
	logger, err := NewOptimizedLogger(config)
	if err != nil {
		return err
	}
	optimizedLogger = logger
	return nil
}

// GetOptimizedLogger mengembalikan optimized logger
func GetOptimizedLogger() *OptimizedLogger {
	if optimizedLogger == nil {
		// Fallback ke default config
		config := DefaultPerformanceConfig()
		env := os.Getenv("GO_ENV")
		if env == "production" {
			config = ProductionConfig()
		} else if env == "development" {
			config = DevelopmentConfig()
		}

		logger, _ := NewOptimizedLogger(config)
		optimizedLogger = logger
	}
	return optimizedLogger
}

// Optimized helper functions
func OptInfo(message string) *OptimizedLogBuilder {
	return GetOptimizedLogger().Build().Info(message)
}

func OptWarn(message string) *OptimizedLogBuilder {
	return GetOptimizedLogger().Build().Warn(message)
}

func OptError(message string) *OptimizedLogBuilder {
	return GetOptimizedLogger().Build().Error(message)
}

func OptDebug(message string) *OptimizedLogBuilder {
	return GetOptimizedLogger().Build().Debug(message)
}

// Performance monitoring
func GetLoggerStats() map[string]interface{} {
	if optimizedLogger == nil {
		return map[string]interface{}{"status": "not_initialized"}
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"config": optimizedLogger.config,
		"rate_limiter": map[string]interface{}{
			"current_logs": optimizedLogger.rateLimiter.current,
			"max_logs":     optimizedLogger.rateLimiter.maxLogs,
		},
		"memory": map[string]interface{}{
			"alloc_mb":       m.Alloc / 1024 / 1024,
			"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
			"sys_mb":         m.Sys / 1024 / 1024,
			"num_gc":         m.NumGC,
		},
	}
}
