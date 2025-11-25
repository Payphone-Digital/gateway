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
}

type APIConfig struct {
	gorm.Model
	Slug         string         `gorm:"type:varchar(100);uniqueIndex;not null;index:idx_api_configs_slug_fast"`
	Method       string         `gorm:"type:varchar(100);not null;index:idx_api_configs_method" json:"method"`
	URLConfigID  uint           `gorm:"not null;index:idx_api_configs_url_config_id" json:"url_config_id"`
	URI          string         `gorm:"type:varchar(500);index:idx_api_configs_uri" json:"uri"`
	Headers      datatypes.JSON `gorm:"type:jsonb;default:'{}'::jsonb"`
	QueryParams  datatypes.JSON `gorm:"type:jsonb;default:'{}'::jsonb"`
	Body         datatypes.JSON `gorm:"type:jsonb"`
	Variables    datatypes.JSON `gorm:"type:jsonb;default:'{}'::jsonb"`
	MaxRetries   int            `gorm:"default:1;check:max_retries >= 0 AND max_retries <= 10"`
	RetryDelay   int            `gorm:"default:1;check:retry_delay >= 0 AND retry_delay <= 300"`
	Timeout      int            `gorm:"default:30;check:timeout >= 1 AND timeout <= 600"`
	Manipulation string         `gorm:"type:varchar(1000)"`
	Description  string         `gorm:"type:varchar(500)"`
	IsAdmin      bool           `gorm:"default:false;index:idx_api_configs_is_admin"`
	URLConfig    URLConfig      `gorm:"foreignKey:URLConfigID;constraint:OnDelete:RESTRICT" json:"url_config"`
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
