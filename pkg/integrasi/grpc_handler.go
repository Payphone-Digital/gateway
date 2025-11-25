package integrasi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/surdiana/gateway/internal/dto"
	"github.com/surdiana/gateway/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
)

// GRPCHandler handles gRPC requests
type GRPCHandler struct {
	connections map[string]*grpc.ClientConn
	protoCache  map[string]protoreflect.MessageDescriptor
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler() *GRPCHandler {
	return &GRPCHandler{
		connections: make(map[string]*grpc.ClientConn),
		protoCache:  make(map[string]protoreflect.MessageDescriptor),
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

	// Get or create connection
	conn, err := h.getOrCreateConnection(config.Address, config.TLSEnabled)
	if err != nil {
		zapLogger.Error("Failed to create gRPC connection", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to create connection: %w", err)
	}

	// Prepare metadata
	md := make(metadata.MD)
	for k, v := range config.Headers {
		md.Set(k, v)
	}

	// Add metadata to context
	requestCtx = metadata.NewOutgoingContext(requestCtx, md)

	zapLogger.Info("Executing gRPC request",
		zap.String("service", config.Service),
		zap.String("method", config.Method),
		zap.Any("message", config.Message),
	)

	// For now, we'll implement a simple JSON-based approach
	// In production, you would want to use proper proto compilation
	result, err := h.executeDynamicGRPC(requestCtx, conn, config)
	if err != nil {
		zapLogger.Error("gRPC request failed", zap.Error(err))
		return nil, 500, fmt.Errorf("grpc error: %w", err)
	}

	zapLogger.Info("gRPC request completed successfully")
	return result, 200, nil
}

// executeDynamicGRPC executes gRPC request dynamically (simplified approach)
func (h *GRPCHandler) executeDynamicGRPC(ctx context.Context, conn *grpc.ClientConn, config GRPCRequestConfig) ([]byte, error) {
	// This is a simplified implementation
	// In production, you would want to:
	// 1. Parse proto files dynamically
	// 2. Generate message types dynamically
	// 3. Use reflection to call methods

	// For now, we'll use a generic approach with dynamic messages
	// Create dynamic message from JSON data
	messageData, err := json.Marshal(config.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create a dynamic message (this is simplified)
	// In production, you'd use proto reflection properly
	dynamicMsg := &structpb.Value{}
	if err := protojson.Unmarshal(messageData, dynamicMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// For demonstration, we'll return the message as JSON
	// In production, this would be actual gRPC call result
	result := map[string]interface{}{
		"service": config.Service,
		"method":  config.Method,
		"message": dynamicMsg,
		"status":  "success",
		"note":    "This is a simplified gRPC implementation. In production, use proper proto compilation.",
	}

	return json.Marshal(result)
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

// CloseConnection closes a gRPC connection
func (h *GRPCHandler) CloseConnection(address string) error {
	if conn, exists := h.connections[address]; exists {
		err := conn.Close()
		delete(h.connections, address)
		return err
	}
	return nil
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
	zapLogger := logger.GetLogger().With(zap.String("slug", config.Slug))

	// Extract address and service from URLConfig
	address := config.URLConfig.URL
	service := config.URLConfig.GRPCService

	// Get method from APIConfig (now contains both HTTP and gRPC methods)
	method := config.Method

	// Build message from body and variables
	message := make(map[string]interface{})
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
		zap.Int("message_fields", len(message)),
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

// Example of how to use the gRPC handler
func ExampleGRPCUsage() {
	handler := NewGRPCHandler()
	defer handler.CloseAllConnections()

	// Example gRPC request
	config := GRPCRequestConfig{
		Address:    "localhost:50051",
		Service:    "user.UserService",
		Method:     "GetUser",
		Headers:    map[string]string{"authorization": "Bearer token123"},
		Message:    map[string]interface{}{"user_id": "12345"},
		Timeout:    5,
		TLSEnabled: false,
	}

	ctx := context.Background()
	result, status, err := handler.ExecuteGRPCRequest(ctx, config)
	if err != nil {
		fmt.Printf("gRPC request failed: %v\n", err)
		return
	}

	fmt.Printf("gRPC request success (status: %d): %s\n", status, string(result))
}

// resolveTemplateForGRPC resolves template variables for gRPC
func resolveTemplateForGRPC(text string, variables map[string]Variable, c interface{}) string {
	// Simple template resolution for gRPC
	// In production, you'd want to use the same template system as HTTP
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
	// Simple implementation - in production use strings.ReplaceAll
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