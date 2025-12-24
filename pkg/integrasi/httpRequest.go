package integrasi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Payphone-Digital/gateway/pkg/logger"
	"go.uber.org/zap"
)

type APIRequestConfig struct {
	Method     string                 // "GET", "POST", etc.
	URL        string                 // Full URL
	Headers    map[string]string      // Custom headers like Authorization
	Query      map[string]string      // Query string
	Body       map[string]interface{} // Body (marshalled to JSON if not nil)
	Timeout    int                    // Timeout in seconds
	MaxRetries int                    // Retry count
	RetryDelay int                    // Retry delay in seconds
	LogFile    string                 // Log file path
	LogLevel   string                 // Log level: info, warn, error
}

// Global gRPC handler
var globalGRPCHandler *GRPCHandler

// Initialize the gRPC handler
func init() {
	globalGRPCHandler = NewGRPCHandler()
}

// The main function to call with context and exponential backoff
func DoRequestSafeWithRetry(ctx context.Context, config APIRequestConfig) ([]byte, int, error) {
	zapLogger := logger.GetLogger().With(
		zap.String("operation", "http_request"),
		zap.String("log_file", config.LogFile),
		zap.String("log_level", config.LogLevel),
	)

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	maxRetries := config.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	retryDelay := time.Duration(config.RetryDelay) * time.Second
	if retryDelay == 0 {
		retryDelay = 1 * time.Second
	}

	var respBody []byte
	var statusCode int
	var lastErr error

	backoff := retryDelay

	for i := 0; i <= maxRetries; i++ {
		// Check context before each try
		if ctx.Err() != nil {
			zapLogger.Warn("Context done before request attempt",
				zap.Error(ctx.Err()),
			)
			return nil, 0, ctx.Err()
		}

		respBody, statusCode, lastErr = doSingleRequest(ctx, config, timeout, zapLogger)
		if lastErr == nil || (statusCode >= 400 && statusCode < 500) {
			break
		}
		zapLogger.Warn("Retry failed",
			zap.Int("attempt", i+1),
			zap.Int("max_retries", maxRetries),
			zap.Error(lastErr),
		)

		// Exponential backoff with context cancellation check
		select {
		case <-time.After(backoff):
			// continue retry
		case <-ctx.Done():
			zapLogger.Warn("Context done during backoff",
				zap.Error(ctx.Err()),
			)
			return nil, 0, ctx.Err()
		}
		backoff *= 2
		if backoff > 30*time.Second {
			backoff = 30 * time.Second // max backoff cap
		}
	}

	if lastErr != nil {
		zapLogger.Error("Final request failed",
			zap.Error(lastErr),
			zap.Int("status_code", statusCode),
		)
	} else {
		zapLogger.Info("Request successful",
			zap.Int("status_code", statusCode),
		)
	}

	return respBody, statusCode, lastErr
}

func doSingleRequest(ctx context.Context, config APIRequestConfig, timeout time.Duration, zapLogger *zap.Logger) ([]byte, int, error) {
	u, err := url.Parse(config.URL)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid URL: %w", err)
	}
	q := u.Query()
	for k, v := range config.Query {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	var bodyReader io.Reader
	if config.Body != nil {
		bodyBytes, err := json.Marshal(config.Body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, config.Method, u.String(), bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}

	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}
	if config.Body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	zapLogger.Info("Making HTTP request",
		zap.String("method", config.Method),
		zap.String("url", u.String()),
	)
	zapLogger.Debug("Request headers",
		zap.Any("headers", config.Headers),
	)
	zapLogger.Debug("Request query",
		zap.Any("query", config.Query),
	)
	if config.Body != nil {
		zapLogger.Debug("Request body",
			zap.Any("body", config.Body),
		)
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		zapLogger.Error("HTTP request failed",
			zap.Error(err),
			zap.String("method", config.Method),
			zap.String("url", u.String()),
		)
		return nil, 0, fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		zapLogger.Error("Failed to read response body",
			zap.Error(err),
		)
		return nil, resp.StatusCode, fmt.Errorf("read body: %w", err)
	}

	zapLogger.Info("HTTP response received",
		zap.Int("status_code", resp.StatusCode),
		zap.String("status_text", http.StatusText(resp.StatusCode)),
	)
	zapLogger.Debug("Response body",
		zap.ByteString("response_data", respData),
	)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		zapLogger.Warn("HTTP error response",
			zap.String("status", resp.Status),
			zap.Int("status_code", resp.StatusCode),
		)
		// return respData, resp.StatusCode, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	return respData, resp.StatusCode, nil
}
