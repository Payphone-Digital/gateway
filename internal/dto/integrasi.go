package dto

type Variable struct {
	Value              interface{}            `json:"value"`
	Encoding           string                 `json:"encoding"`
	DataType           string                 `json:"data_type"`
	IsRequired         bool                   `json:"is_required"`
	Validations        map[string]interface{} `json:"validations"`
	ValidationMessages map[string]string      `json:"validation_messages"`
	CustomMessage      string                 `json:"custom_message"`
}

// Start API Config (Path Config)
type APIConfigRequest struct {
	Path         string                 `json:"path" validate:"required"`   // Dynamic path like "/users", "/products"
	Method       string                 `json:"method" validate:"required"` // HTTP method like "GET", "POST" or gRPC method like "GetUser"
	URLConfigID  uint                   `json:"url_config_id" validate:"required"`
	URI          string                 `json:"uri"` // Optional untuk HTTP, kosong untuk gRPC
	Headers      map[string]string      `json:"headers"`
	QueryParams  map[string]string      `json:"query_params"`
	Body         map[string]interface{} `json:"body"`
	Variables    map[string]Variable    `json:"variables"`
	MaxRetries   int                    `json:"max_retries" validate:"required"`
	RetryDelay   int                    `json:"retry_delay" validate:"required"`
	Timeout      int                    `json:"timeout" validate:"required"`
	Manipulation string                 `json:"manipulation"`
	Description  string                 `json:"description"`
	IsAdmin      bool                   `json:"is_admin"`

	// Caching
	CacheEnabled bool `json:"cache_enabled"`
	CacheTTL     int  `json:"cache_ttl"`

	// Rate Limiting
	RateLimitEnabled bool `json:"rate_limit_enabled"`
	RateLimit        int  `json:"rate_limit"`
	RateLimitWindow  int  `json:"rate_limit_window"`

	// Priority
	Priority int `json:"priority"`

	// Authentication Configuration
	AuthType         string `json:"auth_type"`                     // none, jwt, basic, apikey, gateway
	AuthRequired     bool   `json:"auth_required"`                 // Whether authentication is required
	AuthGRPCConfigID *uint  `json:"auth_grpc_config_id,omitempty"` // Reference to URLConfig for auth gRPC

	// JWT Configuration
	JWTSecretKey  string `json:"jwt_secret_key,omitempty"`
	JWTIssuer     string `json:"jwt_issuer,omitempty"`
	JWTAudience   string `json:"jwt_audience,omitempty"`
	JWTAlgorithm  string `json:"jwt_algorithm,omitempty"`
	JWTExpiration int    `json:"jwt_expiration,omitempty"`

	// Basic Auth Configuration
	BasicAuthUsers []BasicAuthUser `json:"basic_auth_users,omitempty"`

	// API Key Configuration
	APIKeyHeader   string   `json:"api_key_header,omitempty"`
	APIKeyLocation string   `json:"api_key_location,omitempty"` // header, query
	APIKeys        []APIKey `json:"api_keys,omitempty"`
}

// BasicAuthUser represents a user for basic authentication
type BasicAuthUser struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash,omitempty"`
	Password     string `json:"password,omitempty"` // Plain password for create/update, will be hashed
}

