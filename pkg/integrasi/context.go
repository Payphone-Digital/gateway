package integrasi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	ctxutil "github.com/surdiana/gateway/pkg/context"
	"github.com/surdiana/gateway/pkg/logger"
	"github.com/gin-gonic/gin"
)

// ContextAwareAPIRequestConfig API request config dengan context support
type ContextAwareAPIRequestConfig struct {
	*APIRequestConfig
	Context context.Context
}

// NewContextAwareAPIRequestConfig membuat request config dengan context
func NewContextAwareAPIRequestConfig(ctx context.Context, config APIRequestConfig) *ContextAwareAPIRequestConfig {
	return &ContextAwareAPIRequestConfig{
		APIRequestConfig: &config,
		Context:          ctx,
	}
}

// DoRequestWithContext melakukan request dengan context
func DoRequestWithContext(ctx context.Context, config APIRequestConfig) ([]byte, int, error) {
	// Add logging dengan context
	logger.InfoWithContext(ctx, "Starting API request").
		String("method", config.Method).
		String("url", config.URL).
		Int("timeout", config.Timeout).
		Int("max_retries", config.MaxRetries).
		Bool("has_body", config.Body != nil).
		Bool("has_headers", len(config.Headers) > 0).
		Bool("has_query", len(config.Query) > 0).
		Log()

	// Check if context is cancelled before starting
	if err := ctx.Err(); err != nil {
		logger.WarnWithContext(ctx, "Context cancelled before API request").
			Err(err).
			Log()
		return nil, 0, err
	}

	// Create context-aware config with timeout
	requestConfig := NewContextAwareAPIRequestConfig(ctx, config)

	// Add timeout to context if not already set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = ctxutil.WithTimeout(ctx, time.Duration(config.Timeout)*time.Second)
		defer cancel()
		requestConfig.Context = ctx
	}

	// Call the original function with context
	body, statusCode, err := DoRequestSafeWithRetry(requestConfig.Context, *requestConfig.APIRequestConfig)

	// Log result dengan context
	if err != nil {
		logger.ErrorWithContext(ctx, "API request failed").
			String("method", config.Method).
			String("url", config.URL).
			Int("status_code", statusCode).
			Err(err).
			Duration(ctxutil.GetDuration(ctx)).
			Log()
	} else {
		logger.InfoWithContext(ctx, "API request completed").
			String("method", config.Method).
			String("url", config.URL).
			Int("status_code", statusCode).
			Int("response_size", len(body)).
			Duration(ctxutil.GetDuration(ctx)).
			Log()
	}

	return body, statusCode, err
}

// ProcessIntegrationWithContext memproses integrasi dengan context
func ProcessIntegrationWithContext(ctx context.Context, resp APIResponseConfig) (*ContextAwareAPIRequestConfig, error) {
	// Add function info to context
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "ProcessIntegration")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "integrasi")

	logger.InfoWithContext(ctx, "Processing integration").
		String("slug", resp.Slug).
		String("method", resp.Method).
		String("url_template", resp.URL).
		Int("timeout", resp.Timeout).
		Int("max_retries", resp.MaxRetries).
		Log()

	// Check context validity
	if err := ctx.Err(); err != nil {
		logger.WarnWithContext(ctx, "Context cancelled during integration processing").
			Err(err).
			Log()
		return nil, err
	}

	// Build API request config with context
	requestConfig := resp.BuildAPIRequestConfigWithContext(ctx)

	// Log final configuration
	logger.DebugWithContext(ctx, "Final API request configuration").
		String("method", requestConfig.Method).
		String("final_url", requestConfig.URL).
		Int("timeout", requestConfig.Timeout).
		Int("max_retries", requestConfig.MaxRetries).
		Bool("has_body", requestConfig.Body != nil).
		Bool("has_headers", len(requestConfig.Headers) > 0).
		Bool("has_query", len(requestConfig.Query) > 0).
		Fields(map[string]interface{}{
			"headers_count": len(requestConfig.Headers),
			"query_count":   len(requestConfig.Query),
			"body_keys":     getBodyKeys(requestConfig.Body),
		}).
		Log()

	return requestConfig, nil
}

