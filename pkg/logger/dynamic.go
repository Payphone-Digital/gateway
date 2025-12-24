package logger

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// DynamicLogger adalah logger yang flexible dan dynamic
type DynamicLogger struct {
	logger *zap.Logger
}

// LogBuilder adalah builder pattern untuk membuat log entry yang dynamic
type LogBuilder struct {
	logger     *zap.Logger
	fields     []zap.Field
	level      zapcore.Level
	message    string
	skipCaller int
}

// NewDynamicLogger membuat instance baru dari dynamic logger
func NewDynamicLogger() *DynamicLogger {
	return &DynamicLogger{
		logger: GetLogger(),
	}
}

// Build memulai membuat log entry baru
func (dl *DynamicLogger) Build() *LogBuilder {
	return &LogBuilder{
		logger:     dl.logger,
		fields:     make([]zap.Field, 0),
		level:      zap.InfoLevel,
		skipCaller: 1,
	}
}

// RequestID menambahkan request ID
func (lb *LogBuilder) RequestID(id string) *LogBuilder {
	lb.fields = append(lb.fields, zap.String("request_id", id))
	return lb
}

// ClientIP menambahkan client IP
func (lb *LogBuilder) ClientIP(ip string) *LogBuilder {
	lb.fields = append(lb.fields, zap.String("client_ip", ip))
	return lb
}

// UserAgent menambahkan user agent
func (lb *LogBuilder) UserAgent(ua string) *LogBuilder {
	lb.fields = append(lb.fields, zap.String("user_agent", ua))
	return lb
}

// Method menambahkan HTTP method
func (lb *LogBuilder) Method(method string) *LogBuilder {
	lb.fields = append(lb.fields, zap.String("method", method))
	return lb
}

// Path menambahkan request path
func (lb *LogBuilder) Path(path string) *LogBuilder {
	lb.fields = append(lb.fields, zap.String("path", path))
	return lb
}

// UserID menambahkan user ID
func (lb *LogBuilder) UserID(id interface{}) *LogBuilder {
	switch v := id.(type) {
	case string:
		lb.fields = append(lb.fields, zap.String("user_id", v))
	case int:
		lb.fields = append(lb.fields, zap.Int("user_id", v))
	case int64:
		lb.fields = append(lb.fields, zap.Int64("user_id", v))
	case uint:
		lb.fields = append(lb.fields, zap.Uint("user_id", v))
	default:
		lb.fields = append(lb.fields, zap.Any("user_id", id))
	}
	return lb
}

// Module menambahkan module name
func (lb *LogBuilder) Module(module string) *LogBuilder {
	lb.fields = append(lb.fields, zap.String("module", module))
	return lb
}

// Function menambahkan function name
func (lb *LogBuilder) Function(function string) *LogBuilder {
	lb.fields = append(lb.fields, zap.String("function", function))
	return lb
}

// Duration menambahkan duration
func (lb *LogBuilder) Duration(duration time.Duration) *LogBuilder {
	lb.fields = append(lb.fields, zap.Duration("duration", duration))
	return lb
}

// StatusCode menambahkan HTTP status code
func (lb *LogBuilder) StatusCode(code int) *LogBuilder {
	lb.fields = append(lb.fields, zap.Int("status_code", code))
	return lb
}

// Err menambahkan error
func (lb *LogBuilder) Err(err error) *LogBuilder {
	if err != nil {
		lb.fields = append(lb.fields, zap.Error(err))
	}
	return lb
}

// Field menambahkan field generic
func (lb *LogBuilder) Field(key string, value interface{}) *LogBuilder {
	lb.fields = append(lb.fields, zap.Any(key, value))
	return lb
}

// Fields menambahkan multiple fields
func (lb *LogBuilder) Fields(fields map[string]interface{}) *LogBuilder {
	for k, v := range fields {
		lb.fields = append(lb.fields, zap.Any(k, v))
	}
	return lb
}

