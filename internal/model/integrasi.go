package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type URLConfig struct {
	gorm.Model
	Nama        string  `gorm:"type:varchar(255);not null;index:idx_url_configs_nama" json:"nama" binding:"required"`
	Protocol    string  `gorm:"type:varchar(20);default:'http';index:idx_url_configs_protocol;check:protocol IN ('http', 'grpc')" json:"protocol" binding:"required,oneof=http grpc"`
	URL         string  `gorm:"type:varchar(2048);not null;index:idx_url_configs_url" json:"url" binding:"required"`
	Deskripsi   string  `gorm:"type:text" json:"deskripsi"`
	IsActive    bool    `gorm:"default:true;index:idx_url_configs_is_active" json:"is_active"`
	GRPCService *string `gorm:"type:varchar(100);index:idx_url_configs_grpc_service" json:"grpc_service,omitempty"`
	ProtoFile   *string `gorm:"type:varchar(500)" json:"proto_file,omitempty"`
	TLSEnabled  bool    `gorm:"default:false;index:idx_url_configs_tls_enabled" json:"tls_enabled"`

	// Connection Pool Settings
	MaxConnections     int `gorm:"default:100" json:"max_connections"`
	MinIdleConnections int `gorm:"default:10" json:"min_idle_connections"`
	ConnectionTimeout  int `gorm:"default:5" json:"connection_timeout"` // seconds
	ReadTimeout        int `gorm:"default:30" json:"read_timeout"`      // seconds
	WriteTimeout       int `gorm:"default:30" json:"write_timeout"`     // seconds

	// Health Check Settings
	HealthCheckPath     string `gorm:"type:varchar(500);default:'/health'" json:"health_check_path"`
	HealthCheckInterval int    `gorm:"default:30" json:"health_check_interval"` // seconds

	// Circuit Breaker Settings
	CircuitBreakerEnabled   bool   `gorm:"default:true" json:"circuit_breaker_enabled"`
	CircuitBreakerThreshold int    `gorm:"default:5" json:"circuit_breaker_threshold"` // failures before open
	CircuitBreakerTimeout   int    `gorm:"default:30" json:"circuit_breaker_timeout"`  // seconds to wait before half-open
	RetryOnStatusCodes      string `gorm:"type:varchar(100);default:'502,503,504'" json:"retry_on_status_codes"`

	// Upstream Authentication
	AuthType     string `gorm:"type:varchar(20);default:'none';index:idx_url_configs_auth_type" json:"auth_type"` // none, basic, apikey, bearer
	AuthUsername string `gorm:"type:varchar(255)" json:"auth_username,omitempty"`                                  // For basic
	AuthPassword string `gorm:"type:varchar(255)" json:"auth_password,omitempty"`                                  // For basic (TODO: Encrypt this)
	AuthToken    string `gorm:"type:text" json:"auth_token,omitempty"`                                             // For bearer
	AuthKey      string `gorm:"type:varchar(255)" json:"auth_key,omitempty"`                                       // For apikey
	AuthValue    string `gorm:"type:varchar(255)" json:"auth_value,omitempty"`                                     // For apikey
	AuthAddTo    string `gorm:"type:varchar(20);default:'header'" json:"auth_add_to,omitempty"`                    // header, query (for apikey)
}

type APIConfig struct {
	gorm.Model
	Path         string         `gorm:"type:varchar(255);not null;uniqueIndex:idx_api_configs_path_method;index:idx_api_configs_path_fast" json:"path"`
	Method       string         `gorm:"type:varchar(100);not null;uniqueIndex:idx_api_configs_path_method;index:idx_api_configs_method" json:"method"`
	URLConfigID  uint           `gorm:"not null;index:idx_api_configs_url_config_id" json:"url_config_id"`
	URI          string         `gorm:"type:varchar(500);index:idx_api_configs_uri" json:"uri"`
	Headers      datatypes.JSON `gorm:"type:jsonb;default:'{}'::jsonb" json:"headers"`
	QueryParams  datatypes.JSON `gorm:"type:jsonb;default:'{}'::jsonb" json:"query_params"`
	Body         datatypes.JSON `gorm:"type:jsonb" json:"body"`
	Variables    datatypes.JSON `gorm:"type:jsonb;default:'{}'::jsonb" json:"variables"`
	MaxRetries   int            `gorm:"default:1;check:max_retries >= 0 AND max_retries <= 10" json:"max_retries"`
	RetryDelay   int            `gorm:"default:1;check:retry_delay >= 0 AND retry_delay <= 300" json:"retry_delay"`
	Timeout      int            `gorm:"default:30;check:timeout >= 1 AND timeout <= 600" json:"timeout"`
	Manipulation string         `gorm:"type:varchar(1000)" json:"manipulation"`
	Description  string         `gorm:"type:varchar(500)" json:"description"`
	IsAdmin      bool           `gorm:"default:false;index:idx_api_configs_is_admin" json:"is_admin"`

	// Caching Settings
	CacheEnabled bool `gorm:"default:false" json:"cache_enabled"`
	CacheTTL     int  `gorm:"default:0" json:"cache_ttl"` // seconds, 0 = no cache

	// Rate Limiting
	RateLimitEnabled bool `gorm:"default:false" json:"rate_limit_enabled"`
	RateLimit        int  `gorm:"default:100" json:"rate_limit"`       // requests per window
	RateLimitWindow  int  `gorm:"default:60" json:"rate_limit_window"` // seconds

	// Priority (for load balancing)
	Priority int `gorm:"default:0" json:"priority"` // higher = more priority

	// Authentication Configuration
	// AuthType: none = no auth, jwt = JWT token, basic = Basic Auth, apikey = API Key, gateway = Gateway admin auth
	AuthType         string `gorm:"type:varchar(20);default:'none';index:idx_api_configs_auth_type" json:"auth_type"`
	AuthRequired     bool   `gorm:"default:false" json:"auth_required"`
	AuthGRPCConfigID *uint  `gorm:"index:idx_api_configs_auth_grpc" json:"auth_grpc_config_id,omitempty"` // Reference to URLConfig for auth gRPC

	// JWT Configuration (when auth_type = 'jwt')
	JWTSecretKey  string `gorm:"type:varchar(500)" json:"jwt_secret_key,omitempty"`
	JWTIssuer     string `gorm:"type:varchar(255)" json:"jwt_issuer,omitempty"`
	JWTAudience   string `gorm:"type:varchar(255)" json:"jwt_audience,omitempty"`
	JWTAlgorithm  string `gorm:"type:varchar(20);default:'HS256'" json:"jwt_algorithm,omitempty"`
	JWTExpiration int    `gorm:"default:3600" json:"jwt_expiration,omitempty"` // seconds

	// Basic Auth Configuration (when auth_type = 'basic')
	BasicAuthUsers datatypes.JSON `gorm:"type:jsonb" json:"basic_auth_users,omitempty"` // [{username, password_hash}]

	// API Key Configuration (when auth_type = 'apikey')
	APIKeyHeader   string         `gorm:"type:varchar(100);default:'X-API-Key'" json:"api_key_header,omitempty"`
	APIKeyLocation string         `gorm:"type:varchar(20);default:'header'" json:"api_key_location,omitempty"` // header, query
	APIKeys        datatypes.JSON `gorm:"type:jsonb" json:"api_keys,omitempty"`                                // [{"key": "xxx", "name": "App1", "active": true}]

	// Relations
	URLConfig      URLConfig  `gorm:"foreignKey:URLConfigID;constraint:OnDelete:RESTRICT" json:"url_config"`
	AuthGRPCConfig *URLConfig `gorm:"foreignKey:AuthGRPCConfigID;constraint:OnDelete:SET NULL" json:"auth_grpc_config,omitempty"`
}