// APIKey represents an API key configuration
type APIKey struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type APIConfigResponse struct {
	ID           uint                   `json:"id"`
	Path         string                 `json:"path"`
	Protocol     string                 `json:"protocol"` // From URLConfig for backward compatibility
	Method       string                 `json:"method"`   // HTTP method like "GET", "POST" or gRPC method like "GetUser"
	URLConfigID  uint                   `json:"url_config_id"`
	URI          string                 `json:"uri"`
	URL          string                 `json:"url"` // Complete URL = URLConfig.URL + Path (for HTTP only)
	URLConfig    URLConfigResponse      `json:"url_config"`
	Headers      map[string]string      `json:"headers"`
	QueryParams  map[string]string      `json:"query_params"`
	Body         map[string]interface{} `json:"body"`
	Variables    map[string]Variable    `json:"variables"`
	MaxRetries   int                    `json:"max_retries"`
	RetryDelay   int                    `json:"retry_delay"`
	Timeout      int                    `json:"timeout"`
	Manipulation string                 `json:"manipulation"`
	Description  string                 `json:"description"`
	IsAdmin      bool                   `json:"is_admin"`

	// Caching
	CacheEnabled bool `json:"cache_enabled"`
	CacheTTL     int  `json:"cache_ttl"`

	// Rate Limiting
	RateLimitEnabled bool `json:"rate_limit_enabled"`
	RateLimit        int  `json:"rate_limit"`
	RateLimitWindow  int  `json:"rate_limit_window"`

	// Priority
	Priority int `json:"priority"`

	// Authentication Configuration
	AuthType         string             `json:"auth_type"`
	AuthRequired     bool               `json:"auth_required"`
	AuthGRPCConfigID *uint              `json:"auth_grpc_config_id,omitempty"`
	AuthGRPCConfig   *URLConfigResponse `json:"auth_grpc_config,omitempty"`

	// JWT Configuration
	JWTSecretKey  string `json:"jwt_secret_key,omitempty"`
	JWTIssuer     string `json:"jwt_issuer,omitempty"`
	JWTAudience   string `json:"jwt_audience,omitempty"`
	JWTAlgorithm  string `json:"jwt_algorithm,omitempty"`
	JWTExpiration int    `json:"jwt_expiration,omitempty"`

	// Basic Auth Configuration
	BasicAuthUsers []BasicAuthUser `json:"basic_auth_users,omitempty"`

	// API Key Configuration
	APIKeyHeader   string   `json:"api_key_header,omitempty"`
	APIKeyLocation string   `json:"api_key_location,omitempty"`
	APIKeys        []APIKey `json:"api_keys,omitempty"`
}

// End API Config (Path Config)

// Start Api Group
type APIGroupRequest struct {
	Slug    string `json:"slug" validate:"required"`
	Name    string `json:"name" validate:"required"`
	IsAdmin bool   `json:"is_admin" `
}

type APIGroupResponse struct {
	ID      uint   `json:"id"`
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	IsAdmin bool   `json:"is_admin" `
}

// End Api Group

