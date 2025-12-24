package database

import (
	"github.com/Payphone-Digital/gateway/internal/model"
	"gorm.io/gorm"
)

// AutoMigrate runs database migrations for all models
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.URLConfig{},
		&model.APIConfig{},
		&model.APIGroup{},
		&model.APIGroupStep{},
		&model.APIGroupCron{},
	)
}