// BuildAPIRequestConfigWithContext membuild config dengan context
func (resp APIResponseConfig) BuildAPIRequestConfigWithContext(ctx context.Context) *ContextAwareAPIRequestConfig {
	// Extract request parameters from context
	c := getGinContextFromContext(ctx)
	userParams := extractUserVarsFromContext(c, ctx)

	// Create variables map with context variables
	contextVars := map[string]Variable{
		"current_date": {
			Value:    time.Now().UTC().Format(TimeFormat),
			Encoding: string(EncodingNone),
			DataType: TypeString,
		},
		"current_user": {
			Value:    getStringFromContext(ctx, "current_user"),
			Encoding: string(EncodingNone),
			DataType: TypeString,
		},
	}

	// Add request ID if available
	if requestID := ctxutil.GetRequestID(ctx); requestID != "" {
		contextVars["request_id"] = Variable{
			Value:    requestID,
			Encoding: string(EncodingNone),
			DataType: TypeString,
		}
	}

	// Add correlation ID if available
	if correlationID := ctxutil.GetCorrelationID(ctx); correlationID != "" {
		contextVars["correlation_id"] = Variable{
			Value:    correlationID,
			Encoding: string(EncodingNone),
			DataType: TypeString,
		}
	}

	// Add trace ID if available
	if traceID := ctxutil.GetTraceID(ctx); traceID != "" {
		contextVars["trace_id"] = Variable{
			Value:    traceID,
			Encoding: string(EncodingNone),
			DataType: TypeString,
		}
	}

	// Merge variables
	finalVars := resp.Variables
	for k, v := range contextVars {
		if _, exists := finalVars[k]; !exists {
			finalVars[k] = v
		}
	}

	// Process templates with context
	finalURL := resolveTemplateWithContext(resp.URL, finalVars, ctx)
	finalHeaders := make(map[string]string)
	finalQuery := make(map[string]string)

	// Process headers
	for k, v := range resp.Headers {
		resolved := resolveTemplateWithContext(v, finalVars, ctx)
		if resolved != "" {
			finalHeaders[k] = resolved
		}
	}

	// Process query parameters
	for k, v := range resp.QueryParams {
		resolved := resolveTemplateWithContext(v, finalVars, ctx)
		if resolved != "" {
			finalQuery[k] = resolved
		}
	}

	// Process body
	var finalBody map[string]interface{}
	if len(resp.Body) > 0 {
		var rawBody interface{}
		if err := json.Unmarshal(resp.Body, &rawBody); err == nil {
			resolved := resolveBodyInterfaceWithContext(rawBody, finalVars, ctx)
			if b, err := json.Marshal(resolved); err == nil {
				json.Unmarshal(b, &finalBody)
			}
		}
	} else if len(userParams.BodyParams) > 0 {
		resolved := resolveBodyInterfaceWithContext(userParams.BodyParams, finalVars, ctx)
		if b, err := json.Marshal(resolved); err == nil {
			json.Unmarshal(b, &finalBody)
		}
	}

	// Set timeout with context consideration
	timeout := resp.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	// Check if context has shorter deadline
	if deadline, ok := ctx.Deadline(); ok {
		contextTimeout := time.Until(deadline).Seconds()
		if int(contextTimeout) < timeout {
			timeout = int(contextTimeout)
		}
	}

	apiConfig := &APIRequestConfig{
		Method:     resp.Method,
		URL:        finalURL,
		Headers:    finalHeaders,
		Query:      finalQuery,
		Body:       finalBody,
		Timeout:    timeout,
		MaxRetries: resp.MaxRetries,
		RetryDelay: resp.RetryDelay,
		LogFile:    fmt.Sprintf("logs/%s.log", resp.Slug),
		LogLevel:   "info",
	}

	return NewContextAwareAPIRequestConfig(ctx, *apiConfig)
}

