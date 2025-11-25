package integrasi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/surdiana/gateway/internal/constants"
	"github.com/surdiana/gateway/internal/dto"
	"github.com/surdiana/gateway/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Re-export constants for backward compatibility
const (
	TimeFormat     = constants.TimeFormatDefault
	DefaultTimeout = constants.DefaultTimeoutSeconds
	DefaultRetries = constants.DefaultRetriesCount
	DefaultDelay   = constants.DefaultDelaySeconds
	UserLoginKey   = string(constants.CtxKeyUserLogin)
)

// Re-export types for backward compatibility
type DataType = string
type EncodingType = string

const (
	TypeString  DataType = constants.DataTypeString
	TypeNumber  DataType = constants.DataTypeNumber
	TypeBoolean DataType = constants.DataTypeBoolean
	TypeObject  DataType = constants.DataTypeObject
	TypeArray   DataType = constants.DataTypeArray

	EncodingNone      EncodingType = constants.EncodingNone
	EncodingBase64    EncodingType = constants.EncodingBase64
	EncodingBasicAuth EncodingType = constants.EncodingBasicAuth
	EncodingURLEncode EncodingType = constants.EncodingURLEncode
)

type RequestParams struct {
	PathParams   map[string]string
	QueryParams  map[string]string
	BodyParams   map[string]interface{}
	HeaderParams map[string]string
}

// Structures
type Variable struct {
	Value    string   `json:"value"`
	Encoding string   `json:"encoding"`
	DataType DataType `json:"data_type"`
}

type APIResponseConfig struct {
	Slug        string              `json:"slug"`
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	Headers     map[string]string   `json:"headers"`
	QueryParams map[string]string   `json:"query_params"`
	Body        json.RawMessage     `json:"body"`
	MaxRetries  int                 `json:"max_retries"`
	RetryDelay  int                 `json:"retry_delay"`
	Timeout     int                 `json:"timeout"`
	Variables   map[string]Variable `json:"variables"`
	Protocol    string              `json:"protocol"`
}

func getContextVariable(c *gin.Context, key string) (interface{}, bool) {
	if c == nil {
		return nil, false
	}

	switch key {
	case "current_date":
		return time.Now().UTC().Format(TimeFormat), true
	case "current_user":
		if user, exists := c.Get(UserLoginKey); exists {
			return user, true
		}
		return "", true
	}

	value, exists := c.Get(key)
	return value, exists
}

func extractUserVars(c *gin.Context) RequestParams {
	params := RequestParams{
		PathParams:   make(map[string]string),
		QueryParams:  make(map[string]string),
		BodyParams:   make(map[string]interface{}),
		HeaderParams: make(map[string]string),
	}

	// Extract path parameters
	for _, param := range c.Params {
		params.PathParams[param.Key] = param.Value
	}

	// Extract query parameters
	if c.Request.URL.Query() != nil {
		for k, v := range c.Request.URL.Query() {
			if len(v) > 0 {
				params.QueryParams[k] = v[0]
			}
		}
	}

	// Extract headers
	if c.Request.Header != nil {
		for k := range c.Request.Header {
			params.HeaderParams[k] = c.GetHeader(k)
		}
	}

	// Extract and preserve body parameters
	if c.Request.Body != nil {
		bodyData, err := c.GetRawData()
		if err == nil {
			// Restore the body for later use
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyData))

			// Parse the body
			var bodyMap map[string]interface{}
			if err := json.Unmarshal(bodyData, &bodyMap); err == nil {
				params.BodyParams = bodyMap
			}
		}
	}

	return params
}

func convertToDataType(value string, dataType DataType) interface{} {
	switch dataType {
	case TypeBoolean:
		switch strings.ToLower(value) {
		case "true", "1", "yes", "y":
			return true
		case "false", "0", "no", "n", "":
			return false
		}
		if val, err := strconv.ParseBool(value); err == nil {
			return val
		}
		return false

	case TypeNumber:
		if value == "" {
			return 0
		}
		if val, err := strconv.ParseFloat(value, 64); err == nil {
			return val
		}
		if val, err := strconv.Atoi(value); err == nil {
			return val
		}
		return 0

	case TypeObject:
		if value == "" {
			return make(map[string]interface{})
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(value), &obj); err == nil {
			return obj
		}
		return make(map[string]interface{})

	case TypeArray:
		if value == "" {
			return make([]interface{}, 0)
		}
		var arr []interface{}
		if err := json.Unmarshal([]byte(value), &arr); err == nil {
			return arr
		}
		return make([]interface{}, 0)

	default: // TypeString
		return value
	}
}

