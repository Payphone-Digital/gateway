package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	FirstName           string     `gorm:"column:first_name;not null"`
	LastName            string     `gorm:"column:last_name;not null"`
	Phone               string     `gorm:"column:phone"`
	Email               string     `gorm:"column:email;unique;not null"`
	Password            string     `gorm:"column:password;not null"`
	LastLogin           time.Time  `gorm:"column:last_login"`
	TokenVersion        int        `gorm:"column:token_version;default:1;not null"`
	RefreshTokenHash    string     `gorm:"column:refresh_token_hash;default:null;index:idx_users_refresh_token_hash,where:refresh_token_hash IS NOT NULL"`
	RefreshTokenExpires *time.Time `gorm:"column:refresh_token_expires_at;default:null;index:idx_users_token_cleanup,where:refresh_token_expires_at IS NOT NULL"`
}