// Helper functions for context-aware operations
func resolveTemplateWithContext(text string, variables map[string]Variable, ctx context.Context) string {
	re := regexp.MustCompile(`\{\{(.*?)\}\}`)
	return re.ReplaceAllStringFunc(text, func(match string) string {
		key := strings.TrimSpace(re.FindStringSubmatch(match)[1])

		// Try variables first
		if variable, exists := variables[key]; exists {
			resolved := resolveVariableWithContext(variable, variables, ctx)
			return fmt.Sprintf("%v", resolved)
		}

		// Try context values
		if contextVal := getContextValueFromContext(ctx, key); contextVal != "" {
			return contextVal
		}

		return ""
	})
}

func resolveVariableWithContext(v Variable, variables map[string]Variable, ctx context.Context) interface{} {
	resolved := v.Value
	if v.Value != "" {
		resolved = resolveTemplateWithContext(v.Value, variables, ctx)
	}

	// Convert to proper type
	value := convertToDataType(resolved, v.DataType)

	// Apply encoding for strings
	if v.DataType == TypeString && value != "" {
		strValue := fmt.Sprintf("%v", value)
		switch v.Encoding {
		case "basic_auth":
			return "Basic " + base64.StdEncoding.EncodeToString([]byte(strValue))
		case "base64":
			return base64.StdEncoding.EncodeToString([]byte(strValue))
		case "urlencode":
			return url.QueryEscape(strValue)
		}
	}

	return value
}

func resolveBodyInterfaceWithContext(value interface{}, variables map[string]Variable, ctx context.Context) interface{} {
	switch v := value.(type) {
	case string:
		if strings.Contains(v, "{{") && strings.Contains(v, "}}") {
			return resolveTemplateWithContext(v, variables, ctx)
		}
		return v

	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			resolved := resolveBodyInterfaceWithContext(val, variables, ctx)
			if resolved != nil {
				result[key] = resolved
			}
		}
		return result

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = resolveBodyInterfaceWithContext(item, variables, ctx)
		}
		return result

	default:
		return v
	}
}

// Helper functions to extract information from context
func getGinContextFromContext(ctx context.Context) *gin.Context {
	// This is a simplified approach - in real implementation,
	// you might need to pass gin.Context through the request chain
	return nil
}

func extractUserVarsFromContext(c *gin.Context, ctx context.Context) RequestParams {
	params := RequestParams{
		PathParams:   make(map[string]string),
		QueryParams:  make(map[string]string),
		BodyParams:   make(map[string]interface{}),
		HeaderParams: make(map[string]string),
	}

	// Extract from context
	if clientIP := ctxutil.GetClientIP(ctx); clientIP != "" {
		params.HeaderParams["client_ip"] = clientIP
	}

	if userAgent := ctxutil.GetUserAgent(ctx); userAgent != "" {
		params.HeaderParams["user_agent"] = userAgent
	}

	if requestID := ctxutil.GetRequestID(ctx); requestID != "" {
		params.HeaderParams["request_id"] = requestID
	}

	return params
}

func getContextValueFromContext(ctx context.Context, key string) string {
	switch key {
	case "current_date":
		return time.Now().UTC().Format(TimeFormat)
	case "current_user":
		if userID := ctxutil.GetUserID(ctx); userID != nil {
			return fmt.Sprintf("%v", userID)
		}
	case "client_ip":
		return ctxutil.GetClientIP(ctx)
	case "user_agent":
		return ctxutil.GetUserAgent(ctx)
	case "request_id":
		return ctxutil.GetRequestID(ctx)
	case "trace_id":
		return ctxutil.GetTraceID(ctx)
	case "correlation_id":
		return ctxutil.GetCorrelationID(ctx)
	}

	// Try to get from context value directly
	if val := ctx.Value(key); val != nil {
		return fmt.Sprintf("%v", val)
	}

	return ""
}

func getStringFromContext(ctx context.Context, key string) string {
	if val := ctx.Value(key); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", val)
	}
	return ""
}

func getBodyKeys(body map[string]interface{}) []string {
	if body == nil {
		return []string{}
	}

	keys := make([]string, 0, len(body))
	for k := range body {
		keys = append(keys, k)
	}
	return keys
}
