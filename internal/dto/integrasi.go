package dto

type Variable struct {
	Value    string `json:"value"`
	Encoding string `json:"encoding"`
	DataType string `json:"data_type"`
}

// Start APi Config
type APIConfigRequest struct {
	Slug         string                 `json:"slug" validate:"required"`
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
	IsAdmin      bool                   `json:"is_admin" `
}

type APIConfigResponse struct {
	ID           uint                   `json:"id"`
	Slug         string                 `json:"slug"`
	Protocol     string                 `json:"protocol"` // From URLConfig for backward compatibility
	Method       string                 `json:"method"`   // HTTP method like "GET", "POST" or gRPC method like "GetUser"
	URLConfigID  uint                   `json:"url_config_id"`
	URI          string                 `json:"uri"`
	URL          string                 `json:"url"` // Complete URL = URLConfig.URL + URI (for HTTP only)
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
	IsAdmin      bool                   `json:"is_admin" `
}

// End APi Config

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

	// gRPC specific fields (server-level configuration)
	GRPCService string `json:"grpc_service,omitempty"` // Service name like "UserService"
	ProtoFile   string `json:"proto_file,omitempty"`   // Proto file name like "user.proto"
	TLSEnabled  bool   `json:"tls_enabled"`            // TLS for gRPC connection
}

type URLConfigResponse struct {
	ID        uint   `json:"id"`
	Nama      string `json:"nama"`
	Protocol  string `json:"protocol"`
	URL       string `json:"url"`
	Deskripsi string `json:"deskripsi"`
	IsActive  bool   `json:"is_active"`

	// gRPC specific fields (server-level configuration)
	GRPCService string `json:"grpc_service,omitempty"` // Service name like "UserService"
	ProtoFile   string `json:"proto_file,omitempty"`   // Proto file name like "user.proto"
	TLSEnabled  bool   `json:"tls_enabled"`            // TLS for gRPC connection
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