func resolveTemplate(text string, variables map[string]Variable, c *gin.Context) string {
	re := regexp.MustCompile(`\{\{(.*?)\}\}`)
	return re.ReplaceAllStringFunc(text, func(match string) string {
		key := strings.TrimSpace(re.FindStringSubmatch(match)[1])

		if variable, exists := variables[key]; exists {
			resolved := resolveVariable(variable, variables, c)
			return fmt.Sprintf("%v", resolved)
		}

		if contextVal, exists := getContextVariable(c, key); exists {
			return fmt.Sprintf("%v", contextVal)
		}

		return ""
	})
}

func resolveVariable(v Variable, variables map[string]Variable, c *gin.Context) interface{} {
	resolved := v.Value
	if v.Value != "" {
		resolved = resolveTemplate(v.Value, variables, c)
	}

	// Convert to proper type
	value := convertToDataType(resolved, v.DataType)

	// Apply encoding if needed for strings
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

func resolveBodyInterface(value interface{}, variables map[string]Variable, c *gin.Context) interface{} {
	switch v := value.(type) {
	case string:
		if strings.Contains(v, "{{") && strings.Contains(v, "}}") {
			re := regexp.MustCompile(`^{{(.*?)}}$`)
			if matches := re.FindStringSubmatch(v); len(matches) > 1 {
				varName := strings.TrimSpace(matches[1])
				if variable, exists := variables[varName]; exists {
					resolved := resolveTemplate(v, variables, c)
					return convertToDataType(resolved, variable.DataType)
				}
			}
			return resolveTemplate(v, variables, c)
		}
		return v

	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			resolved := resolveBodyInterface(val, variables, c)
			if resolved != nil {
				result[key] = resolved
			}
		}
		return result

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = resolveBodyInterface(item, variables, c)
		}
		return result

	default:
		return v
	}
}

func (resp APIResponseConfig) BuildAPIRequestConfig(c *gin.Context) APIRequestConfig {
	zapLogger := logger.GetLogger().With(zap.String("slug", resp.Slug))

	userParams := extractUserVars(c)

	// Create variables map with context variables
	contextVars := map[string]Variable{
		"current_date": {
			Value:    time.Now().UTC().Format(TimeFormat),
			Encoding: string(EncodingNone),
			DataType: TypeString,
		},
		"current_user": {
			Value:    c.GetString(UserLoginKey),
			Encoding: string(EncodingNone),
			DataType: TypeString,
		},
	}

	// Inisialisasi map untuk melacak parameter yang didefinisikan
	definedParams := make(map[string]string)

	// Tandai parameter yang didefinisikan di URL (path params)
	re := regexp.MustCompile(`\{\{(.*?)\}\}`)
	matches := re.FindAllStringSubmatch(resp.URL, -1)
	for _, match := range matches {
		if len(match) > 1 {
			definedParams[match[1]] = "path"
		}
	}

	// Tandai parameter yang didefinisikan di QueryParams
	for _, v := range resp.QueryParams {
		matches := re.FindAllStringSubmatch(v, -1)
		for _, match := range matches {
			if len(match) > 1 {
				definedParams[match[1]] = "query"
			}
		}
	}

	// Tandai parameter yang didefinisikan di Headers
	for _, v := range resp.Headers {
		matches := re.FindAllStringSubmatch(v, -1)
		for _, match := range matches {
			if len(match) > 1 {
				definedParams[match[1]] = "header"
			}
		}
	}

	// Merge variables based on their intended destination and definition
	finalVars := resp.Variables
	for key, variable := range finalVars {
		if variable.Value == "" {
			paramType, isDefined := definedParams[key]
			if !isDefined {
				continue
			}

			switch paramType {
			case "path":
				if pathValue, exists := userParams.PathParams[key]; exists {
					finalVars[key] = Variable{
						Value:    pathValue,
						Encoding: variable.Encoding,
						DataType: variable.DataType,
					}
				}
			case "query":
				if queryValue, exists := userParams.QueryParams[key]; exists {
					finalVars[key] = Variable{
						Value:    queryValue,
						Encoding: variable.Encoding,
						DataType: variable.DataType,
					}
				}
			case "header":
				if headerValue, exists := userParams.HeaderParams[key]; exists {
					finalVars[key] = Variable{
						Value:    headerValue,
						Encoding: variable.Encoding,
						DataType: variable.DataType,
					}
				}
			}
		}
	}

	// Merge context variables
	for k, v := range contextVars {
		if _, exists := finalVars[k]; !exists {
			finalVars[k] = v
		}
	}

	finalURL := resolveTemplate(resp.URL, finalVars, c)
	zapLogger.Info("Processing request",
		zap.String("method", resp.Method),
		zap.String("url", finalURL),
	)

	// Process Headers
	finalHeaders := make(map[string]string)
	for k, v := range resp.Headers {
		resolved := resolveTemplate(v, finalVars, c)
		if resolved != "" {
			finalHeaders[k] = resolved
		}
	}
	zapLogger.Info("Headers processed",
		zap.Any("headers", finalHeaders),
	)

	// Process Query Parameters
	finalQuery := make(map[string]string)
	for k, v := range resp.QueryParams {
		resolved := resolveTemplate(v, finalVars, c)
		if resolved != "" {
			finalQuery[k] = resolved
		}
	}
	zapLogger.Info("Query parameters processed",
		zap.Any("query", finalQuery),
	)

	// Process Body with user parameters
	var finalBody map[string]interface{}
	if len(resp.Body) > 0 {
		var rawBody interface{}
		if err := json.Unmarshal(resp.Body, &rawBody); err == nil {
			// If we have user-provided body params, merge them or use them
			if len(userParams.BodyParams) > 0 {
				if mapBody, ok := rawBody.(map[string]interface{}); ok {
					// Merge user params with configured body
					for k, v := range userParams.BodyParams {
						mapBody[k] = v
					}
					rawBody = mapBody
				} else {
					// If configured body is not a map, use user params directly
					rawBody = userParams.BodyParams
				}
			}

			resolved := resolveBodyInterface(rawBody, finalVars, c)
			zapLogger.Info("Request body processed",
				zap.Any("body", resolved),
			)
			if b, err := json.Marshal(resolved); err == nil {
				if err := json.Unmarshal(b, &finalBody); err != nil {
					zapLogger.Error("Failed to unmarshal final body",
						zap.Error(err),
					)
				}
			}
		}
	} else if len(userParams.BodyParams) > 0 {
		// If no configured body but we have user params, use them directly
		resolved := resolveBodyInterface(userParams.BodyParams, finalVars, c)
		zapLogger.Info("User body processed",
			zap.Any("body", resolved),
		)
		if b, err := json.Marshal(resolved); err == nil {
			if err := json.Unmarshal(b, &finalBody); err != nil {
				zapLogger.Error("Failed to unmarshal user body",
					zap.Error(err),
				)
			}
		}
	}

	timeout := resp.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	maxRetries := resp.MaxRetries
	if maxRetries < 0 {
		maxRetries = DefaultRetries
	}

	retryDelay := resp.RetryDelay
	if retryDelay < 0 {
		retryDelay = DefaultDelay
	}

	return APIRequestConfig{
		Method:     resp.Method,
		URL:        finalURL,
		Headers:    finalHeaders,
		Query:      finalQuery,
		Body:       finalBody,
		Timeout:    timeout,
		MaxRetries: maxRetries,
		RetryDelay: retryDelay,
		LogFile:    fmt.Sprintf("logs/%s.log", resp.Slug),
		LogLevel:   "info",
	}
}

