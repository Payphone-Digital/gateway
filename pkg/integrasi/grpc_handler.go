package integrasi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Payphone-Digital/gateway/internal/dto"
	"github.com/Payphone-Digital/gateway/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

// GRPCHandler handles gRPC requests
type GRPCHandler struct {
	connections map[string]*grpc.ClientConn
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler() *GRPCHandler {
	return &GRPCHandler{
		connections: make(map[string]*grpc.ClientConn),
	}
}

// GRPCRequestConfig represents gRPC request configuration
type GRPCRequestConfig struct {
	Address    string                 // gRPC server address
	Service    string                 // gRPC service name
	Method     string                 // gRPC method name
	Headers    map[string]string      // gRPC metadata headers
	Message    map[string]interface{} // Request message data
	Timeout    int                    // Timeout in seconds
	TLSEnabled bool                   // TLS enabled
}

// ExecuteGRPCRequest executes a gRPC request
func (h *GRPCHandler) ExecuteGRPCRequest(ctx context.Context, config GRPCRequestConfig) ([]byte, int, error) {
	zapLogger := logger.GetLogger().With(
		zap.String("operation", "grpc_request"),
		zap.String("service", config.Service),
		zap.String("method", config.Method),
		zap.String("address", config.Address),
	)

	// Set timeout
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	// Create timeout context
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 1. Get or create connection
	conn, err := h.getOrCreateConnection(config.Address, config.TLSEnabled)
	if err != nil {
		zapLogger.Error("Failed to create gRPC connection", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to create connection: %w", err)
	}

	// 2. Prepare metadata
	md := make(metadata.MD)
	for k, v := range config.Headers {
		md.Set(k, v)
	}
	requestCtx = metadata.NewOutgoingContext(requestCtx, md)

	zapLogger.Info("Executing real gRPC request via reflection",
		zap.String("service", config.Service),
		zap.String("method", config.Method),
	)

	// 3. Use Reflection to resolve service and method
	// Create reflection client
	refClient := grpcreflect.NewClient(requestCtx, reflectpb.NewServerReflectionClient(conn))
	defer refClient.Reset() // Clean up

	// Resolve Service
	svcDesc, err := refClient.ResolveService(config.Service)
	if err != nil {
		zapLogger.Error("Failed to resolve service via reflection", zap.Error(err))
		return nil, 404, fmt.Errorf("service not found: %s (ensure reflection is enabled on server)", config.Service)
	}

	// Resolve Method
	methodDesc := svcDesc.FindMethodByName(config.Method)
	if methodDesc == nil {
		zapLogger.Error("Method not found in service", zap.String("method", config.Method))
		return nil, 404, fmt.Errorf("method not found: %s", config.Method)
	}

	// 4. Create Dynamic Message for Input
	inputMsg := dynamic.NewMessage(methodDesc.GetInputType())

	// Convert JSON map to JSON bytes first, then unmarshal into Dynamic Message
	msgBytes, err := json.Marshal(config.Message)
	if err != nil {
		return nil, 400, fmt.Errorf("failed to marshal input message: %w", err)
	}

	err = inputMsg.UnmarshalJSON(msgBytes)
	if err != nil {
		zapLogger.Error("Failed to map JSON to Proto Message", zap.Error(err))
		return nil, 400, fmt.Errorf("invalid input for method %s: %w", config.Method, err)
	}

	// 5. Invoke RPC
	// Create an empty dynamic message for response
	outputMsg := dynamic.NewMessage(methodDesc.GetOutputType())

	// Native Invoke using the full method name: /package.Service/Method
	fullMethodName := fmt.Sprintf("/%s/%s", config.Service, config.Method)

	err = conn.Invoke(requestCtx, fullMethodName, inputMsg, outputMsg)
	if err != nil {
		zapLogger.Error("gRPC execution failed", zap.Error(err))
		return nil, 500, fmt.Errorf("rpc failure: %w", err)
	}

	// 6. Convert Output to JSON
	outputJSON, err := outputMsg.MarshalJSON()
	if err != nil {
		zapLogger.Error("Failed to marshal response", zap.Error(err))
		return nil, 500, fmt.Errorf("failed to process response: %w", err)
	}

	zapLogger.Info("gRPC request completed successfully")
	return outputJSON, 200, nil
}

// getOrCreateConnection gets or creates a gRPC connection
func (h *GRPCHandler) getOrCreateConnection(address string, tlsEnabled bool) (*grpc.ClientConn, error) {
	// Check if connection already exists
	if conn, exists := h.connections[address]; exists {
		return conn, nil
	}

	// Create new connection
	var opts []grpc.DialOption

	if !tlsEnabled {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(nil)))
	}

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", address, err)
	}

	// Cache connection
	h.connections[address] = conn

	return conn, nil
}