type APIGroup struct {
	gorm.Model
	Slug           string         `gorm:"type:varchar(100);uniqueIndex;not null;index:idx_api_groups_slug_fast" json:"slug" binding:"required"`
	Name           string         `gorm:"type:varchar(255);index:idx_api_groups_name" json:"name"`
	Description    string         `gorm:"type:varchar(500)" json:"description"`
	IsAdmin        bool           `gorm:"default:false;idx_api_groups_is_admin" json:"is_admin"`
	IsActive       bool           `gorm:"default:true;idx_api_groups_is_active" json:"is_active"`
	Steps          []APIGroupStep `gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE" json:"steps"`
	LastExecuted   *time.Time     `gorm:"index:idx_api_groups_last_executed" json:"last_executed,omitempty"`
	ExecutionCount int            `gorm:"default:0" json:"execution_count"`
}

type APIGroupStep struct {
	gorm.Model
	GroupID     uint           `json:"group_id" binding:"required"`
	APIConfigID uint           `json:"api_config_id" binding:"required"`
	OrderIndex  int            `gorm:"not null;index:idx_api_group_steps_order" json:"order_index" binding:"required"`
	Alias       string         `gorm:"type:varchar(100);not null;index:idx_api_group_steps_alias" json:"alias" binding:"required"`
	Description string         `gorm:"type:varchar(500)" json:"description"`
	Variables   datatypes.JSON `gorm:"type:jsonb;default:'{}'::jsonb" json:"variables"`
	IsEnabled   bool           `gorm:"default:true;index:idx_api_group_steps_enabled" json:"is_enabled"`
	Timeout     int            `gorm:"default:30;check:timeout >= 1 AND timeout <= 600" json:"timeout"`
	APIConfig   APIConfig      `gorm:"foreignKey:APIConfigID;constraint:OnDelete:CASCADE" json:"api_config"`
	APIGroup    APIGroup       `gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE" json:"api_group"`
}

type APIGroupCron struct {
	gorm.Model
	Slug         string     `gorm:"type:varchar(100);not null;index:idx_api_group_cron_slug" json:"slug"`
	Schedule     string     `gorm:"type:varchar(50);not null;check:schedule ~ '^(\\*|[0-5]?\\d) (\\*|[01]?\\d|2[0-3]) (\\*|[0-2]?\\d|3[01]) (\\*|[0-1]?\\d) (\\*|[0-6])$'" json:"schedule"` // cron format validation
	Timezone     string     `gorm:"type:varchar(50);default:'UTC';check:timezone IN ('UTC', 'Asia/Jakarta', 'Asia/Singapore', 'America/New_York', 'Europe/London')" json:"timezone"`
	Enabled      bool       `gorm:"default:true;index:idx_api_group_cron_enabled" json:"enabled"`
	LastRun      *time.Time `gorm:"index:idx_api_group_cron_last_run" json:"last_run,omitempty"`
	NextRun      *time.Time `gorm:"index:idx_api_group_cron_next_run" json:"next_run,omitempty"`
	IsRunning    bool       `gorm:"default:false;index:idx_api_group_cron_running" json:"is_running"`
	SuccessCount int        `gorm:"default:0" json:"success_count"`
	FailureCount int        `gorm:"default:0" json:"failure_count"`
	AvgDuration  int        `gorm:"default:0" json:"avg_duration"`
	APIGroup     APIGroup   `gorm:"foreignKey:Slug;references:Slug;constraint:OnDelete:CASCADE" json:"api_group"`
}
