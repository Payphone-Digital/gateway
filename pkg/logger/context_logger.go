package logger

import (
	"context"
	"time"

	ctxutil "github.com/Payphone-Digital/gateway/pkg/context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ContextLogBuilder builder dengan context support
type ContextLogBuilder struct {
	logger     *OptimizedLogger
	ctx        context.Context
	level      zapcore.Level
	fields     []zap.Field
	message    string
	shouldLog  bool
	autoFields bool
}

// WithContext membuat log builder dengan context
func (ol *OptimizedLogger) WithContext(ctx context.Context) *ContextLogBuilder {
	shouldLog := ol.ShouldLog(zapcore.InfoLevel) // Default level, akan diupdate saat Info/Warn/Error dipanggil

	return &ContextLogBuilder{
		logger:     ol,
		ctx:        ctx,
		level:      zapcore.InfoLevel,
		fields:     make([]zap.Field, 0, 12), // Pre-allocate lebih banyak karena ada context fields
		shouldLog:  shouldLog,
		autoFields: true, // Default auto-extract context fields
	}
}

// AutoFields mengatur apakah otomatis mengekstrak fields dari context
func (clb *ContextLogBuilder) AutoFields(auto bool) *ContextLogBuilder {
	clb.autoFields = auto
	return clb
}

// extractContextFields mengekstrak fields dari context
func (clb *ContextLogBuilder) extractContextFields() {
	if !clb.autoFields || clb.ctx == nil {
		return
	}

	// Extract common context fields
	if requestID := ctxutil.GetRequestID(clb.ctx); requestID != "" {
		clb.fields = append(clb.fields, zap.String("request_id", requestID))
	}

	if traceID := ctxutil.GetTraceID(clb.ctx); traceID != "" {
		clb.fields = append(clb.fields, zap.String("trace_id", traceID))
	}

	if correlationID := ctxutil.GetCorrelationID(clb.ctx); correlationID != "" {
		clb.fields = append(clb.fields, zap.String("correlation_id", correlationID))
	}

	if clientIP := ctxutil.GetClientIP(clb.ctx); clientIP != "" {
		clb.fields = append(clb.fields, zap.String("client_ip", clientIP))
	}

	if userAgent := ctxutil.GetUserAgent(clb.ctx); userAgent != "" {
		clb.fields = append(clb.fields, zap.String("user_agent", userAgent))
	}

	if userID := ctxutil.GetUserID(clb.ctx); userID != nil {
		switch v := userID.(type) {
		case string:
			clb.fields = append(clb.fields, zap.String("user_id", v))
		case int:
			clb.fields = append(clb.fields, zap.Int("user_id", v))
		case int64:
			clb.fields = append(clb.fields, zap.Int64("user_id", v))
		case uint:
			clb.fields = append(clb.fields, zap.Uint("user_id", v))
		default:
			clb.fields = append(clb.fields, zap.Any("user_id", userID))
		}
	}

	if module := ctxutil.GetModule(clb.ctx); module != "" {
		clb.fields = append(clb.fields, zap.String("module", module))
	}

	if function := ctxutil.GetFunction(clb.ctx); function != "" {
		clb.fields = append(clb.fields, zap.String("function", function))
	}

	// Add duration if start time exists
	if duration := ctxutil.GetDuration(clb.ctx); duration > 0 {
		clb.fields = append(clb.fields, zap.Duration("duration", duration))
	}
}

// Level methods dengan context
func (clb *ContextLogBuilder) Info(message string) *ContextLogBuilder {
	if !clb.logger.ShouldLog(zapcore.InfoLevel) {
		clb.shouldLog = false
		return clb
	}
	clb.level = zapcore.InfoLevel
	clb.message = message
	clb.extractContextFields()
	return clb
}

func (clb *ContextLogBuilder) Warn(message string) *ContextLogBuilder {
	if !clb.logger.ShouldLog(zapcore.WarnLevel) {
		clb.shouldLog = false
		return clb
	}
	clb.level = zapcore.WarnLevel
	clb.message = message
	clb.extractContextFields()
	return clb
}

func (clb *ContextLogBuilder) Error(message string) *ContextLogBuilder {
	if !clb.logger.ShouldLog(zapcore.ErrorLevel) {
		clb.shouldLog = false
		return clb
	}
	clb.level = zapcore.ErrorLevel
	clb.message = message
	clb.extractContextFields()
	return clb
}

func (clb *ContextLogBuilder) Debug(message string) *ContextLogBuilder {
	if !clb.logger.ShouldLog(zapcore.DebugLevel) {
		clb.shouldLog = false
		return clb
	}
	clb.level = zapcore.DebugLevel
	clb.message = message
	clb.extractContextFields()
	return clb
}

// Field methods (sama seperti OptimizedLogBuilder)
func (clb *ContextLogBuilder) String(key, value string) *ContextLogBuilder {
	if clb.shouldLog {
		clb.fields = append(clb.fields, zap.String(key, value))
	}
	return clb
}

func (clb *ContextLogBuilder) Int(key string, value int) *ContextLogBuilder {
	if clb.shouldLog {
		clb.fields = append(clb.fields, zap.Int(key, value))
	}
	return clb
}

func (clb *ContextLogBuilder) Int64(key string, value int64) *ContextLogBuilder {
	if clb.shouldLog {
		clb.fields = append(clb.fields, zap.Int64(key, value))
	}
	return clb
}

func (clb *ContextLogBuilder) Bool(key string, value bool) *ContextLogBuilder {
	if clb.shouldLog {
		clb.fields = append(clb.fields, zap.Bool(key, value))
	}
	return clb
}

func (clb *ContextLogBuilder) Float64(key string, value float64) *ContextLogBuilder {
	if clb.shouldLog {
		clb.fields = append(clb.fields, zap.Float64(key, value))
	}
	return clb
}

func (clb *ContextLogBuilder) Duration(value time.Duration) *ContextLogBuilder {
	if clb.shouldLog {
		clb.fields = append(clb.fields, zap.Duration("duration", value))
	}
	return clb
}

func (clb *ContextLogBuilder) Err(err error) *ContextLogBuilder {
	if clb.shouldLog && err != nil {
		clb.fields = append(clb.fields, zap.Error(err))
	}
	return clb
}

func (clb *ContextLogBuilder) Any(key string, value interface{}) *ContextLogBuilder {
	if clb.shouldLog {
		clb.fields = append(clb.fields, zap.Any(key, value))
	}
	return clb
}

// Fields menambahkan multiple fields dari map
func (clb *ContextLogBuilder) Fields(fields map[string]interface{}) *ContextLogBuilder {
	if clb.shouldLog {
		for k, v := range fields {
			clb.fields = append(clb.fields, zap.Any(k, v))
		}
	}
	return clb
}

func (clb *ContextLogBuilder) Module(module string) *ContextLogBuilder {
	if clb.shouldLog {
		clb.fields = append(clb.fields, zap.String("module", module))
	}
	return clb
}

func (clb *ContextLogBuilder) Function(function string) *ContextLogBuilder {
	if clb.shouldLog {
		clb.fields = append(clb.fields, zap.String("function", function))
	}
	return clb
}

func (clb *ContextLogBuilder) Method(method string) *ContextLogBuilder {
	if clb.shouldLog {
		clb.fields = append(clb.fields, zap.String("method", method))
	}
	return clb
}

func (clb *ContextLogBuilder) Path(path string) *ContextLogBuilder {
	if clb.shouldLog {
		clb.fields = append(clb.fields, zap.String("path", path))
	}
	return clb
}

func (clb *ContextLogBuilder) StatusCode(code int) *ContextLogBuilder {
	if clb.shouldLog {
		clb.fields = append(clb.fields, zap.Int("status_code", code))
	}
	return clb
}

// Context-specific fields
func (clb *ContextLogBuilder) WithContextValue(key ctxutil.ContextKey, value interface{}) *ContextLogBuilder {
	if clb.shouldLog {
		clb.fields = append(clb.fields, zap.Any(string(key), value))
	}
	return clb
}

// Log menulis log dengan context
func (clb *ContextLogBuilder) Log() {
	if !clb.shouldLog {
		return
	}

	// Check if context is cancelled
	if clb.ctx != nil {
		select {
		case <-clb.ctx.Done():
			// Context is cancelled, don't log
			return
		default:
			// Continue logging
		}
	}

	switch clb.level {
	case zapcore.DebugLevel:
		clb.logger.logger.Debug(clb.message, clb.fields...)
	case zapcore.InfoLevel:
		clb.logger.logger.Info(clb.message, clb.fields...)
	case zapcore.WarnLevel:
		clb.logger.logger.Warn(clb.message, clb.fields...)
	case zapcore.ErrorLevel:
		clb.logger.logger.Error(clb.message, clb.fields...)
	}
}

// Global context logger helper functions
func WithContext(ctx context.Context) *ContextLogBuilder {
	return GetOptimizedLogger().WithContext(ctx)
}

func InfoWithContext(ctx context.Context, message string) *ContextLogBuilder {
	return GetOptimizedLogger().WithContext(ctx).Info(message)
}

func WarnWithContext(ctx context.Context, message string) *ContextLogBuilder {
	return GetOptimizedLogger().WithContext(ctx).Warn(message)
}

func ErrorWithContext(ctx context.Context, message string) *ContextLogBuilder {
	return GetOptimizedLogger().WithContext(ctx).Error(message)
}

func DebugWithContext(ctx context.Context, message string) *ContextLogBuilder {
	return GetOptimizedLogger().WithContext(ctx).Debug(message)
}

// ContextLogger interface untuk dependency injection
type ContextLogger interface {
	WithContext(ctx context.Context) *ContextLogBuilder
	InfoWithContext(ctx context.Context, message string) *ContextLogBuilder
	WarnWithContext(ctx context.Context, message string) *ContextLogBuilder
	ErrorWithContext(ctx context.Context, message string) *ContextLogBuilder
	DebugWithContext(ctx context.Context, message string) *ContextLogBuilder
}

// DefaultContextLogger implementasi default
type DefaultContextLogger struct {
	optimizedLogger *OptimizedLogger
}

func NewDefaultContextLogger() ContextLogger {
	return &DefaultContextLogger{
		optimizedLogger: GetOptimizedLogger(),
	}
}

func (dcl *DefaultContextLogger) WithContext(ctx context.Context) *ContextLogBuilder {
	return dcl.optimizedLogger.WithContext(ctx)
}

func (dcl *DefaultContextLogger) InfoWithContext(ctx context.Context, message string) *ContextLogBuilder {
	return dcl.optimizedLogger.WithContext(ctx).Info(message)
}

func (dcl *DefaultContextLogger) WarnWithContext(ctx context.Context, message string) *ContextLogBuilder {
	return dcl.optimizedLogger.WithContext(ctx).Warn(message)
}

func (dcl *DefaultContextLogger) ErrorWithContext(ctx context.Context, message string) *ContextLogBuilder {
	return dcl.optimizedLogger.WithContext(ctx).Error(message)
}

func (dcl *DefaultContextLogger) DebugWithContext(ctx context.Context, message string) *ContextLogBuilder {
	return dcl.optimizedLogger.WithContext(ctx).Debug(message)
}