// String menambahkan string field
func (lb *LogBuilder) String(key, value string) *LogBuilder {
	lb.fields = append(lb.fields, zap.String(key, value))
	return lb
}

// Int menambahkan int field
func (lb *LogBuilder) Int(key string, value int) *LogBuilder {
	lb.fields = append(lb.fields, zap.Int(key, value))
	return lb
}

// Int64 menambahkan int64 field
func (lb *LogBuilder) Int64(key string, value int64) *LogBuilder {
	lb.fields = append(lb.fields, zap.Int64(key, value))
	return lb
}

// Bool menambahkan bool field
func (lb *LogBuilder) Bool(key string, value bool) *LogBuilder {
	lb.fields = append(lb.fields, zap.Bool(key, value))
	return lb
}

// Any menambahkan any type field
func (lb *LogBuilder) Any(key string, value interface{}) *LogBuilder {
	lb.fields = append(lb.fields, zap.Any(key, value))
	return lb
}

// Float64 menambahkan float64 field
func (lb *LogBuilder) Float64(key string, value float64) *LogBuilder {
	lb.fields = append(lb.fields, zap.Float64(key, value))
	return lb
}

// Level menentukan log level
func (lb *LogBuilder) Level(level zapcore.Level) *LogBuilder {
	lb.level = level
	return lb
}

// Message menentukan pesan log
func (lb *LogBuilder) Message(message string) *LogBuilder {
	lb.message = message
	return lb
}

// SkipCaller mengatur skip caller untuk stack trace
func (lb *LogBuilder) SkipCaller(skip int) *LogBuilder {
	lb.skipCaller = skip
	return lb
}

// Info log dengan level INFO
func (lb *LogBuilder) Info() {
	lb.logger.Info(lb.message, lb.fields...)
}

// Warn log dengan level WARN
func (lb *LogBuilder) Warn() {
	lb.logger.Warn(lb.message, lb.fields...)
}

// Error log dengan level ERROR
func (lb *LogBuilder) Error() {
	lb.logger.Error(lb.message, lb.fields...)
}

// Debug log dengan level DEBUG
func (lb *LogBuilder) Debug() {
	lb.logger.Debug(lb.message, lb.fields...)
}

// Log log sesuai level yang sudah diset
func (lb *LogBuilder) Log() {
	switch lb.level {
	case zapcore.DebugLevel:
		lb.logger.Debug(lb.message, lb.fields...)
	case zapcore.InfoLevel:
		lb.logger.Info(lb.message, lb.fields...)
	case zapcore.WarnLevel:
		lb.logger.Warn(lb.message, lb.fields...)
	case zapcore.ErrorLevel:
		lb.logger.Error(lb.message, lb.fields...)
	default:
		lb.logger.Info(lb.message, lb.fields...)
	}
}

// Global instance
var dynamicLogger *DynamicLogger

func init() {
	dynamicLogger = NewDynamicLogger()
}

// GetDynamicLogger mengembalikan global dynamic logger instance
func GetDynamicLogger() *DynamicLogger {
	return dynamicLogger
}

// Helper functions untuk kemudahan penggunaan

// Info membuat log INFO cepat
func Info(message string) *LogBuilder {
	return GetDynamicLogger().Build().Message(message).Level(zapcore.InfoLevel)
}

// Warn membuat log WARN cepat
func Warn(message string) *LogBuilder {
	return GetDynamicLogger().Build().Message(message).Level(zapcore.WarnLevel)
}

// Error membuat log ERROR cepat
func Error(message string) *LogBuilder {
	return GetDynamicLogger().Build().Message(message).Level(zapcore.ErrorLevel)
}

// Debug membuat log DEBUG cepat
func Debug(message string) *LogBuilder {
	return GetDynamicLogger().Build().Message(message).Level(zapcore.DebugLevel)
}

// Log membuat log dengan level default (INFO)
func Log(message string) *LogBuilder {
	return GetDynamicLogger().Build().Message(message)
}