func ConvertToAPIResponseConfig(resp *dto.APIConfigResponse) APIResponseConfig {
	vars := make(map[string]Variable)
	for k, v := range resp.Variables {
		vars[k] = Variable{
			Value:    v.Value,
			Encoding: v.Encoding,
			DataType: DataType(v.DataType),
		}
	}

	var bodyRaw json.RawMessage
	if resp.Body != nil {
		if bodyBytes, err := json.Marshal(resp.Body); err == nil {
			bodyRaw = bodyBytes
		}
	}

	return APIResponseConfig{
		Slug:        resp.Slug,
		Method:      resp.Method,
		URL:         resp.URL,
		Headers:     resp.Headers,
		QueryParams: resp.QueryParams,
		Body:        bodyRaw,
		MaxRetries:  resp.MaxRetries,
		RetryDelay:  resp.RetryDelay,
		Timeout:     resp.Timeout,
		Variables:   vars,
		Protocol:    resp.Protocol,
	}
}

// DoRequestWithProtocol supports both HTTP and gRPC protocols
func DoRequestWithProtocol(ctx context.Context, resp *dto.APIConfigResponse, c *gin.Context) ([]byte, int, error) {
	zapLogger := logger.GetLogger().With(zap.String("slug", resp.Slug))

	switch resp.Protocol {
	case "grpc":
		// Handle gRPC request
		zapLogger.Info("Processing gRPC request")

		// Convert DTO variables to internal variables
		vars := make(map[string]Variable)
		for k, v := range resp.Variables {
			vars[k] = Variable{
				Value:    v.Value,
				Encoding: v.Encoding,
				DataType: DataType(v.DataType),
			}
		}
		grpcConfig := BuildGRPCRequestConfig(*resp, vars, c)
		return globalGRPCHandler.ExecuteGRPCRequest(ctx, grpcConfig)

	case "http", "":
		// Handle HTTP request (default)
		zapLogger.Info("Processing HTTP request")

		apiConfig := ConvertToAPIResponseConfig(resp)
		requestConfig := apiConfig.BuildAPIRequestConfig(c)
		return DoRequestSafeWithRetry(ctx, requestConfig)

	default:
		return nil, 400, fmt.Errorf("unsupported protocol: %s", resp.Protocol)
	}
}