// CloseAllConnections closes all gRPC connections
func (h *GRPCHandler) CloseAllConnections() error {
	var lastErr error
	for address, conn := range h.connections {
		if err := conn.Close(); err != nil {
			lastErr = err
		}
		delete(h.connections, address)
	}
	return lastErr
}

// BuildGRPCRequestConfig builds gRPC request config from API config
func BuildGRPCRequestConfig(config dto.APIConfigResponse, variables map[string]Variable, c interface{}) GRPCRequestConfig {
	zapLogger := logger.GetLogger().With(zap.String("slug", config.Path))

	// Extract address and service from URLConfig
	address := config.URLConfig.URL
	service := config.URLConfig.GRPCService
	method := config.Method
	if config.URI != "" {
		method = config.URI
		// Remove leading slash if present
		if len(method) > 0 && method[0] == '/' {
			method = method[1:]
		}
	}

	// Build message from body and variables
	message := make(map[string]interface{})
	
	// 1. Start with configured Static Body (from DB)
	if config.Body != nil {
		for k, v := range config.Body {
			// Apply template rendering to message values
			if strVal, ok := v.(string); ok {
				if containsTemplate(strVal) {
					resolved := resolveTemplateForGRPC(strVal, variables, c)
					message[k] = resolved
				} else {
					message[k] = v
				}
			} else {
				message[k] = v
			}
		}
	}

	// 2. Merge with Incoming Request Body (Dynamic Data)
	// This ensures user input (e.g. registration form) is included
	if ctx, ok := c.(*gin.Context); ok {
		if ctx.Request.Body != nil {
			bodyData, err := ctx.GetRawData() // Reads and restores body
			if err == nil && len(bodyData) > 0 {
				var incomingBody map[string]interface{}
				if err := json.Unmarshal(bodyData, &incomingBody); err == nil {
					// Merge incoming body into message, overwriting defaults
					for k, v := range incomingBody {
						message[k] = v
					}
					zapLogger.Debug("Merged incoming request body", zap.Any("body", incomingBody))
				} else {
					zapLogger.Warn("Failed to unmarshal incoming request body", zap.Error(err))
				}
			} else if err != nil {
				zapLogger.Warn("Failed to read incoming request body", zap.Error(err))
			}
			// Important: GetRawData already restores the body, so we don't need to do it manually
		}
	}

	// Process headers (for gRPC metadata)
	headers := make(map[string]string)
	for k, v := range config.Headers {
		if containsTemplate(v) {
			resolved := resolveTemplateForGRPC(v, variables, c)
			headers[k] = resolved
		} else {
			headers[k] = v
		}
	}

	zapLogger.Info("Built gRPC request config",
		zap.String("address", address),
		zap.String("service", service),
		zap.String("method", method),
		zap.Bool("tls_enabled", config.URLConfig.TLSEnabled),
	)

	return GRPCRequestConfig{
		Address:    address,
		Service:    service,
		Method:     method,
		Headers:    headers,
		Message:    message,
		Timeout:    config.Timeout,
		TLSEnabled: config.URLConfig.TLSEnabled,
	}
}

// Helper function to check if string contains template
func containsTemplate(s string) bool {
	return len(s) > 3 && (contains(s, "{{") || contains(s, "%{"))
}

// Helper function to check substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr))))
}

// Helper function to find substring
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// resolveTemplateForGRPC resolves template variables for gRPC
func resolveTemplateForGRPC(text string, variables map[string]Variable, c interface{}) string {
	result := text
	for key, variable := range variables {
		placeholder := "{{" + key + "}}"
		if contains(result, placeholder) {
			result = replaceAll(result, placeholder, variable.Value)
		}
	}
	return result
}

// replaceAll replaces all occurrences of old with new in s
func replaceAll(s, old, new string) string {
	result := s
	for contains(result, old) {
		result = replaceSingle(result, old, new)
	}
	return result
}

// replaceSingle replaces first occurrence of old with new in s
func replaceSingle(s, old, new string) string {
	if old == "" {
		return s
	}
	for i := 0; i <= len(s)-len(old); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}