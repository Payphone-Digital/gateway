package logger

import (
	"os"
	"path/filepath"

	"github.com/surdiana/gateway/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger *zap.Logger
	Sugar  *zap.SugaredLogger
)

// InitLogger initializes Zap logger with configuration
func InitLogger(cfg *config.Config) error {
	var err error

	// Create logs directory if it doesn't exist
	logsPath := getEnv("LOGS_PATH", "./logs")
	if err = os.MkdirAll(logsPath, 0755); err != nil {
		return err
	}

	// Configure log level based on environment
	var zapLevel zapcore.Level
	switch cfg.App.Environment {
	case "production":
		zapLevel = zapcore.InfoLevel
	case "staging":
		zapLevel = zapcore.DebugLevel
	default:
		zapLevel = zapcore.DebugLevel
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create paths for log files
	infoLogPath := filepath.Join(logsPath, "info.log")
	errorLogPath := filepath.Join(logsPath, "error.log")
	debugLogPath := filepath.Join(logsPath, "debug.log")

	// Create file handles
	infoFile, err := os.OpenFile(infoLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	errorFile, err := os.OpenFile(errorLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		infoFile.Close()
		return err
	}

	debugFile, err := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		infoFile.Close()
		errorFile.Close()
		return err
	}

	// Create multi-writer for different log levels
	infoWriter := zapcore.AddSync(infoFile)
	errorWriter := zapcore.AddSync(errorFile)
	debugWriter := zapcore.AddSync(debugFile)

	// Create cores for different log levels
	infoCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(infoWriter, zapcore.AddSync(os.Stdout)),
		zapLevel,
	)

	errorCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(errorWriter, zapcore.AddSync(os.Stderr)),
		zapcore.ErrorLevel,
	)

	debugCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(debugWriter),
		zapcore.DebugLevel,
	)

	// In production, use console encoder for better readability
	if cfg.App.Environment == "production" {
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

		infoCore = zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.NewMultiWriteSyncer(infoWriter, zapcore.AddSync(os.Stdout)),
			zapcore.InfoLevel,
		)

		errorCore = zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.NewMultiWriteSyncer(errorWriter, zapcore.AddSync(os.Stderr)),
			zapcore.ErrorLevel,
		)
	}

	// Combine cores
	core := zapcore.NewTee(infoCore, errorCore, debugCore)

	// Create logger with caller information
	Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	Sugar = Logger.Sugar()

	return nil
}

// GetLogger returns the structured logger
func GetLogger() *zap.Logger {
	return Logger
}

// GetSugarLogger returns the sugared logger
func GetSugarLogger() *zap.SugaredLogger {
	return Sugar
}

// Sync syncs all logs (call this before application exits)
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

// WithFields adds structured fields to the logger
func WithFields(fields ...zap.Field) *zap.Logger {
	return Logger.With(fields...)
}

// WithSugarFields adds fields to the sugared logger
func WithSugarFields(args ...interface{}) *zap.SugaredLogger {
	return Sugar.With(args...)
}

// LogRequest logs HTTP request information
func LogRequest(method, path string, statusCode int, duration int64, clientIP string, userAgent string) {
	Logger.Info("HTTP Request",
		zap.String("method", method),
		zap.String("path", path),
		zap.Int("status_code", statusCode),
		zap.Int64("duration_ms", duration),
		zap.String("client_ip", clientIP),
		zap.String("user_agent", userAgent),
	)
}

// LogError logs error with stack trace
func LogError(err error, message string, fields ...zap.Field) {
	allFields := append([]zap.Field{
		zap.Error(err),
	}, fields...)

	Logger.Error(message, allFields...)
}

// LogPanic logs panic and recovers
func LogPanic(recovered interface{}) {
	Logger.Error("Panic recovered",
		zap.Any("panic", recovered),
		zap.Stack("stack"),
	)
}

// LogDatabase logs database operations
func LogDatabase(operation, table string, duration int64, fields ...zap.Field) {
	allFields := append([]zap.Field{
		zap.String("operation", operation),
		zap.String("table", table),
		zap.Int64("duration_ms", duration),
	}, fields...)

	Logger.Debug("Database operation", allFields...)
}

// LogAuth logs authentication events
func LogAuth(userID, action string, success bool, fields ...zap.Field) {
	allFields := append([]zap.Field{
		zap.String("user_id", userID),
		zap.String("action", action),
		zap.Bool("success", success),
	}, fields...)

	if success {
		Logger.Info("Authentication success", allFields...)
	} else {
		Logger.Warn("Authentication failure", allFields...)
	}
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}