// Start Group Step
type APIGroupStepRequest struct {
	GroupID     uint                   `json:"group_id" validate:"required"`
	APIConfigID uint                   `json:"api_config_id" validate:"required"`
	OrderIndex  int                    `json:"order_index" validate:"required"`
	Alias       string                 `json:"alias" validate:"required"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
}

type APIGroupStepResponse struct {
	ID          uint                   `json:"id"`
	GroupID     uint                   `json:"group_id"`
	APIConfigID uint                   `json:"api_config_id"`
	OrderIndex  int                    `json:"order_index"`
	Alias       string                 `json:"alias"`
	Variables   map[string]interface{} `json:"variables"`
	APIConfig   APIConfigResponse      `json:"api_config"`
}

// End Group Step

// Start Group Cron
type APIGroupCronRequest struct {
	Slug     string `json:"slug" validate:"required"`     // slug dari APIGroup
	Schedule string `json:"schedule" validate:"required"` // format cron (mis: "0 9 * * *")
	Enabled  bool   `json:"enabled" `
}
type APIGroupCronResponse struct {
	ID       uint   `json:"id"`
	Slug     string `json:"slug"`     // slug dari APIGroup
	Schedule string `json:"schedule"` // format cron (mis: "0 9 * * *")
	Enabled  bool   `json:"enabled"`
}

// End Group Cron

// Start URL Config
type URLConfigRequest struct {
	Nama      string `json:"nama" validate:"required"`
	Protocol  string `json:"protocol" validate:"required,oneof=http grpc"`
	URL       string `json:"url" validate:"required"`
	Deskripsi string `json:"deskripsi"`
	IsActive  bool   `json:"is_active"`

	// gRPC specific fields
	GRPCService string `json:"grpc_service,omitempty"`
	ProtoFile   string `json:"proto_file,omitempty"`
	TLSEnabled  bool   `json:"tls_enabled"`

	// Connection Pool Settings
	MaxConnections     int `json:"max_connections"`
	MinIdleConnections int `json:"min_idle_connections"`
	ConnectionTimeout  int `json:"connection_timeout"`
	ReadTimeout        int `json:"read_timeout"`
	WriteTimeout       int `json:"write_timeout"`

	// Health Check Settings
	HealthCheckPath     string `json:"health_check_path"`
	HealthCheckInterval int    `json:"health_check_interval"`

	// Circuit Breaker Settings
	CircuitBreakerEnabled   bool   `json:"circuit_breaker_enabled"`
	CircuitBreakerThreshold int    `json:"circuit_breaker_threshold"`
	CircuitBreakerTimeout   int    `json:"circuit_breaker_timeout"`
	RetryOnStatusCodes      string `json:"retry_on_status_codes"`

	// Load Balancing
	LoadBalancingStrategy string `json:"load_balancing_strategy"`

	// Upstream Authentication
	AuthType     string `json:"auth_type"`
	AuthUsername string `json:"auth_username"`
	AuthPassword string `json:"auth_password"`
	AuthToken    string `json:"auth_token"`
	AuthKey      string `json:"auth_key"`
	AuthValue    string `json:"auth_value"`
	AuthAddTo    string `json:"auth_add_to"`
}

type URLConfigResponse struct {
	ID        uint   `json:"id"`
	Nama      string `json:"nama"`
	Protocol  string `json:"protocol"`
	URL       string `json:"url"`
	Deskripsi string `json:"deskripsi"`
	IsActive  bool   `json:"is_active"`

	// gRPC specific fields
	GRPCService string `json:"grpc_service,omitempty"`
	ProtoFile   string `json:"proto_file,omitempty"`
	TLSEnabled  bool   `json:"tls_enabled"`

	// Connection Pool Settings
	MaxConnections     int `json:"max_connections"`
	MinIdleConnections int `json:"min_idle_connections"`
	ConnectionTimeout  int `json:"connection_timeout"`
	ReadTimeout        int `json:"read_timeout"`
	WriteTimeout       int `json:"write_timeout"`

	// Health Check Settings
	HealthCheckPath     string `json:"health_check_path"`
	HealthCheckInterval int    `json:"health_check_interval"`

	// Circuit Breaker Settings
	CircuitBreakerEnabled   bool   `json:"circuit_breaker_enabled"`
	CircuitBreakerThreshold int    `json:"circuit_breaker_threshold"`
	CircuitBreakerTimeout   int    `json:"circuit_breaker_timeout"`
	RetryOnStatusCodes      string `json:"retry_on_status_codes"`

	// Load Balancing
	LoadBalancingStrategy string `json:"load_balancing_strategy"`

	// Upstream Authentication
	AuthType     string `json:"auth_type"`
	AuthUsername string `json:"auth_username,omitempty"`
	AuthPassword string `json:"auth_password,omitempty"`
	AuthToken    string `json:"auth_token,omitempty"`
	AuthKey      string `json:"auth_key,omitempty"`
	AuthValue    string `json:"auth_value,omitempty"`
	AuthAddTo    string `json:"auth_add_to,omitempty"`
}

// End URL Config

// API Config Filter DTO for GET /api/v1/path-config
type APIConfigFilter struct {
	URLConfigID uint   `query:"url_config_id"` // Filter by URL Config ID
	Protocol    string `query:"protocol"`      // Filter by protocol (http/grpc)
	Method      string `query:"method"`        // Filter by HTTP method
	IsAdmin     *bool  `query:"is_admin"`      // Filter by admin status (nil = no filter)
	Status      string `query:"status"`        // Custom status filter
}

// URL Config Filter DTO for GET /api/v1/url-config
type URLConfigFilter struct {
	Protocol string `query:"protocol"`  // Filter by protocol (http/grpc)
	IsActive *bool  `query:"is_active"` // Filter by active status (nil = no filter)
}
